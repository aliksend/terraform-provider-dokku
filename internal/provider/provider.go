package provider

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/blang/semver"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/melbahja/goph"
	"golang.org/x/crypto/ssh"
)

// TODO Добавить примеры в папку с примерамиx. Дописать Description-ы для аттрибутов

// Ensure dokkuProvider satisfies various provider interfaces.
var _ provider.Provider = &dokkuProvider{}

func New() provider.Provider {
	return &dokkuProvider{}
}

// dokkuProvider defines the provider implementation.
type dokkuProvider struct{}

// dokkuProviderModel describes the provider data model.
type dokkuProviderModel struct {
	SshHost               types.String `tfsdk:"ssh_host"`
	SshPort               types.Int64  `tfsdk:"ssh_port"`
	SshUser               types.String `tfsdk:"ssh_user"`
	SshCert               types.String `tfsdk:"ssh_cert"`
	ScpUser               types.String `tfsdk:"scp_user"`
	ScpCert               types.String `tfsdk:"scp_cert"`
	FailOnUntestedVersion types.Bool   `tfsdk:"fail_on_untested_version"`
	LogSshCommands        types.Bool   `tfsdk:"log_ssh_commands"`
}

func (p *dokkuProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "dokku"
}

func (p *dokkuProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with dokku",
		Attributes: map[string]schema.Attribute{
			"ssh_host": schema.StringAttribute{
				Required:    true,
				Description: "Host to connect to",
			},
			"ssh_port": schema.Int64Attribute{
				Optional:    true,
				Description: "Port to connect to. Default: 22",
			},
			"ssh_user": schema.StringAttribute{
				Optional:    true,
				Description: "Username to use. Default: dokku",
			},
			"ssh_cert": schema.StringAttribute{
				Optional: true,
				Description: strings.Join([]string{
					"Certificate to use. Supported formats:",
					"- file:/a or /a or ./a or ~/a - use provided value as path to certificate file",
					"- env:ABCD or $ABCD - use env var ABCD",
					"- raw:----.. or ----... - use provided value as raw certificate",
					"Default: ~/.ssh/id_rsa",
				}, "\n"),
			},
			"scp_user": schema.StringAttribute{
				Optional:    true,
				Description: "Username that will be used to copy files to server. If not set then scp feature will be disabled",
			},
			"scp_cert": schema.StringAttribute{
				Optional:    true,
				Description: "Cert for username that will be used to copy files to server. Default: ssh_cert value",
			},
			"fail_on_untested_version": schema.BoolAttribute{
				Optional: true,
			},
			"log_ssh_commands": schema.BoolAttribute{
				Optional:    true,
				Description: "Print SSH commands with ERROR verbose",
			},
		},
	}
}

