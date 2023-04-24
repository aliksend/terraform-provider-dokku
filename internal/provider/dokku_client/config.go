package dokkuclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) ConfigExport(ctx context.Context, appName string) (res map[string]string, err error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("config:export --format=json %s", appName))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(stdout), &res)
	if err != nil {
		return nil, err
	}
	return
}

func (c *Client) ConfigSet(ctx context.Context, appName string, data map[string]string) error {
	dataStr := ""
	for k, v := range data {
		dataStr = fmt.Sprintf("%s %s=%q", dataStr, k, base64.StdEncoding.EncodeToString([]byte(v)))
	}
	_, _, err := c.Run(ctx, fmt.Sprintf("config:set --no-restart --encoded %s %s", appName, dataStr))
	return err
}

func (c *Client) ConfigUnset(ctx context.Context, appName string, names []string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("config:unset --no-restart %s %s", appName, strings.Join(names, " ")))
	return err
}
