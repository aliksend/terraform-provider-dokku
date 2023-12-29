package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) SimpleServiceExists(ctx context.Context, servicePluginName string, serviceName string) (bool, error) {
	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s:exists %s", servicePluginName, serviceName))
	if err != nil {
		if strings.Contains(stdout, fmt.Sprintf("%s service %s does not exist", servicePluginName, serviceName)) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (c *Client) SimpleServiceDestroy(ctx context.Context, servicePluginName string, serviceName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s:destroy %s --force", servicePluginName, serviceName))
	return err
}

func (c *Client) SimpleServiceCreate(ctx context.Context, servicePluginName string, serviceName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s:create %s", servicePluginName, serviceName))
	return err
}
