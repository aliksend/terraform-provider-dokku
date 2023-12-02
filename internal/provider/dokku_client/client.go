package dokkuclient

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/blang/semver"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/melbahja/goph"
)

func New(client *goph.Client, logSshCommands bool, uploadAppName string, uploadSplitBytes int) *Client {
	return &Client{
		client:         client,
		logSshCommands: logSshCommands,

		uploadAppName:    uploadAppName,
		uploadSplitBytes: uploadSplitBytes,
	}
}

type Client struct {
	client         *goph.Client
	logSshCommands bool

	uploadAppName    string
	uploadSplitBytes int

	dokkuVersion semver.Version
}

var mutex = &sync.Mutex{}

// Run runs any ssh command
//
// Deprecated: Use specific methods.
func (c *Client) Run(ctx context.Context, cmd string, sensitiveStrings ...string) (stdout string, status int, err error) {
	// disabling concurrent calls
	mutex.Lock()
	defer mutex.Unlock()

	cmdSafe := cmd
	for _, toReplace := range sensitiveStrings {
		cmdSafe = strings.Replace(cmdSafe, toReplace, "*******", -1)
	}

	if c.logSshCommands {
		tflog.Error(ctx, "SSH cmd", map[string]any{"cmd": cmdSafe})
	} else {
		tflog.Debug(ctx, "SSH cmd", map[string]any{"cmd": cmdSafe})
	}

	stdoutRaw, err := c.client.RunContext(ctx, "--quiet "+cmd)

	stdout = string(stdoutRaw)
	for _, toReplace := range sensitiveStrings {
		stdout = strings.Replace(stdout, toReplace, "*******", -1)
	}
	stdout = strings.TrimSuffix(stdout, "\n")

	if err != nil {
		status = parseStatusCode(err.Error())
		if c.logSshCommands {
			tflog.Error(ctx, "SSH error", map[string]any{"status": status, "stdout": stdout})
		} else {
			tflog.Debug(ctx, "SSH error", map[string]any{"status": status, "stdout": stdout})
		}
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

var (
	ErrInvalidUser = errors.New("must use a dokku user for authentication, see the docs")
)

func (c *Client) GetVersion(ctx context.Context) (rawVersion string, parsedVersion semver.Version, err error) {
	stdout, status, _ := c.Run(ctx, "version")

	// Check for 127 status code... suggests that we're not authenticating
	// with a dokku user (see https://github.com/aaronstillwell/terraform-provider-dokku/issues/1)
	if status == 127 {
		return "", semver.Version{}, ErrInvalidUser
	}

	re := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+`)
	found := re.FindString(stdout)

	parsedVersion, err = semver.Parse(found)
	if err != nil {
		return found, semver.Version{}, err
	}
	c.dokkuVersion = parsedVersion
	return found, parsedVersion, err
}
