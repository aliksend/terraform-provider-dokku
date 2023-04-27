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

func (c *Client) DeployFromImage(ctx context.Context, appName string, dockerImage string) error {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("git:from-image %s %s", appName, dockerImage))
	if err != nil {
		if strings.Contains(stdout, "No changes detected, skipping git commit") {
			return c.DeployRebuild(ctx, appName)
		}

		return err
	}
	return nil
}

func (c *Client) DeploySyncRepository(ctx context.Context, appName string, repositoryUrl string, build bool, ref string) error {
	buildStr := ""
	if build {
		buildStr = "--build"
	}
	_, _, err := c.Run(ctx, fmt.Sprintf("git:sync %s %s %s %s", buildStr, appName, repositoryUrl, ref))
	return err
}
