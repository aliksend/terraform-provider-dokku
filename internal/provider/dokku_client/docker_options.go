package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) DockerOptionExists(ctx context.Context, appName string, phase string, value string) (bool, error) {
	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("docker-options:report %s", appName))
	if err != nil {
		return false, err
	}

	expectedName := fmt.Sprintf("Docker options %s", phase)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		name := strings.TrimSpace(parts[0])
		if expectedName != name {
			continue
		}
		existingOptions := strings.TrimSpace(parts[1])
		if !strings.Contains(existingOptions, value) {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

func (c *Client) DockerOptionAdd(ctx context.Context, appName string, phases []string, value string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("docker-options:add %s %s %s", appName, strings.Join(phases, ","), value))
	return err
}

func (c *Client) DockerOptionRemove(ctx context.Context, appName string, phases []string, value string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("docker-options:remove %s %s %s", appName, strings.Join(phases, ","), value))
	return err
}
