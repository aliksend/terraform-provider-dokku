package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) SimpleServiceLinkExists(ctx context.Context, servicePluginName string, serviceName string, appName string) (bool, error) {
	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s:linked %s %s", servicePluginName, serviceName, appName))
	if err != nil {
		if strings.Contains(stdout, fmt.Sprintf("Service %s (%s) is not linked to %s", serviceName, servicePluginName, appName)) {
			return false, nil
		}
		if strings.Contains(stdout, fmt.Sprintf("App %s does not exist", appName)) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *Client) SimpleServiceLinkCreate(ctx context.Context, servicePluginName string, serviceName string, appName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s:link %s %s", servicePluginName, serviceName, appName))
	return err
}

func (c *Client) SimpleServiceLinkRemove(ctx context.Context, servicePluginName string, serviceName string, appName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s:unlink %s %s", servicePluginName, serviceName, appName))
	return err
}
