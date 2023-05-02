package dokkuclient

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/melbahja/goph"
	"github.com/pkg/sftp"
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

func (c *Client) storageEnsure(ctx context.Context, name string) error {
	if name != "" && name[0] != '/' {
		_, _, err := c.Run(ctx, fmt.Sprintf("storage:ensure-directory %s", name))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) StorageEnsure(ctx context.Context, name string, localDirectory *string) error {
	err := c.storageEnsure(ctx, name)
	if err != nil {
		return fmt.Errorf("unable to ensure storage: %w", err)
	}

	if localDirectory != nil {
		if c.sftpClient == nil {
			return fmt.Errorf("SCP connection not configured")
		}

		err := storageSyncDirectories(ctx, c.sftpClient, *localDirectory, getPathToMount(name))
		if err != nil {
			return err
		}

		// Run ensure again to restore permissions
		err = c.storageEnsure(ctx, name)
		if err != nil {
			return fmt.Errorf("unable to ensure storage: %w", err)
		}
	}

	return nil
}

func storageUploadFile(sftp *sftp.Client, localFile string, remoteFile string) error {
	local, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("unable to open local file %s: %w", localFile, err)
	}
	defer local.Close()

	remote, err := sftp.Create(remoteFile)
	if err != nil {
		return fmt.Errorf("unable to create remote file %s: %w", remoteFile, err)
	}
	defer remote.Close()

	_, err = io.Copy(remote, local)
	if err != nil {
		return fmt.Errorf("unable to copy local file to remote: %w", err)
	}

	return nil
}

func storageListLocalFilesAndDirs(ctx context.Context, rootDirectory string) (localFiles []string, localDirs []string, err error) {
	err = filepath.WalkDir(rootDirectory, func(path string, d fs.DirEntry, err error) error {
		relpath, e := filepath.Rel(rootDirectory, path)
		if e != nil {
			// Log error and skip file
			tflog.Error(ctx, "Unable to find relative path to file", map[string]any{"path": path, "basedir": rootDirectory})
			//nolint:nilerr
			return nil
		}

		if d.Type().IsRegular() {
			localFiles = append(localFiles, relpath)
		} else if d.Type().IsDir() {
			localDirs = append(localDirs, relpath)
		}
		return nil
	})
	return
}

func storageRemoveRemoteFilesAndDirectories(ctx context.Context, sftp *sftp.Client, rootDirectory string, filesToKeep []string, dirsToKeep []string) error {
	walker := sftp.Walk(rootDirectory)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			// fmt.Fprintln(os.Stderr, err)
			return fmt.Errorf("unable to get file: %w", err)
		}
		path := walker.Path()
		relpath, err := filepath.Rel(rootDirectory, path)
		if err != nil {
			// Log error and skip file
			tflog.Error(ctx, "Unable to find relative path to file", map[string]any{"path": path, "basedir": rootDirectory})
			continue
		}
		if relpath == "" {
			continue
		}

		if walker.Stat().IsDir() {
			found := false
			for _, localDir := range dirsToKeep {
				if localDir == relpath {
					found = true
					break
				}
			}
			if !found {
				tflog.Debug(ctx, "Removing dir from remote", map[string]any{"dirname": relpath, "fullpath": path})
				err := sftp.RemoveDirectory(path)
				if err != nil {
					return fmt.Errorf("unable to remove remote dir: %w", err)
				}
			}
		} else {
			found := false
			for _, localFile := range filesToKeep {
				if localFile == relpath {
					found = true
					break
				}
			}
			if !found {
				tflog.Debug(ctx, "Removing file from remote", map[string]any{"filename": relpath, "fullpath": path})
				err := sftp.Remove(path)
				if err != nil {
					return fmt.Errorf("unable to remove remote file: %w", err)
				}
			}
		}
	}

	return nil
}

func storageCreateDirsOnRemote(ctx context.Context, sftp *sftp.Client, rootDirectory string, dirsToCreate []string) error {
	for _, dirname := range dirsToCreate {
		remoteDir := filepath.Join(rootDirectory, dirname)
		err := sftp.MkdirAll(remoteDir)
		if err != nil {
			return fmt.Errorf("unable to make dir %s: %w", remoteDir, err)
		}
	}
	return nil
}

func storageUploadFilesToRemote(ctx context.Context, sftp *sftp.Client, localRootDirectory string, remoteRootDirectory string, filesToUpload []string) error {
	for _, filename := range filesToUpload {
		localFile := filepath.Join(localRootDirectory, filename)
		remoteFile := filepath.Join(remoteRootDirectory, filename)
		tflog.Debug(ctx, "Uploading file to remote", map[string]any{"local_file": localFile, "remote_file": remoteFile})
		err := storageUploadFile(sftp, localFile, remoteFile)
		if err != nil {
			return fmt.Errorf("unable to upload file %s: %w", localFile, err)
		}
	}

	return nil
}

func storageSyncDirectories(ctx context.Context, sftpClient *goph.Client, localDirectory string, remoteDirectory string) error {
	tflog.Debug(ctx, "Uploading local directory to remote", map[string]any{"local_directory": localDirectory, "remote_directory": remoteDirectory})

	sftp, err := sftpClient.NewSftp()
	if err != nil {
		return fmt.Errorf("unable to setup sftp: %w", err)
	}
	defer sftp.Close()

	localFiles, localDirs, err := storageListLocalFilesAndDirs(ctx, localDirectory)
	if err != nil {
		return fmt.Errorf("unable to walk local directory: %w", err)
	}

	tflog.Debug(ctx, "Local directory content", map[string]any{"local_files": localFiles, "local_dirs": localDirs})

	// To make modifications in remote dir we need to change it's owner to current user because it has no write permissions for anyone except owner.
	// So we need to have ID of current user and then run chown with sudo to change owner.
	// After all storage:ensure-directory will be called to fix permissions.
	rawuid, err := sftpClient.RunContext(ctx, "id -u")
	if err != nil {
		return fmt.Errorf("unable to run id -u")
	}
	uid := strings.TrimSpace(string(rawuid))

	_, err = sftpClient.RunContext(ctx, fmt.Sprintf("sudo chown -R '%s:%s' %q", uid, uid, remoteDirectory))
	if err != nil {
		return fmt.Errorf("unable to chown dir")
	}

	err = storageRemoveRemoteFilesAndDirectories(ctx, sftp, remoteDirectory, localFiles, localDirs)
	if err != nil {
		return err
	}

	err = storageCreateDirsOnRemote(ctx, sftp, remoteDirectory, localDirs)
	if err != nil {
		return err
	}

	err = storageUploadFilesToRemote(ctx, sftp, localDirectory, remoteDirectory, localFiles)
	if err != nil {
		return err
	}

	return nil
}
