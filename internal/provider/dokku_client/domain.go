package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) DomainExists(ctx context.Context, appName string, domain string) (bool, error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("domains:report %s", appName))
	if err != nil {
		return false, err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		key := strings.TrimSpace(parts[0])
		if key == "Domains app vhosts" {
			domainList := strings.TrimSpace(parts[1])
			if domainList != "" {
				for _, existingDomain := range strings.Split(domainList, " ") {
					if existingDomain == domain {
						return true, nil
					}
				}
			}
			break
		}
	}
	return false, nil
}

func (c *Client) DomainAdd(ctx context.Context, appName string, domain string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("domains:add %s %s", appName, domain))
	return err
}

func (c *Client) DomainRemove(ctx context.Context, appName string, domain string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("domains:remove %s %s", appName, domain))
	return err
}
