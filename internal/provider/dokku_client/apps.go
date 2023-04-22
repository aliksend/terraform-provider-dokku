package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) AppCreate(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("apps:create %s", appName))
	return err
}

func (c *Client) AppExists(ctx context.Context, appName string) (bool, error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("apps:exists %s", appName))
	if err != nil {
		if strings.Contains(stdout, fmt.Sprintf("App %s does not exist", appName)) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *Client) AppDestroy(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("apps:destroy %s --force", appName))
	return err
}
