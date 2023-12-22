package dokkuclient

import (
	"context"
	"fmt"
)

func (c *Client) ProcessRestart(ctx context.Context, appName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("ps:restart %s", appName))
	return err
}
