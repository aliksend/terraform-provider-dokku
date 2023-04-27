package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) LetsencryptIsEnabled(ctx context.Context, appName string) (bool, error) {
	stdout, _, err := c.Run(ctx, "letsencrypt:list")
	if err != nil {
		return false, err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		existingApp := strings.TrimSpace(parts[0])
		if existingApp == appName {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) LetsencryptSetEmail(ctx context.Context, appName string, email string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("letsencrypt:set %s email %s", appName, email))
	return err
}

func (c *Client) LetsencryptEnable(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("letsencrypt:enable %s", appName))
	return err
}

func (c *Client) LetsencryptAddCronJob(ctx context.Context) error {
	_, _, err := c.Run(ctx, "letsencrypt:cron-job --add")
	return err
}

func (c *Client) LetsencryptDisable(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("letsencrypt:disable %s", appName))
	return err
}
