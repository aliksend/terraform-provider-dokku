package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) SimpleServiceExists(ctx context.Context, servicePluginName string, serviceName string) (bool, error) {
	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s:exists %s", servicePluginName, serviceName))
	if err != nil {
		if strings.Contains(stdout, fmt.Sprintf("service %s does not exist", serviceName)) {
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

func (c *Client) SimpleServiceCreate(ctx context.Context, servicePluginName string, serviceName string, args ...string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s:create %s %s", servicePluginName, serviceName, strings.Join(args, " ")))
	return err
}

func (c *Client) SimpleServiceInfo(ctx context.Context, servicePluginName string, serviceName string) (map[string]string, error) {
	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s:info %s", servicePluginName, serviceName))
	if err != nil {
		return nil, err
	}

	res := make(map[string]string)
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		index := strings.Index(line, ":")
		if index == -1 {
			continue
		}
		key := strings.TrimSpace(line[:index])
		value := strings.TrimSpace(line[index+1:])
		res[key] = value
	}
	return res, nil
}
