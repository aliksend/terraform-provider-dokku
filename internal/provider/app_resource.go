package provider

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                   = &appResource{}
	_ resource.ResourceWithConfigure      = &appResource{}
	_ resource.ResourceWithImportState    = &appResource{}
	_ resource.ResourceWithValidateConfig = &appResource{}
)

func NewAppResource() resource.Resource {
	return &appResource{}
}

type appResource struct {
	client *dokkuclient.Client
}

type appResourceModel struct {
	AppName       types.String                 `tfsdk:"app_name"`
	Config        map[string]types.String      `tfsdk:"config"`
	Storage       map[string]storageModel      `tfsdk:"storage"`
	Checks        *checkModel                  `tfsdk:"checks"`
	Domains       []types.String               `tfsdk:"domains"`
	ProxyPorts    map[string]proxyPortModel    `tfsdk:"proxy_ports"`
	DockerOptions map[string]dockerOptionModel `tfsdk:"docker_options"`
	Networks      *networkModel                `tfsdk:"networks"`
	Deploy        *deployModel                 `tfsdk:"deploy"`
}

type storageModel struct {
	LocalDirectory types.String `tfsdk:"local_directory"`
	MountPath      types.String `tfsdk:"mount_path"`
}

type checkModel struct {
	Status types.String `tfsdk:"status"`
}

type proxyPortModel struct {
	Scheme        types.String `tfsdk:"scheme"`
	ContainerPort types.String `tfsdk:"container_port"`
}

type dockerOptionModel struct {
	Phase types.Set `tfsdk:"phase"`
}

type networkModel struct {
	AttachPostCreate types.String `tfsdk:"attach_post_create"`
	AttachPostDeploy types.String `tfsdk:"attach_post_deploy"`
	InitialNetwork   types.String `tfsdk:"initial_network"`
}

type deployModel struct {
	Type             types.String `tfsdk:"type"`
	Login            types.String `tfsdk:"login"`
	Password         types.String `tfsdk:"password"`
	DockerImage      types.String `tfsdk:"docker_image"`
	GitRepository    types.String `tfsdk:"git_repository"`
	GitRepositoryRef types.String `tfsdk:"git_repository_ref"`
	ArchiveType      types.String `tfsdk:"archive_type"`
	ArchiveUrl       types.String `tfsdk:"archive_url"`
}

