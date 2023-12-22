package dokkuclient

import (
	"context"
	"strings"
)

func (c *Client) PluginIsInstalled(ctx context.Context, pluginNameToFind string) (bool, error) {
	stdout, _, err := c.RunQuiet(ctx, "plugin:list")
	if err != nil {
		return false, err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), " ")
		pluginName := strings.TrimSpace(parts[0])
		if pluginNameToFind != pluginName {
			continue
		}
		return true, nil
	}
	return false, nil
}
