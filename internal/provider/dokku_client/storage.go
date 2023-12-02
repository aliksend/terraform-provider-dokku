package dokkuclient

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/crypto/ssh"
)

const hostStoragePrefix = "/var/lib/dokku/data/storage/"

func (c *Client) StorageExport(ctx context.Context, appName string) (res map[string]string, err error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("storage:list %s", appName))
	if err != nil {
		return nil, err
	}

	res = make(map[string]string)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ":")
		hostpath := strings.TrimSpace(parts[0])
		if len(hostpath) > len(hostStoragePrefix) && hostpath[:len(hostStoragePrefix)] == hostStoragePrefix {
			res[hostpath[len(hostStoragePrefix):]] = parts[1]
		} else {
			res[hostpath] = parts[1]
		}
	}
	if len(res) == 0 {
		res = nil
	}
	return
}

func getPathToMount(name string) string {
	if name == "" {
		return ""
	}
	if name[0] == '/' {
		return name
	}
	return hostStoragePrefix + name
}

func (c *Client) StorageMount(ctx context.Context, appName string, name string, mountPath string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("storage:mount %s %s:%s", appName, getPathToMount(name), mountPath))
	return err
}

func (c *Client) StorageUnmount(ctx context.Context, appName string, name string, mountPath string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("storage:unmount %s %s:%s", appName, getPathToMount(name), mountPath))
	return err
}

func (c *Client) storageEnsureDirectory(ctx context.Context, name string) error {
	if name != "" && name[0] != '/' {
		_, _, err := c.Run(ctx, fmt.Sprintf("storage:ensure-directory %s", name))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) StorageEnsure(ctx context.Context, name string, localDirectory *string) error {
	err := c.storageEnsureDirectory(ctx, name)
	if err != nil {
		return fmt.Errorf("unable to ensure storage: %w", err)
	}

	if localDirectory != nil {
		err := c.storageSyncDirectories(ctx, name, *localDirectory, getPathToMount(name))
		if err != nil {
			return err
		}

		// Run ensure again to restore permissions
		err = c.storageEnsureDirectory(ctx, name)
		if err != nil {
			return fmt.Errorf("unable to ensure storage: %w", err)
		}
	}

	return nil
}

func (c *Client) makeTarArchiveForDirectory(ctx context.Context, localDirectory string, writer io.Writer) error {
	if _, err := os.Stat(localDirectory); os.IsNotExist(err) {
		return fmt.Errorf("Directory %s does not exist", localDirectory)
	} else if err != nil {
		return fmt.Errorf("Error checking directory %s: %s", localDirectory, err)
	}

	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	err := filepath.Walk(localDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			tflog.Error(ctx, "Error walking directory:"+err.Error())
			return err
		}

		// Create a tar header for the file
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			tflog.Error(ctx, "Error creating tar header:"+err.Error())
			return err
		}

		// Modify the header name to be relative to the source directory
		relPath, _ := filepath.Rel(localDirectory, path)
		header.Name = relPath

		// Write the header to the tar archive
		if err := tarWriter.WriteHeader(header); err != nil {
			tflog.Error(ctx, "Error writing tar header:"+err.Error())
			return err
		}

		// If the file is not a directory, copy its contents to the tar archive
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				tflog.Error(ctx, "Error opening file:"+err.Error())
				return err
			}
			defer file.Close()

			// Copy the file data to the tar archive
			_, err = io.Copy(tarWriter, file)
			if err != nil {
				tflog.Error(ctx, "Error copying file to tar archive:"+err.Error())
				return err
			}
		}

		return nil
	})

	return err
}

