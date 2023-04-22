package dokkuclient

import (
	"context"
	"encoding/base64"
	"fmt"
)

func (c *Client) ConfigGet(ctx context.Context, appName string, name string) (value string, err error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("config:get %s %s", appName, name))
	return stdout, err
}

func (c *Client) ConfigSet(ctx context.Context, appName string, name string, value string) error {
	// TODO --no-restart ?
	_, _, err := c.Run(ctx, fmt.Sprintf("config:set --encoded %s %s=%q", appName, name, base64.StdEncoding.EncodeToString([]byte(value))))
	return err
}

func (c *Client) ConfigUnset(ctx context.Context, appName string, name string) error {
	// TODO --no-restart ?
	_, _, err := c.Run(ctx, fmt.Sprintf("config:unset %s %s", appName, name))
	return err
}