// Metadata returns the resource type name.
func (r *appResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

// Configure adds the provider configured client to the resource.
func (r *appResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *appResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of application to manage",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9-]*$`), "invalid app_name"),
				},
			},
			"config": schema.MapAttribute{
				Optional:    true,
				Description: "Config (env vars) for app",
				ElementType: types.StringType,
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`), "invalid name")),
					mapvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"storage": schema.MapNestedAttribute{
				Optional:    true,
				Description: "Persistent storage setup for app. Keys are storage names or absolute paths to host directories",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"local_directory": schema.StringAttribute{
							Optional:    true,
							Description: "Now working like a crutch. Uploads local directory to host (always, without checking is it changed). Requires SCP to be configured with user that can call sudo without password",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						// Improvements:
						// Variant 1.
						// Calculate checksum of files on remote host on Read. Upload local files on Update only if checksum changed
						// - calculate only for directories with set local_directory to prevent processing large storages
						// - requires run "sha1sum" (or smth like that) on remote host side
						// Variant 2.
						// Calculate checksum of local files and save it. Upload local files on Update if checksum changed
						// - unable to track changes that made on remote host without terraform
						"mount_path": schema.StringAttribute{
							Required:    true,
							Description: "Path inside container to mount to",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
				},
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"checks": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Checks setup for app",
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						Required:    true,
						Description: "Checks status. Default: enabled",
						Validators: []validator.String{
							stringvalidator.OneOf("enabled", "disabled", "skipped"),
						},
					},
				},
			},
			"domains": schema.SetAttribute{
				Optional:    true,
				Description: "Domains setup for app",
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"proxy_ports": schema.MapNestedAttribute{
				Optional:    true,
				Description: "Proxy ports setup for app. Keys are host ports",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"scheme": schema.StringAttribute{
							Required:    true,
							Description: "Scheme to use. Allowed values: http, https",
							Validators: []validator.String{
								stringvalidator.OneOf("http", "https"),
							},
						},
						"container_port": schema.StringAttribute{
							Required:    true,
							Description: "Port inside container to proxy",
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "Must be integer"),
							},
						},
					},
				},
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "Must be integer")),
				},
			},
			"docker_options": schema.MapNestedAttribute{
				Optional:    true,
				Description: "Docker options for app. Keys are options",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"phase": schema.SetAttribute{
							Required:    true,
							Description: "Phase to apply docker-options to. Allowed values: build, deploy, run",
							ElementType: types.StringType,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(stringvalidator.OneOf("build", "deploy", "run")),
							},
						},
					},
				},
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"networks": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Network setup for app",
				Attributes: map[string]schema.Attribute{
					"attach_post_create": schema.StringAttribute{
						Optional:    true,
						Description: "Name of network to use as attach-post-create",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"attach_post_deploy": schema.StringAttribute{
						Optional:    true,
						Description: "Name of network to use as attach-post-deploy",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"initial_network": schema.StringAttribute{
						Optional:    true,
						Description: "Name of network to use as initial-network",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
			},
			"deploy": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Deploy setup for app",
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Type of deploy to use. Allowed values: archive, docker_image, git_repository",
						Validators: []validator.String{
							stringvalidator.OneOf("archive", "docker_image", "git_repository"),
						},
					},
					"login": schema.StringAttribute{
						Optional:    true,
						Description: "Login to use for deployment",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("password")),
						},
					},
					"password": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "Password to use for deployment",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
							stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("login")),
						},
					},
					"docker_image": schema.StringAttribute{
						Optional:    true,
						Description: "Docker image to deploy from. If login and password is provided then it will be used to sign in to docker registry.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"git_repository": schema.StringAttribute{
						Optional:    true,
						Description: "Git repository to deploy from. If login and password is provided then it will be used to sign in to repository.",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"git_repository_ref": schema.StringAttribute{
						Optional:    true,
						Description: "Ref of git repository to deploy from",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"archive_url": schema.StringAttribute{
						Optional:    true,
						Description: "URL of archive to delpoy from. Login and password will not be used",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"archive_type": schema.StringAttribute{
						Optional:    true,
						Description: "Type of archive to deploy. Allowed values: tar, tar.gz, zip", // https://github.com/dokku/dokku/blob/master/plugins/git/git-from-archive#L25
						Validators: []validator.String{
							stringvalidator.OneOf("tar", "tar.gz", "zip"),
						},
					},
				},
			},
		},
	}
}
func (r *appResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data appResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Deploy != nil {
		switch data.Deploy.Type.ValueString() {
		case "archive":
			if data.Deploy.ArchiveUrl.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("deploy").AtName("archive_url"), "archive_url must be set for type archive", "archive_url must be set for type archive")
			}
		case "docker_image":
			if data.Deploy.DockerImage.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("deploy").AtName("docker_image"), "docker_image must be set for type archive", "docker_image must be set for type archive")
			}
		case "git_repository":
			if data.Deploy.GitRepository.IsNull() {
				resp.Diagnostics.AddAttributeError(path.Root("deploy").AtName("git_repository"), "git_repository must be set for type archive", "git_repository must be set for type archive")
			}
		default:
			resp.Diagnostics.AddAttributeError(path.Root("deploy").AtName("type"), "Invalid type value", "Invalid type value")
		}
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *appResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state appResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check app existence
	exists, err := r.client.AppExists(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to check app existence", "Unable to check app existence. "+err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	config, err := r.client.ConfigExport(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("config"), "Unable to get config", "Unable to get config")
	} else {
		cfg := make(map[string]basetypes.StringValue)
		for k, v := range config {
			found := false
			for knownK := range state.Config {
				if k == knownK {
					found = true
					break
				}
			}
			// only known keys
			if found {
				cfg[k] = basetypes.NewStringValue(v)
			}
		}
		if len(cfg) == 0 {
			state.Config = nil
		} else {
			state.Config = cfg
		}
	}

	storage, err := r.client.StorageExport(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("storage"), "Unable to get storage", "Unable to get storage")
	} else {
		if len(storage) == 0 {
			state.Storage = nil
		} else {
			stateStorage := make(map[string]storageModel)
			for k, v := range storage {
				localDirectory := basetypes.NewStringNull()
				if storageConfig, ok := state.Storage[k]; ok {
					localDirectory = storageConfig.LocalDirectory
				}

				stateStorage[k] = storageModel{
					MountPath:      basetypes.NewStringValue(v),
					LocalDirectory: localDirectory,
				}
			}
			state.Storage = stateStorage
		}
	}

	checkStatus, err := r.client.ChecksGet(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("checks"), "Unable to get checks", "Unable to get checks")
	} else {
		if checkStatus == "enabled" {
			state.Checks = nil
		} else {
			state.Checks = &checkModel{
				Status: basetypes.NewStringValue(checkStatus),
			}
		}
	}

	domains, err := r.client.DomainsExport(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("domains"), "Unable to get domains", "Unable to get domains")
	} else {
		if len(domains) == 0 {
			state.Domains = nil
		} else {
			state.Domains = make([]types.String, len(domains))
			for i, d := range domains {
				state.Domains[i] = basetypes.NewStringValue(d)
			}
		}
	}

	proxyPorts, err := r.client.ProxyPortsExport(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("proxy_ports"), "Unable to get proxy_ports", "Unable to get proxy_ports")
	} else {
		pp := make(map[string]proxyPortModel)
		for _, p := range proxyPorts {
			found := false
			for v := range state.ProxyPorts {
				if v == p.HostPort {
					found = true
					break
				}
			}
			// only known hostport's
			if found {
				pp[p.HostPort] = proxyPortModel{
					Scheme:        basetypes.NewStringValue(p.Scheme),
					ContainerPort: basetypes.NewStringValue(p.ContainerPort),
				}
			}
		}
		if len(pp) == 0 {
			state.ProxyPorts = nil
		} else {
			state.ProxyPorts = pp
		}
	}

	// dockerOptions -- unable to read because it can be set externally

	networks, err := r.client.NetworksReport(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("networks"), "Unable to get networks", "Unable to get networks")
	} else {
		var attachPostCreate types.String
		if networks["attach post create"] != "" {
			attachPostCreate = basetypes.NewStringValue(networks["attach post create"])
		} else {
			attachPostCreate = basetypes.NewStringNull()
		}
		var attachPostDeploy types.String
		if networks["attach post deploy"] != "" {
			attachPostDeploy = basetypes.NewStringValue(networks["attach post deploy"])
		} else {
			attachPostDeploy = basetypes.NewStringNull()
		}
		var initialNetwork types.String
		if networks["initial network"] != "" {
			initialNetwork = basetypes.NewStringValue(networks["initial network"])
		} else {
			initialNetwork = basetypes.NewStringNull()
		}
		if attachPostCreate.IsNull() && attachPostDeploy.IsNull() && initialNetwork.IsNull() {
			state.Networks = nil
		} else {
			state.Networks = &networkModel{
				AttachPostCreate: attachPostCreate,
				AttachPostDeploy: attachPostDeploy,
				InitialNetwork:   initialNetwork,
			}
		}
	}

	// deploy -- unable to read

	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *appResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan appResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check app existence
	exists, err := r.client.AppExists(ctx, plan.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to check app existence", "Unable to check app existence. "+err.Error())
		return
	}
	if exists {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "App already exists", "App with specified name already exists")
		return
	}

	// Create new app
	err = r.client.AppCreate(ctx, plan.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create app", "Unable to create app. "+err.Error())
		// if not created - return to not try to destroy on other errors
		return
	}

	if len(plan.Config) != 0 {
		config := make(map[string]string)
		for k, v := range plan.Config {
			config[k] = v.ValueString()
		}
		err := r.client.ConfigSet(ctx, plan.AppName.ValueString(), config)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("config"), "Unable to set config", "Unable to set config")
		}
	}

	for hostPath, storage := range plan.Storage {
		err := r.client.StorageEnsureAndMount(ctx, plan.AppName.ValueString(), hostPath, storage.MountPath.ValueString(), storage.LocalDirectory.ValueStringPointer())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("storage").AtMapKey(hostPath), "Unable to mount storage", "Unable to mount storage")
		}
	}

	if plan.Checks != nil {
		if !plan.Checks.Status.IsNull() {
			err := r.client.ChecksSet(ctx, plan.AppName.ValueString(), plan.Checks.Status.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("checks"), "Unable to set checks", "Unable to set checks")
			}
		}
	}

	if len(plan.Domains) != 0 {
		var domains []string
		for _, domain := range plan.Domains {
			domains = append(domains, domain.ValueString())
		}
		err := r.client.DomainsSet(ctx, plan.AppName.ValueString(), domains)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("domains"), "Unable to add domain", "Unable to add domain")
		}
		err = r.client.DomainsEnable(ctx, plan.AppName.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("domains"), "Unable to enable domains support", "Unable to enable domains support")
		}
	} else {
		err = r.client.DomainsDisable(ctx, plan.AppName.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("domains"), "Unable to disable domains support", "Unable to disable domains support")
		}
	}

	if len(plan.ProxyPorts) != 0 {
		var proxyPorts []dokkuclient.ProxyPort
		for hostPort, proxyPort := range plan.ProxyPorts {
			proxyPorts = append(proxyPorts, dokkuclient.ProxyPort{
				Scheme:        proxyPort.Scheme.ValueString(),
				HostPort:      hostPort,
				ContainerPort: proxyPort.ContainerPort.ValueString(),
			})
		}
		err := r.client.ProxyPortsSet(ctx, plan.AppName.ValueString(), proxyPorts)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("proxy_ports"), "Unable to set proxy-ports", "Unable to set proxy-ports")
		}
		err = r.client.ProxyEnable(ctx, plan.AppName.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("proxy_ports"), "Unable to enable proxy-ports", "Unable to enable proxy-ports")
		}
	} else {
		err = r.client.ProxyDisable(ctx, plan.AppName.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("proxy_ports"), "Unable to disable proxy-ports", "Unable to disable proxy-ports")
		}
	}

	for option, dockerOption := range plan.DockerOptions {
		err := r.client.DockerOptionAdd(ctx, plan.AppName.ValueString(), formatDockerOptionsPhases(dockerOption.Phase), option)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("docker_options").AtMapKey(option), "Unable to add docker option", "Unable to add docker option")
		}
	}

	if plan.Networks != nil {
		if !plan.Networks.AttachPostCreate.IsNull() {
			err := r.client.NetworkEnsureAndSetForApp(ctx, plan.AppName.ValueString(), "attach-post-create", plan.Networks.AttachPostCreate.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("attach_post_create"), "Unable to set network", "Unable to set network")
			}
		}
		if !plan.Networks.AttachPostDeploy.IsNull() {
			err := r.client.NetworkEnsureAndSetForApp(ctx, plan.AppName.ValueString(), "attach-post-deploy", plan.Networks.AttachPostDeploy.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("attach_post_deploy"), "Unable to set network", "Unable to set network")
			}
		}
		if !plan.Networks.InitialNetwork.IsNull() {
			err := r.client.NetworkEnsureAndSetForApp(ctx, plan.AppName.ValueString(), "initial_network", plan.Networks.InitialNetwork.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("initial_network"), "Unable to set network", "Unable to set network")
			}
		}
	}

	if plan.Deploy != nil && !resp.Diagnostics.HasError() {
		err := r.deploy(ctx, plan.AppName.ValueString(), *plan.Deploy)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("deploy"), "Unable to deploy", "Unable to deploy. "+err.Error())
		}
	}

	if resp.Diagnostics.HasError() {
		err := r.client.AppDestroy(ctx, plan.AppName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to destroy app", "Unable to destroy app. "+err.Error())
		}
		return
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *appResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan appResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state appResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "App name can't be changed", "App name can't be changed")
		return
	}
	appName := plan.AppName.ValueString()

	restartRequired := false

	// -- config
	var namesToUnset []string
	for stateName := range state.Config {
		found := false
		for planName := range plan.Config {
			if planName == stateName {
				found = true
				break
			}
		}
		if !found {
			namesToUnset = append(namesToUnset, stateName)
		}
	}
	if len(namesToUnset) != 0 {
		err := r.client.ConfigUnset(ctx, appName, namesToUnset)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("config"), "Unable to unset config", "Unable to unset config. "+err.Error())
		}
		restartRequired = true
	}

	configToSet := make(map[string]string)
	for k, v := range plan.Config {
		if !state.Config[k].Equal(v) {
			configToSet[k] = v.ValueString()
		}
	}
	if len(configToSet) != 0 {
		err := r.client.ConfigSet(ctx, appName, configToSet)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("config"), "Unable to set config", "Unable to set config. "+err.Error())
		}
		restartRequired = true
	}
	// --

	// -- storage
	for existingName, existingStorage := range state.Storage {
		found := false
		for planName, planStorage := range plan.Storage {
			if existingName == planName {
				found = true

				if !existingStorage.MountPath.Equal(planStorage.MountPath) || !planStorage.LocalDirectory.IsNull() {
					err := r.client.StorageUnmount(ctx, appName, existingName, existingStorage.MountPath.ValueString())
					if err != nil {
						resp.Diagnostics.AddAttributeError(path.Root("storage").AtMapKey(existingName), "Unable to unmount storage", "Unable to unmount storage. "+err.Error())
					}

					err = r.client.StorageEnsureAndMount(ctx, appName, planName, planStorage.MountPath.ValueString(), planStorage.LocalDirectory.ValueStringPointer())
					if err != nil {
						resp.Diagnostics.AddAttributeError(path.Root("storage").AtMapKey(existingName), "Unable to mount storage", "Unable to mount storage. "+err.Error())
					}
				}

				break
			}
		}
		if !found {
			err := r.client.StorageUnmount(ctx, appName, existingName, existingStorage.MountPath.ValueString())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("storage").AtMapKey(existingName), "Unable to unmount storage", "Unable to unmount storage. "+err.Error())
			}
		}
	}
	for planName, planStorage := range plan.Storage {
		found := false
		for existingName := range state.Storage {
			if existingName == planName {
				found = true
				break
			}
		}
		if !found {
			err := r.client.StorageEnsureAndMount(ctx, appName, planName, planStorage.MountPath.ValueString(), planStorage.LocalDirectory.ValueStringPointer())
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("storage").AtMapKey(planName), "Unable to mount storage", "Unable to mount storage. "+err.Error())
			}
		}
	}
	// --

	// -- checks
	stateCheckStatus := "enabled"
	if state.Checks != nil {
		stateCheckStatus = state.Checks.Status.ValueString()
	}
	planCheckStatus := "enabled"
	if plan.Checks != nil {
		planCheckStatus = plan.Checks.Status.ValueString()
	}
	if stateCheckStatus != planCheckStatus {
		err := r.client.ChecksSet(ctx, appName, planCheckStatus)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("checks"), "Unable to set checks", "Unable to set checks. "+err.Error())
		}
	}
	// --

	// -- domains
	needToSetDomains := false
	var domainsToSet []string
	for _, existingDomain := range state.Domains {
		found := false
		for _, planDomain := range plan.Domains {
			if planDomain == existingDomain {
				found = true
				break
			}
		}
		if !found {
			needToSetDomains = true
		}
	}
	for _, planDomain := range plan.Domains {
		found := false
		for _, existingDomain := range state.Domains {
			if planDomain == existingDomain {
				found = true
				break
			}
		}
		if !found {
			needToSetDomains = true
		}
		domainsToSet = append(domainsToSet, planDomain.ValueString())
	}
	if needToSetDomains {
		var err error
		if len(domainsToSet) == 0 {
			err = r.client.DomainsDisable(ctx, appName)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("domains"), "Unable to disable domains support", "Unable to disable domains support. "+err.Error())
			}

			err = r.client.DomainsClear(ctx, appName)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("domains"), "Unable to clear domains", "Unable to clear domains. "+err.Error())
			}
		} else {
			err = r.client.DomainsEnable(ctx, appName)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("domains"), "Unable to enable domains support", "Unable to enable domains support. "+err.Error())
			}

			err = r.client.DomainsSet(ctx, appName, domainsToSet)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("domains"), "Unable to set domains", "Unable to set domains. "+err.Error())
			}
		}
	}
	// --

	// -- proxy ports
	needToSetProxyPorts := false
	var proxyPortsToSet []dokkuclient.ProxyPort
	for existingHostPort, existingProxyPort := range state.ProxyPorts {
		found := false
		for planHostPort, planProxyPort := range plan.ProxyPorts {
			if planHostPort == existingHostPort {
				if planProxyPort.Scheme.Equal(existingProxyPort.Scheme) && planProxyPort.ContainerPort.Equal(existingProxyPort.ContainerPort) {
					found = true
				}
				break
			}
		}
		if !found {
			needToSetProxyPorts = true
		}
	}
	for planHostPort, planProxyPort := range plan.ProxyPorts {
		found := false
		for existingHostPort, existingProxyPort := range state.ProxyPorts {
			if planHostPort == existingHostPort {
				if planProxyPort.Scheme.Equal(existingProxyPort.Scheme) && planProxyPort.ContainerPort.Equal(existingProxyPort.ContainerPort) {
					found = true
				}
				break
			}
		}
		if !found {
			needToSetProxyPorts = true
		}
		proxyPortsToSet = append(proxyPortsToSet, dokkuclient.ProxyPort{
			Scheme:        planProxyPort.Scheme.ValueString(),
			HostPort:      planHostPort,
			ContainerPort: planProxyPort.ContainerPort.ValueString(),
		})
	}
	if needToSetProxyPorts {
		if len(proxyPortsToSet) == 0 {
			err := r.client.ProxyPortsClear(ctx, appName)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("proxy_ports"), "Unable to clear proxy ports", "Unable to clear proxy ports. "+err.Error())
			}

			err = r.client.ProxyDisable(ctx, appName)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("proxy_ports"), "Unable to disable proxy ports", "Unable to disable proxy ports. "+err.Error())
			}
		} else {
			err := r.client.ProxyPortsSet(ctx, appName, proxyPortsToSet)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("proxy_ports"), "Unable to set proxy ports", "Unable to set proxy ports. "+err.Error())
			}

			err = r.client.ProxyEnable(ctx, appName)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("proxy_ports"), "Unable to enable proxy ports", "Unable to enable proxy ports. "+err.Error())
			}
		}
	}
	// --

	// -- docker options
	for existingValue, existingDockerOption := range state.DockerOptions {
		found := false
		for planValue, planDockerOption := range plan.DockerOptions {
			if existingValue == planValue {
				found = true

				if !existingDockerOption.Phase.Equal(planDockerOption.Phase) {
					err := r.client.DockerOptionRemove(ctx, appName, formatDockerOptionsPhases(existingDockerOption.Phase), existingValue)
					if err != nil {
						resp.Diagnostics.AddAttributeError(path.Root("storage").AtMapKey(existingValue), "Unable to remove docker option", "Unable to remove docker option. "+err.Error())
					}

					err = r.client.DockerOptionAdd(ctx, appName, formatDockerOptionsPhases(planDockerOption.Phase), planValue)
					if err != nil {
						resp.Diagnostics.AddAttributeError(path.Root("storage").AtMapKey(existingValue), "Unable to add docker option", "Unable to add docker option. "+err.Error())
					}
				}

				break
			}
		}
		if !found {
			err := r.client.DockerOptionRemove(ctx, appName, formatDockerOptionsPhases(existingDockerOption.Phase), existingValue)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("docker_options").AtMapKey(existingValue), "Unable to remove docker option", "Unable to remove docker option. "+err.Error())
			}
		}
	}
	for planValue, planDockerOption := range plan.DockerOptions {
		found := false
		for existingValue := range state.DockerOptions {
			if existingValue == planValue {
				found = true
				break
			}
		}
		if !found {
			err := r.client.DockerOptionAdd(ctx, appName, formatDockerOptionsPhases(planDockerOption.Phase), planValue)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("docker_options").AtMapKey(planValue), "Unable to add docker option", "Unable to add docker option. "+err.Error())
			}
		}
	}
	// --

	// -- networks
	if state.Networks != nil {
		if plan.Networks != nil {
			if !plan.Networks.AttachPostCreate.Equal(state.Networks.AttachPostCreate) {
				err := r.client.NetworkEnsureAndSetForApp(ctx, appName, "attach-post-create", plan.Networks.AttachPostCreate.ValueString())
				if err != nil {
					resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("attach_post_create"), "Unable to set network", "Unable to set network. "+err.Error())
				}
			}
			if !plan.Networks.AttachPostDeploy.Equal(state.Networks.AttachPostDeploy) {
				err := r.client.NetworkEnsureAndSetForApp(ctx, appName, "attach-post-deploy", plan.Networks.AttachPostDeploy.ValueString())
				if err != nil {
					resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("attach_post_deploy"), "Unable to set network", "Unable to set network. "+err.Error())
				}
			}
			if !plan.Networks.InitialNetwork.Equal(state.Networks.InitialNetwork) {
				err := r.client.NetworkEnsureAndSetForApp(ctx, appName, "initial-network", plan.Networks.InitialNetwork.ValueString())
				if err != nil {
					resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("initial_network"), "Unable to set network", "Unable to set network. "+err.Error())
				}
			}
		} else {
			if !state.Networks.AttachPostCreate.IsNull() {
				err := r.client.NetworkUnsetForApp(ctx, appName, "attach-post-create")
				if err != nil {
					resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("attach_post_create"), "Unable to unset network", "Unable to unset network. "+err.Error())
				}
			}
			if !state.Networks.AttachPostDeploy.IsNull() {
				err := r.client.NetworkUnsetForApp(ctx, appName, "attach-post-deploy")
				if err != nil {
					resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("attach_post_deploy"), "Unable to unset network", "Unable to unset network. "+err.Error())
				}
			}
			if !state.Networks.InitialNetwork.IsNull() {
				err := r.client.NetworkUnsetForApp(ctx, appName, "initial-network")
				if err != nil {
					resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("initial_network"), "Unable to unset network", "Unable to unset network. "+err.Error())
				}
			}
		}
	} else {
		if plan.Networks != nil {
			if !plan.Networks.AttachPostCreate.IsNull() {
				err := r.client.NetworkEnsureAndSetForApp(ctx, appName, "attach-post-create", plan.Networks.AttachPostCreate.ValueString())
				if err != nil {
					resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("attach_post_create"), "Unable to set network", "Unable to set network. "+err.Error())
				}
			}
			if !plan.Networks.AttachPostDeploy.IsNull() {
				err := r.client.NetworkEnsureAndSetForApp(ctx, appName, "attach-post-deploy", plan.Networks.AttachPostDeploy.ValueString())
				if err != nil {
					resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("attach_post_deploy"), "Unable to set network", "Unable to set network. "+err.Error())
				}
			}
			if !plan.Networks.InitialNetwork.IsNull() {
				err := r.client.NetworkEnsureAndSetForApp(ctx, appName, "initial-network", plan.Networks.InitialNetwork.ValueString())
				if err != nil {
					resp.Diagnostics.AddAttributeError(path.Root("networks").AtName("initial_network"), "Unable to set network", "Unable to set network. "+err.Error())
				}
			}
		}
	}
	// --

	// -- deploy
	if plan.Deploy != nil {
		err := r.deploy(ctx, plan.AppName.ValueString(), *plan.Deploy)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("deploy"), "Unable to deploy", "Unable to deploy. "+err.Error())
		}
		restartRequired = false
	}
	// --

	if !resp.Diagnostics.HasError() && restartRequired {
		err := r.client.ProcessRestart(ctx, appName)
		if err != nil {
			resp.Diagnostics.AddError("Unable to restart process", "Unable to restart process. "+err.Error())
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *appResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state appResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.client.AppExists(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to check app existence", "Unable to check app existence. "+err.Error())
		return
	}
	if !exists {
		return
	}

	// Delete existing app
	err = r.client.AppDestroy(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to destroy app", "Unable to destroy app. "+err.Error())
		return
	}
}

func (r *appResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to app_name attribute
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_name"), req.ID)...)
}

func (r *appResource) deploy(ctx context.Context, appName string, deployModel deployModel) error {
	var err error
	switch deployModel.Type.ValueString() {
	case "archive":
		err = r.client.DeployFromArchive(ctx, appName, deployModel.ArchiveType.ValueString(), deployModel.ArchiveUrl.ValueString())
	case "docker_image":
		if !deployModel.Login.IsNull() && !deployModel.Password.IsNull() {
			u, err := url.Parse("https://" + deployModel.DockerImage.ValueString())
			if err != nil {
				return fmt.Errorf("unable to parse url: %w", err)
			}
			err = r.client.RegistryLogin(ctx, u.Host, deployModel.Login.ValueString(), deployModel.Password.ValueString())
			if err != nil {
				return fmt.Errorf("unable to login to registry: %w", err)
			}
		}

		err = r.client.DeployFromImage(ctx, appName, deployModel.DockerImage.ValueString())
	case "git_repository":
		if !deployModel.Login.IsNull() && !deployModel.Password.IsNull() {
			u, err := url.Parse(deployModel.GitRepository.ValueString())
			if err != nil {
				return fmt.Errorf("unable to parse url: %w", err)
			}
			err = r.client.GitAuth(ctx, u.Host, deployModel.Login.ValueString(), deployModel.Password.ValueString())
			if err != nil {
				return fmt.Errorf("unable to login to git: %w", err)
			}
		}

		err = r.client.DeploySyncRepository(ctx, appName, deployModel.GitRepository.ValueString(), deployModel.GitRepositoryRef.ValueString())
	default:
		err = fmt.Errorf("Unknown deploy type %s", deployModel.Type.ValueString())
	}
	return err
}

func formatDockerOptionsPhases(phasesSet types.Set) (phases []string) {
	for _, phase := range phasesSet.Elements() {
		//nolint:forcetypeassert
		phaseStr := phase.(types.String)
		phases = append(phases, phaseStr.ValueString())
	}
	return
}
