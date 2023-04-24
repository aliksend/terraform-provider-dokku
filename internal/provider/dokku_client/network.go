package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) NetworkExists(ctx context.Context, name string) (bool, error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("network:exists %s", name))
	if err != nil {
		if strings.Contains(stdout, "Network does not exist") {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (c *Client) NetworksReport(ctx context.Context, name string) (networks map[string]string, err error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("network:report %s", name))
	if err != nil {
		return nil, err
	}

	networks = make(map[string]string)
	prefix := "Network "
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), ":")
		title := parts[0]
		if title[:len(prefix)] != prefix {
			continue
		}
		name := title[len(prefix):]
		value := strings.TrimSpace(parts[1])
		networks[name] = value
	}
	return
}

func (c *Client) NetworkCreate(ctx context.Context, name string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("network:create %s", name))
	return err
}

func (c *Client) NetworkGetNameForApp(ctx context.Context, appName string, networkType string) (string, error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("network:report %s --network-%s", appName, networkType))
	return stdout, err
}

func (c *Client) NetworkEnsureAndSetForApp(ctx context.Context, appName string, networkType string, name string) error {
	exists, err := c.NetworkExists(ctx, name)
	if err != nil {
		return err
	}
	if !exists {
		err = c.NetworkCreate(ctx, name)
		if err != nil {
			return err
		}
	}

	_, _, err = c.Run(ctx, fmt.Sprintf("network:set %s %s %s", appName, networkType, name))
	return err
}

func (c *Client) NetworkUnsetForApp(ctx context.Context, appName string, networkType string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("network:set %s %s", appName, networkType))
	return err
}
