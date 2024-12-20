package dokkuclient

import (
	"context"
	"fmt"
)

func (c *Client) ProxyDisable(ctx context.Context, appName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("proxy:disable %s", appName))
	return err
}

func (c *Client) ProxyEnable(ctx context.Context, appName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("proxy:enable %s", appName))
	return err
}

func (c *Client) ProxyBuildConfig(ctx context.Context, appName string) error {
	if appName == "--global" {
		appName = "--all"
	}
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("proxy:build-config %s", appName))
	return err
}
