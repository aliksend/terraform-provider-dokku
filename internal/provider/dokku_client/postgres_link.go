package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) PostgresLinkExists(ctx context.Context, serviceName string, appName string) (bool, error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("postgres:linked %s %s", serviceName, appName))
	if err != nil {
		if strings.Contains(stdout, fmt.Sprintf("Service %s is not linked to %s", serviceName, appName)) {
			return false, nil
		}
		if strings.Contains(stdout, fmt.Sprintf("App %s does not exist", appName)) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *Client) PostgresLinkCreate(ctx context.Context, serviceName string, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("postgres:link %s %s", serviceName, appName))
	return err
}

func (c *Client) PostgresLinkRemove(ctx context.Context, serviceName string, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("postgres:unlink %s %s", serviceName, appName))
	return err
}
