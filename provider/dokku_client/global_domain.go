package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) GlobalDomainExists(ctx context.Context, domain string) (bool, error) {
	stdout, _, err := c.RunQuiet(ctx, "domains:report --global")
	if err != nil {
		return false, err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		key := strings.TrimSpace(parts[0])
		if key == "Domains global vhosts" {
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

func (c *Client) GlobalDomainAdd(ctx context.Context, domain string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("domains:add-global %s", domain))
	return err
}

func (c *Client) GlobalDomainRemove(ctx context.Context, domain string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("domains:remove-global %s", domain))
	return err
}
