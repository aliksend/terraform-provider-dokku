package dokkuclient

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) DomainsExport(ctx context.Context, appName string) (res []string, err error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("domains:report %s", appName))
	if err != nil {
		return nil, err
	}

	enabled := false
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		key := strings.TrimSpace(parts[0])
		if key == "Domains app vhosts" {
			domainList := strings.TrimSpace(parts[1])
			if domainList != "" {
				res = append(res, strings.Split(domainList, " ")...)
			}
		} else if key == "Domains app enabled" {
			enabled = strings.TrimSpace(parts[1]) == "true"
		}
	}

	if !enabled {
		return nil, nil
	}

	return
}

func (c *Client) DomainAdd(ctx context.Context, appName string, domain string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("domains:add %s %s", appName, domain))
	return err
}

func (c *Client) DomainsSet(ctx context.Context, appName string, domains []string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("domains:set %s %s", appName, strings.Join(domains, " ")))
	return err
}

func (c *Client) DomainsClear(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("domains:clear %s", appName))
	return err
}

func (c *Client) DomainsDisable(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("domains:disable %s", appName))
	return err
}

func (c *Client) DomainsEnable(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("domains:enable %s", appName))
	return err
}

func (c *Client) DomainRemove(ctx context.Context, appName string, domain string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("domains:remove %s %s", appName, domain))
	return err
}
