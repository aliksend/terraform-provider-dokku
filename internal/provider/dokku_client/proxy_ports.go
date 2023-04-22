package dokkuclient

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (c *Client) ProxyPortExists(ctx context.Context, appName string, hostPort int64) (exists bool, scheme string, containerPort int64, err error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports %s", appName))
	if err != nil {
		return false, "", 0, err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
		scheme := strings.TrimSpace(parts[0])
		hostPortStr := strings.TrimSpace(parts[1])
		containerPortStr := strings.TrimSpace(parts[2])

		existingHostPort, err := strconv.ParseInt(hostPortStr, 10, 64)
		if err != nil {
			// resp.Diagnostics.AddAttributeError(path.Root("host_port"), "Invalid value", "Invalid value. "+err.Error())
			return false, "", 0, fmt.Errorf("Invalid host_port value: %w", err)
		}
		existingContainerPort, err := strconv.ParseInt(containerPortStr, 10, 64)
		if err != nil {
			// resp.Diagnostics.AddAttributeError(path.Root("container_port"), "Invalid value", "Invalid value. "+err.Error())
			return false, "", 0, fmt.Errorf("Invalid container_port value: %w", err)
		}

		if existingHostPort == hostPort {
			return true, scheme, existingContainerPort, nil
		}
	}

	return false, "", 0, nil
}

func (c *Client) ProxyPortRemove(ctx context.Context, appName string, hostPort int64) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports-remove %s %d", appName, hostPort))
	return err
}

func (c *Client) ProxyPortAdd(ctx context.Context, appName string, scheme string, hostPort int64, containerPort int64) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports-add %s %s:%d:%d", appName, scheme, hostPort, containerPort))
	return err
}
