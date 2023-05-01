package dokkuclient

import (
	"context"
	"fmt"
	"strings"
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

func (c *Client) StorageEnsureAndMount(ctx context.Context, appName string, name string, mountPath string) error {
	if name != "" && name[0] != '/' {
		_, _, err := c.Run(ctx, fmt.Sprintf("storage:ensure-directory %s", name))
		if err != nil {
			return err
		}
	}
	_, _, err := c.Run(ctx, fmt.Sprintf("storage:mount %s %s:%s", appName, getPathToMount(name), mountPath))
	return err
}
