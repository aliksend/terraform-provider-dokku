package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) HttpAuthReport(ctx context.Context, appName string) (enabled bool, users []string, err error) {
	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("http-auth:report %s", appName))
	if err != nil {
		return false, nil, err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		key := strings.TrimSpace(parts[0])
		if key == "Http auth enabled" {
			enabled = strings.TrimSpace(parts[1]) == "true"
		} else if key == "Http auth users" {
			usersStr := strings.TrimSpace(parts[1])
			if usersStr != "" {
				users = strings.Split(usersStr, " ")
			}
		}
	}

	return
}

func (c *Client) HttpAuthDisable(ctx context.Context, appName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("http-auth:disable %s", appName))
	return err
}

func (c *Client) HttpAuthEnable(ctx context.Context, appName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("http-auth:enable %s", appName))
	return err
}

func (c *Client) HttpAuthAddUser(ctx context.Context, appName string, user string, password string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("http-auth:add-user %s %s %s", appName, user, password), password)
	return err
}

func (c *Client) HttpAuthRemoveUser(ctx context.Context, appName string, user string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("http-auth:remove-user %s %s", appName, user))
	return err
}
