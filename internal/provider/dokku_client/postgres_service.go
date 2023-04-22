package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) PostgresServiceExists(ctx context.Context, serviceName string) (bool, error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("postgres:exists %s", serviceName))
	if err != nil {
		if strings.Contains(stdout, fmt.Sprintf("Postgres service %s does not exist", serviceName)) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (c *Client) PostgresServiceDestroy(ctx context.Context, serviceName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("postgres:destroy %s --force", serviceName))
	return err
}

func (c *Client) PostgresServiceCreate(ctx context.Context, serviceName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("postgres:create %s", serviceName))
	return err
}