func (p *dokkuProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config dokkuProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.SshHost.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("ssh_host"),
			"Unknown SSH host",
			"Unknown SSH host",
		)
	}
	if config.SshPort.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("ssh_port"),
			"Unknown SSH port",
			"Unknown SSH port",
		)
	}
	if config.SshUser.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("ssh_user"),
			"Unknown SSH user",
			"Unknown SSH user",
		)
	}
	if config.SshCert.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("ssh_cert"),
			"Unknown SSH cert",
			"Unknown SSH cert",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := ""
	port := uint(22)
	sshUsername := "dokku"
	sshCertPath := "~/.ssh/id_rsa"
	scpUsername := ""
	scpCertPath := ""
	failOnUntestedVersion := true
	logSshCommands := false

	if !config.SshHost.IsNull() {
		host = config.SshHost.ValueString()
	}
	if !config.SshPort.IsNull() {
		port = uint(config.SshPort.ValueInt64())
	}
	if !config.SshUser.IsNull() {
		sshUsername = config.SshUser.ValueString()
	}
	if !config.SshCert.IsNull() {
		var err error
		sshCertPath, err = getCertFilename(ctx, config.SshCert.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("ssh_cert"), "Unable to read cert", "Unable to read cert. "+err.Error())
			return
		}
	}
	if !config.ScpUser.IsNull() {
		scpUsername = config.ScpUser.ValueString()
	}
	if !config.ScpCert.IsNull() {
		var err error
		scpCertPath, err = getCertFilename(ctx, config.ScpCert.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("scp_cert"), "Unable to read cert", "Unable to read cert. "+err.Error())
			return
		}
	}
	if !config.FailOnUntestedVersion.IsNull() {
		failOnUntestedVersion = config.FailOnUntestedVersion.ValueBool()
	}
	if !config.LogSshCommands.IsNull() {
		logSshCommands = config.LogSshCommands.ValueBool()
	}

	usr, err := user.Current()
	if err == nil {
		_ = os.MkdirAll(filepath.Join(usr.HomeDir, ".ssh"), os.ModePerm)
	}

	sshCertPath, err = resolveHomeDir(sshCertPath)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("ssh_cert"), "Unable to get SSH cert", "Unable to get SSH cert. "+err.Error())
	}
	if scpCertPath != "" {
		scpCertPath, err = resolveHomeDir(scpCertPath)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("ssh_cert"), "Unable to get SSH cert for scp", "Unable to get SSH cert for scp. "+err.Error())
		}
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if host == "" {
		resp.Diagnostics.AddAttributeError(path.Root("ssh_host"), "Missing SSH host", "Missing SSH host")
	}
	if port == 0 {
		resp.Diagnostics.AddAttributeError(path.Root("ssh_port"), "Missing SSH port", "Missing SSH port")
	}
	if sshUsername == "" {
		resp.Diagnostics.AddAttributeError(path.Root("ssh_user"), "Missing SSH user", "Missing SSH user")
	}
	if sshCertPath == "" {
		resp.Diagnostics.AddAttributeError(path.Root("ssh_cert"), "Missing SSH cert", "Missing SSH cert")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "cert", map[string]any{"path": sshCertPath})

	sshAuth, err := goph.Key(sshCertPath, "")
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("ssh_cert"), "Unable to find cert for ssh", "Unable to find cert for ssh. "+err.Error())
		return
	}

	scpAuth := sshAuth
	if scpCertPath != "" && scpCertPath != sshCertPath {
		scpAuth, err = goph.Key(scpCertPath, "")
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("scp_cert"), "Unable to find cert for scp", "Unable to find cert for scp. "+err.Error())
			return
		}

	}

	tflog.Debug(ctx, "ssh connection", map[string]any{"host": host, "port": port, "user": sshUsername})

	sshConfig := &goph.Config{
		Auth:     sshAuth,
		Addr:     host,
		Port:     port,
		User:     sshUsername,
		Callback: verifyHost,
	}

	client, err := goph.NewConn(sshConfig)
	if err != nil {
		resp.Diagnostics.AddError("Unable to establish SSH connection", "Unable to establish SSH connection. "+err.Error())
		return
	}

	var sftpClient *goph.Client
	if scpUsername != "" {
		tflog.Debug(ctx, "scp connection", map[string]any{"host": host, "port": port, "user": scpUsername})

		scpConfig := &goph.Config{
			Auth:     scpAuth,
			Addr:     host,
			Port:     port,
			User:     scpUsername,
			Callback: verifyHost,
		}

		sftpClient, err = goph.NewConn(scpConfig)
		if err != nil {
			resp.Diagnostics.AddError("Unable to establish SSH connection for SCP", "Unable to establish SSH connection for SCP. "+err.Error())
			return
		}
	}

	dokkuClient := dokkuclient.New(client, sftpClient, logSshCommands)
	stdout, status, _ := dokkuClient.Run(ctx, "version")

	// Check for 127 status code... suggests that we're not authenticating
	// with a dokku user (see https://github.com/aaronstillwell/terraform-provider-dokku/issues/1)
	if status == 127 {
		resp.Diagnostics.AddError("must use a dokku user for authentication, see the docs", "must use a dokku user for authentication, see the docs")
		return
	}

	re := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+`)
	found := re.FindString(stdout)

	hostVersion, err := semver.Parse(found)

	testedVersions := ">=0.24.0 <=0.30.3"
	testedErrMsg := fmt.Sprintf("This provider has not been tested against Dokku version %s. Tested version range: %s", found, testedVersions)

	if err == nil {
		tflog.Debug(ctx, "host version", map[string]any{"version": hostVersion})

		compat, _ := semver.ParseRange(testedVersions)

		if !compat(hostVersion) {
			tflog.Debug(ctx, "fail_on_untested_version", map[string]any{"value": failOnUntestedVersion})

			if failOnUntestedVersion {
				resp.Diagnostics.AddError(testedErrMsg, testedErrMsg)
				return
			}
			resp.Diagnostics.AddWarning(testedErrMsg, testedErrMsg)
		}
	} else {
		resp.Diagnostics.AddError("Unable to detect dokku version", "Unable to detect dokku version. "+err.Error())
		return
	}

	tflog.Debug(ctx, "Connected!")

	// Make the dokku client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = dokkuClient
	resp.ResourceData = dokkuClient
}

func (p *dokkuProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAppResource,
		NewDomainResource,
		NewHttpAuthResource,
		NewLetsencryptResource,
		NewPluginResource,
		NewPostgresLinkResource,
		NewPostgresResource,
	}
}

func (p *dokkuProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

func verifyHost(host string, remote net.Addr, key ssh.PublicKey) error {
	// If you want to connect to new hosts.
	// here your should check new connections public keys
	// if the key not trusted you shuld return an error

	// hostFound: is host in known hosts file.
	// err: error if key not in known hosts file OR host in known hosts file but key changed!
	hostFound, err := goph.CheckKnownHost(host, remote, key, "")

	// Host in known hosts but key mismatch!
	// Maybe because of MAN IN THE MIDDLE ATTACK!
	if hostFound && err != nil {

		return err
	}

	// handshake because public key already exists.
	if hostFound && err == nil {

		return nil
	}

	// // Ask user to check if he trust the host public key.
	// if askIsHostTrusted(host, key) == false {

	// 	// Make sure to return error on non trusted keys.
	// 	return errors.New("you typed no, aborted!")
	// }

	// Add the new host to known hosts file.
	return goph.AddKnownHost(host, remote, key, "")
}

func tmpFileWithValue(value string) (string, error) {
	file, err := ioutil.TempFile("", "ssh_cert")
	if err != nil {
		return "", err
	}
	err = ioutil.WriteFile(file.Name(), []byte(value), os.ModePerm)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

func getCertFilename(ctx context.Context, value string) (certPath string, err error) {
	parts := strings.Split(value, ":")
	switch parts[0] {
	case "file":
		certPath = parts[1]
	case "env":
		certPath, err = tmpFileWithValue(os.Getenv(parts[1]))
		if err != nil {
			return "", fmt.Errorf("Unable to create temp file: %w", err)
		}
		tflog.Debug(ctx, "Save ssh_cert from env var to tmp file", map[string]any{"certPath": certPath})
	case "raw":
		var err error
		certPath, err = tmpFileWithValue(parts[1])
		if err != nil {
			return "", fmt.Errorf("Unable to create temp file: %w", err)
		}
		tflog.Debug(ctx, "Save ssh_cert from raw string to tmp file", map[string]any{"certPath": certPath})
	default:
		if value[0] == '~' || value[1] == '/' || value[1] == '.' {
			certPath = value
		} else if value[0] == '$' {
			var err error
			certPath, err = tmpFileWithValue(os.Getenv(value[1:]))
			if err != nil {
				return "", fmt.Errorf("Unable to create temp file: %w", err)
			}
			tflog.Debug(ctx, "Save ssh_cert from env var to tmp file", map[string]any{"certPath": certPath})
		} else if value[0] == '-' {
			var err error
			certPath, err = tmpFileWithValue(value)
			if err != nil {
				return "", fmt.Errorf("Unable to create temp file: %w", err)
			}
			tflog.Debug(ctx, "Save ssh_cert from raw string to tmp file", map[string]any{"certPath": certPath})
		} else {
			return "", fmt.Errorf("Unknown cert format")
		}
	}

	return
}

func resolveHomeDir(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("Unable to get current user: %w", err)
	}
	return filepath.Join(usr.HomeDir, path[2:]), nil
}
