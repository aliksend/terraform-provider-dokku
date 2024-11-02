package dokkuclient

import (
	"context"
	"fmt"
)

func (c *Client) NginxConfigGetValue(ctx context.Context, appName string, property string) (string, error) {
	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("nginx:report %s --nginx-%s", appName, property))
	return stdout, err
}

func (c *Client) NginxConfigSetValue(ctx context.Context, appName string, property string, value string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf(`nginx:set %s %s "%s"`, appName, property, value))
	return err
}

func (c *Client) NginxConfigResetValue(ctx context.Context, appName string, property string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("nginx:set %s %s", appName, property))
	return err
}
