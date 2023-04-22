package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

const hostStoragePrefix = "/var/lib/dokku/data/storage/"

func (c *Client) StorageExists(ctx context.Context, appName string, name string) (exists bool, mountPath string, err error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("storage:list %s", appName))
	if err != nil {
		return false, "", err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ":")
		hostpath := strings.TrimSpace(parts[0])
		if hostpath[:len(hostStoragePrefix)] != hostStoragePrefix {
			continue
		}
		if name != hostpath[len(hostStoragePrefix):] {
			continue
		}
		return true, parts[1], nil
	}

	return false, "", nil
}

func (c *Client) StorageEnsure(ctx context.Context, name string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("storage:ensure-directory %s", name))
	return err
}

func (c *Client) StorageMount(ctx context.Context, appName string, name string, mountPath string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("storage:mount %s %s:%s", appName, hostStoragePrefix+name, mountPath))
	return err
}

func (c *Client) StorageUnmount(ctx context.Context, appName string, name string, mountPath string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("storage:unmount %s %s:%s", appName, hostStoragePrefix+name, mountPath))
	return err
}
