package dokkuclient

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

type ProxyPort struct {
	Scheme        string
	HostPort      string
	ContainerPort string
}

func (c *Client) ProxyPortsExport(ctx context.Context, appName string) (res []ProxyPort, err error) {
	stdout, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports %s", appName))
	if err != nil {
		if strings.Contains(stdout, "No port mappings configured for app") {
			return nil, nil
		}

		return nil, err
	}

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		parts := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
		scheme := strings.TrimSpace(parts[0])
		hostPortStr := strings.TrimSpace(parts[1])
		containerPortStr := strings.TrimSpace(parts[2])

		res = append(res, ProxyPort{
			Scheme:        scheme,
			HostPort:      hostPortStr,
			ContainerPort: containerPortStr,
		})
	}

	return
}

func (c *Client) ProxyDisable(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports-disable %s", appName))
	return err
}

func (c *Client) ProxyEnable(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports-enable %s", appName))
	return err
}

func (c *Client) ProxyPortRemove(ctx context.Context, appName string, hostPort int64) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports-remove %s %d", appName, hostPort))
	return err
}

func (c *Client) ProxyPortAdd(ctx context.Context, appName string, scheme string, hostPort int64, containerPort int64) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports-add %s %s:%d:%d", appName, scheme, hostPort, containerPort))
	return err
}

func (c *Client) ProxyPortsSet(ctx context.Context, appName string, proxyPorts []ProxyPort) error {
	proxyPortsStr := ""
	for _, p := range proxyPorts {
		proxyPortsStr = fmt.Sprintf("%s %s:%s:%s", proxyPortsStr, p.Scheme, p.HostPort, p.ContainerPort)
	}
	_, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports-set %s %s", appName, proxyPortsStr))
	return err
}

func (c *Client) ProxyPortsClear(ctx context.Context, appName string) error {
	_, _, err := c.Run(ctx, fmt.Sprintf("proxy:ports-clear %s", appName))
	return err
}