// / dokku apps:create <APP_NAME>
// / dokku checks:disable <APP_NAME>
// / dokku config:set <APP_NAME> DOKKU_DOCKERFILE_START_CMD='sleep infinity'
// / dokku storage:mount <APP_NAME> <REMOTE_DIRECTORY>:/mnt
// / dokku git:from-image <APP_NAME> busybox
// / dokku enter <APP_NAME> web sh
// /     ## in pseudo-tty
// /     # remove old tmp archive (if present)
// /     rm -f /mnt/tmp.tar.base64
// /     # multiple echo commands with base64-encoded tar archive
// /     echo -n '...' >> /mnt/tmp.tar.base64
// /     # untar archive
// /     cat /mnt/tmp.tar.base64 | base64 -d | tar x -C /mnt
// /     # remove tmp archive
// /     rm -f /mnt/tmp.tar.base64
// /     exit
// / dokku apps:destroy --force <APP_NAME>
func (c *Client) storageSyncDirectories(ctx context.Context, storageName string, localDirectory string, remoteDirectory string) error {
	tflog.Debug(ctx, "Uploading local directory to remote", map[string]any{"local_directory": localDirectory, "remote_directory": remoteDirectory})

	appName := c.uploadAppName
	err := c.AppCreate(ctx, appName)
	if err != nil {
		return fmt.Errorf("unable to create app: %w", err)
	}

	defer func() {
		_ = c.AppDestroy(ctx, appName)
	}()

	err = c.ChecksSet(ctx, appName, "disabled")
	if err != nil {
		return fmt.Errorf("unable to disable checks: %w", err)
	}

	err = c.ConfigSet(ctx, appName, map[string]string{
		"DOKKU_DOCKERFILE_START_CMD": "sleep infinity",
	})
	if err != nil {
		return fmt.Errorf("unable to set config: %w", err)
	}

	err = c.StorageMount(ctx, appName, storageName, "/mnt")
	if err != nil {
		return fmt.Errorf("unable to mount storage: %w", err)
	}

	deployed, err := c.DeployFromImage(ctx, appName, "busybox", false)
	if err != nil {
		return fmt.Errorf("unable to deploy sync app: %w", err)
	}
	if !deployed {
		return fmt.Errorf("sync app wasn't deployed")
	}

	// _, _, err = c.Run(ctx, fmt.Sprintf("run %s find /mnt -mindepth 1 -delete", appName))
	// if err != nil {
	// 	return fmt.Errorf("unable to clear mounted directory: %w", err)
	// }

	// -- copy tar archive to remote host
	return c.copyToRemoteHost(ctx, appName, localDirectory)
	// --
}

func (c *Client) copyToRemoteHost(ctx context.Context, appName string, localDirectory string) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("unable to open ssh session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("unable to setup stdin pipe: %w", err)
	}
	defer stdin.Close()

	var sessionStdoutCollector singleWriter
	session.Stdout = &sessionStdoutCollector
	session.Stderr = &sessionStdoutCollector

	if err := session.RequestPty("xterm", 40, 256, ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}); err != nil {
		return fmt.Errorf("request for pseudo terminal failed: %w", err)
	}

	err = session.Start(fmt.Sprintf("enter %s web sh", appName))
	if err != nil {
		return fmt.Errorf("unable to start copying files to remote directory: %w", err)
	}

	_, err = io.WriteString(stdin, "rm -f /mnt/tmp.tar.base64\n")
	if err != nil {
		return fmt.Errorf("unable to write string to file: %w", err)
	}

	pReader, pWriter := io.Pipe()

	go func() {
		defer pWriter.Close()
		base64encoder := base64.NewEncoder(base64.StdEncoding, pWriter)
		defer func() {
			if err := base64encoder.Close(); err != nil {
				log.Printf("[error] unable to close base64encoder: %v\n", err)
			}
		}()

		err := c.makeTarArchiveForDirectory(ctx, localDirectory, base64encoder)
		if err != nil {
			// return fmt.Errorf("unable to make tar archive: %w", err)
			log.Printf("[error] unable to make tar archive: %v\n", err)
		}
	}()

	scanner := bufio.NewScanner(pReader)
	buf := make([]byte, c.uploadSplitBytes)
	scanner.Buffer(buf, 0)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		lenData := len(data)
		// log.Printf("try %3d %s\n", lenData, data)
		if lenData < c.uploadSplitBytes && !atEOF {
			// log.Printf("  try again\n")
			return 0, nil, nil
		}

		// log.Printf("  split %d\n", lenData)
		if atEOF {
			err = bufio.ErrFinalToken
		}
		return lenData, data, err
	})
	for scanner.Scan() {
		// log.Printf("write %d\n", len(scanner.Text()))
		_, err = io.WriteString(stdin, fmt.Sprintf("echo -n '%s' >> /mnt/tmp.tar.base64\n", scanner.Text()))
		if err != nil {
			return fmt.Errorf("unable to write string to file: %w", err)
		}
	}

	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("unable to scan all: %w", err)
	}

	_, err = io.WriteString(stdin, "cat /mnt/tmp.tar.base64 | base64 -d | tar x -C /mnt\n")
	if err != nil {
		return fmt.Errorf("unable to extract tar archive: %w", err)
	}

	_, err = io.WriteString(stdin, "rm -f /mnt/tmp.tar.base64\n")
	if err != nil {
		return fmt.Errorf("unable to remove tmp archive: %w", err)
	}

	_, err = io.WriteString(stdin, "exit\n")
	if err != nil {
		return fmt.Errorf("unable to write string to file: %w", err)
	}

	err = stdin.Close()
	if err != nil {
		return fmt.Errorf("unable to close stdin: %w", err)
	}

	// stdout := sessionStdoutCollector.b.Bytes()
	// fmt.Println("--------------\n" + string(stdout) + "\n--------------")

	err = session.Wait()
	if err != nil {
		return fmt.Errorf("unable to copy: %w", err)
	}

	return nil
}

type singleWriter struct {
	b  bytes.Buffer
	mu sync.Mutex
}

func (w *singleWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}
