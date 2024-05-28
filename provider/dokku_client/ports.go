package dokkuclient

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blang/semver"
)

type Port struct {
	Scheme        string
	HostPort      string
	ContainerPort string
}

var (
	ltThan31 = semver.MustParseRange("<0.31.0")
)

func (c *Client) portsCommand(name string) string {
	if ltThan31(c.dokkuVersion) {
		return "proxy:ports-" + name
	}

	return "ports:" + name
}

func (c *Client) PortsExport(ctx context.Context, appName string) (res []Port, err error) {
	var command string
	if ltThan31(c.dokkuVersion) {
		command = "proxy:ports"
	} else {
		command = "ports:list"
	}

	stdout, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s %s", command, appName))
	if err != nil {
		if strings.Contains(stdout, "No port mappings configured for app") {
			return nil, nil
		}

		return nil, err
	}

	prefix := "----->"
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > len(prefix) && trimmedLine[:len(prefix)] == prefix {
			continue
		}

		parts := regexp.MustCompile(`\s+`).Split(trimmedLine, -1)
		scheme := strings.TrimSpace(parts[0])
		hostPortStr := strings.TrimSpace(parts[1])
		containerPortStr := strings.TrimSpace(parts[2])

		res = append(res, Port{
			Scheme:        scheme,
			HostPort:      hostPortStr,
			ContainerPort: containerPortStr,
		})
	}

	return
}

func (c *Client) PortRemove(ctx context.Context, appName string, hostPort int64) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s %s %d", c.portsCommand("remove"), appName, hostPort))
	return err
}

func (c *Client) PortAdd(ctx context.Context, appName string, scheme string, hostPort int64, containerPort int64) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s %s %s:%d:%d", c.portsCommand("add"), appName, scheme, hostPort, containerPort))
	return err
}

func (c *Client) PortsSet(ctx context.Context, appName string, ports []Port) error {
	portsStr := ""
	for _, p := range ports {
		portsStr = fmt.Sprintf("%s %s:%s:%s", portsStr, p.Scheme, p.HostPort, p.ContainerPort)
	}
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s %s %s", c.portsCommand("set"), appName, portsStr))
	return err
}

func (c *Client) PortsClear(ctx context.Context, appName string) error {
	_, _, err := c.RunQuiet(ctx, fmt.Sprintf("%s %s", c.portsCommand("clear"), appName))
	return err
}
