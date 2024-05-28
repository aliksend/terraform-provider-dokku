package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) ChecksSet(ctx context.Context, appName string, status string) error {
	var action string
	switch status {
	case "enabled":
		action = "enable"
	case "disabled":
		action = "disable"
	case "skipped":
		action = "skip"
	default:
		return fmt.Errorf("Invalid status value. Valid values are: enabled, disabled, skipped")
	}

	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("checks:%s %s", action, appName))
	return err
}

func (c *Client) ChecksGet(ctx context.Context, appName string) (status string, err error) {
	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("checks:report %s", appName))
	if err != nil {
		return "", err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		title := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch title {
		case "Checks disabled list":
			if value == "_all_" {
				return "disabled", nil
			}
		case "Checks skipped list":
			if value == "_all_" {
				return "skipped", nil
			}
		}
	}
	return "enabled", nil
}
