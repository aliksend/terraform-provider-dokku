package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) DeployUnsetSourceImage(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("git:set %s source-image", appName))
	return err
}

func (c *Client) DeployFromArchive(ctx context.Context, appName string, archiveType string, archiveUrl string) error {
	if archiveType != "" {
		archiveType = fmt.Sprintf("--archive-type %s", archiveType)
	}
	_, _, err := c.Run(ctx, fmt.Sprintf("git:from-archive %s %s %s", archiveType, appName, archiveUrl))
	return err
}

func (c *Client) DeployRebuild(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("ps:rebuild %s", appName))
	return err
}

func (c *Client) DeployFromImage(ctx context.Context, appName string, dockerImage string, allowRebuild bool) error {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("git:from-image %s %s", appName, dockerImage))
	if err != nil {
		if strings.Contains(stdout, "No changes detected, skipping git commit") {
			if allowRebuild {
				return c.DeployRebuild(ctx, appName)
			}
			return nil
		}

		return err
	}
	return nil
}

func (c *Client) DeploySyncRepository(ctx context.Context, appName string, repositoryUrl string, ref string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("git:sync --build %s %s %s", appName, repositoryUrl, ref))
	return err
}
