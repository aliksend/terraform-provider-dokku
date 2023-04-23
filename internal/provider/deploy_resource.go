package provider

import (
	"context"
	"regexp"
	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &deployResource{}
	_ resource.ResourceWithConfigure      = &deployResource{}
	_ resource.ResourceWithValidateConfig = &deployResource{}
)

func NewDeployResource() resource.Resource {
	return &deployResource{}
}

type deployResource struct {
	client *dokkuclient.Client
}

type deployResourceModel struct {
	AppName            types.String `tfsdk:"app_name"`
	Type               types.String `tfsdk:"type"`
	DockerImage        types.String `tfsdk:"docker_image"`
	GitRepository      types.String `tfsdk:"git_repository"`
	GitRepositoryBuild types.Bool   `tfskd:"git_repository_build"`
	GitRepositoryRef   types.String `tfsdk:"git_repository_ref"`
	ArchiveType        types.String `tfsdk:"archive_type"`
	ArchiveUrl         types.String `tfsdk:"archive_url"`
}

// Metadata returns the resource type name.
func (r *deployResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deploy"
}

// Configure adds the provider configured client to the resource.
func (r *deployResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *deployResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9-]*$`), "invalid app_name"),
				},
			},
			"type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("archive", "docker_image", "git_repository"),
				},
			},
			"docker_image": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"git_repository": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"git_repository_build": schema.BoolAttribute{
				Default: booldefault.StaticBool(false),
			},
			"git_repository_ref": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"archive_type": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"archive_url": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *deployResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data deployResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch data.Type.ValueString() {
	case "archive":
		if data.ArchiveUrl.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("archive_url"), "archive_url must be set for type archive", "archive_url must be set for type archive")
		}
	case "docker_image":
		if data.DockerImage.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("docker_image"), "docker_image must be set for type archive", "docker_image must be set for type archive")
		}
	case "git_repository":
		if data.GitRepository.IsNull() {
			resp.Diagnostics.AddAttributeError(path.Root("git_repository"), "git_repository must be set for type archive", "git_repository must be set for type archive")
		}
	default:
		resp.Diagnostics.AddAttributeError(path.Root("type"), "Invalid type value", "Invalid type value")
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *deployResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state deployResourceModel
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

	// Nothing else to read

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *deployResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan deployResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error
	switch plan.Type.ValueString() {
	case "archive":
		err = r.client.DeployFromArchive(ctx, plan.AppName.ValueString(), plan.ArchiveType.ValueString(), plan.ArchiveUrl.ValueString())
	case "docker_image":
		err = r.client.DeployFromImage(ctx, plan.AppName.ValueString(), plan.DockerImage.ValueString())
	case "git_repository":
		err = r.client.DeploySyncRepository(ctx, plan.AppName.ValueString(), plan.GitRepository.ValueString(), plan.GitRepositoryBuild.ValueBool(), plan.GitRepositoryRef.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to deploy", "Unable to deploy. "+err.Error())
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
func (r *deployResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan deployResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state deployResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to change app name", "Unable to change app name")
		return
	}
	if plan.Type.ValueString() != state.Type.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("type"), "Unable to change type", "Unable to change type")
		return
	}

	var err error
	switch plan.Type.ValueString() {
	case "archive":
		err = r.client.DeployFromArchive(ctx, plan.AppName.ValueString(), plan.ArchiveType.ValueString(), plan.ArchiveUrl.ValueString())
	case "docker_image":
		err = r.client.DeployFromImage(ctx, plan.AppName.ValueString(), plan.DockerImage.ValueString())
	case "git_repository":
		err = r.client.DeploySyncRepository(ctx, plan.AppName.ValueString(), plan.GitRepository.ValueString(), plan.GitRepositoryBuild.ValueBool(), plan.GitRepositoryRef.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to deploy", "Unable to deploy. "+err.Error())
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *deployResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state deployResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeployUnsetSourceImage(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to unset source image", "Unable to unset source image. "+err.Error())
		return
	}
}
