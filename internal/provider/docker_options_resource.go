package provider

import (
	"context"
	"regexp"
	"strings"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &dockerOptionResource{}
	_ resource.ResourceWithConfigure   = &dockerOptionResource{}
	_ resource.ResourceWithImportState = &dockerOptionResource{}
)

func NewDockerOptionResource() resource.Resource {
	return &dockerOptionResource{}
}

type dockerOptionResource struct {
	client *dokkuclient.Client
}

type dockerOptionResourceModel struct {
	AppName types.String `tfsdk:"app_name"`
	Phase   types.String `tfsdk:"phase"`
	Value   types.String `tfsdk:"value"`
}

// Metadata returns the resource type name.
func (r *dockerOptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_docker_option"
}

// Configure adds the provider configured client to the resource.
func (r *dockerOptionResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *dockerOptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"phase": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("build", "deploy", "run"),
				},
			},
			"value": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *dockerOptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state dockerOptionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read docker option
	exists, err := r.client.DockerOptionExists(ctx, state.AppName.ValueString(), state.Phase.ValueString(), state.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check docker options existence", "Unable to check docker options existence. "+err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
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
func (r *dockerOptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan dockerOptionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.client.DockerOptionExists(ctx, plan.AppName.ValueString(), plan.Phase.ValueString(), plan.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check docker options existence", "Unable to check docker options existence. "+err.Error())
		return
	}
	if exists {
		resp.Diagnostics.AddError("Docker option already exists", "Docker option already exists")
		return
	}

	// Add docker-option
	err = r.client.DockerOptionAdd(ctx, plan.AppName.ValueString(), plan.Phase.ValueString(), plan.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to add docker-option", "Unable to add docker-option. "+err.Error())
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
func (r *dockerOptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Resource doesn't support Update", "Resource doesn't support Update")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *dockerOptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state dockerOptionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := r.client.DockerOptionExists(ctx, state.AppName.ValueString(), state.Phase.ValueString(), state.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check docker options existence", "Unable to check docker options existence. "+err.Error())
		return
	}
	if !exists {
		return
	}

	// Remove docker-option
	err = r.client.DockerOptionRemove(ctx, state.AppName.ValueString(), state.Phase.ValueString(), state.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to remove docker-option", "Unable to remove docker-option. "+err.Error())
		return
	}
}

func (r *dockerOptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, " ")
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("phase"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), parts[2])...)
}
