package dokkuclient

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/melbahja/goph"
)

func New(client *goph.Client) *Client {
	return &Client{
		client: client,
	}
}

type Client struct {
	client           *goph.Client
	sensitiveStrings []string
}

var mutex = &sync.Mutex{}

// Run runs any ssh command
//
// Deprecated: Use specific methods.
func (c *Client) Run(ctx context.Context, cmd string) (stdout string, status int, err error) {
	// disabling concurrent calls
	mutex.Lock()
	defer mutex.Unlock()

	cmdSafe := cmd
	for _, toReplace := range c.sensitiveStrings {
		cmdSafe = strings.Replace(cmdSafe, toReplace, "*******", -1)
	}

	tflog.Debug(ctx, "SSH cmd", map[string]any{"cmd": cmdSafe})

	stdoutRaw, err := c.client.RunContext(ctx, "--quiet "+cmd)

	stdout = string(stdoutRaw)
	for _, toReplace := range c.sensitiveStrings {
		stdout = strings.Replace(stdout, toReplace, "*******", -1)
	}
	stdout = strings.TrimSuffix(stdout, "\n")

	if err != nil {
		status = parseStatusCode(err.Error())
		tflog.Debug(ctx, "SSH error", map[string]any{"status": status, "stdout": stdout})
		err = fmt.Errorf("Error [%d]: %s", status, stdout)
	}
	return
}

func parseStatusCode(str string) int {
	re := regexp.MustCompile("^Process exited with status ([0-9]+)$")
	found := re.FindStringSubmatch(str)

	if found == nil {
		return 0
	}

	i, err := strconv.Atoi(found[1])

	if err != nil {
		return 0
	}

	return i
}
