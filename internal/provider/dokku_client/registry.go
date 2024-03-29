package dokkuclient

import (
	"context"
	"fmt"
)

func (c *Client) RegistryLogin(ctx context.Context, host string, login string, password string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("registry:login %s %s %s", host, login, password), password)
	return err
}

func (c *Client) GitAuth(ctx context.Context, host string, login string, password string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("git:auth %s %s %s", host, login, password), password)
	return err
}
