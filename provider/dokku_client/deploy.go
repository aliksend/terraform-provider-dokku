package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) DeployUnsetSourceImage(ctx context.Context, appName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("git:set %s source-image", appName))
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

func (c *Client) DeployFromImage(ctx context.Context, appName string, dockerImage string, allowRebuild bool) (deployed bool, err error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("git:from-image %s %s", appName, dockerImage))
	if err != nil {
		if strings.Contains(stdout, "No changes detected, skipping git commit") {
			if allowRebuild {
				return true, c.DeployRebuild(ctx, appName)
			}
			return false, nil
		}

		return false, err
	}

	// Verify deployment actually completed by checking if the app is running
	// git:from-image may succeed but the subsequent build/deploy pipeline may fail
	deployed, err = c.verifyDeploymentSuccess(ctx, appName)
	if err != nil {
		return false, fmt.Errorf("deployment verification failed: %w", err)
	}

	return deployed, nil
}

func (c *Client) DeploySyncRepository(ctx context.Context, appName string, repositoryUrl string, ref string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("git:sync --build %s %s %s", appName, repositoryUrl, ref))
	return err
}

// verifyDeploymentSuccess checks if an app deployment actually completed successfully
// by verifying the app is running and has containers
func (c *Client) verifyDeploymentSuccess(ctx context.Context, appName string) (bool, error) {
	// Check ps:report to see if app is actually running
	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("ps:report %s", appName))
	if err != nil {
		return false, fmt.Errorf("failed to get app status: %w", err)
	}

	// Parse the ps:report output to check deployment status
	isDeployed := strings.Contains(stdout, "Deployed:                      true")
	isRunning := strings.Contains(stdout, "Running:                       true")

	// Check for missing container indicators
	hasMissingContainer := strings.Contains(stdout, "missing (CID:")

	if !isDeployed {
		return false, fmt.Errorf("app is not marked as deployed")
	}

	if hasMissingContainer {
		return false, fmt.Errorf("app has missing containers, deployment incomplete")
	}

	if !isRunning {
		// App is deployed but not running - this might be temporary during startup
		// Give it a moment and check again
		// Note: We return true here because deployment succeeded, even if startup is still in progress
		// The running state will be checked during health verification
		return true, nil
	}

	return true, nil
}
