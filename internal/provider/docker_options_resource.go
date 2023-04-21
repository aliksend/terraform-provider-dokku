package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/melbahja/goph"
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
	client *goph.Client
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
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *dockerOptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
			},
			"phase": schema.StringAttribute{
				Required: true,
			},
			"value": schema.StringAttribute{
				Required: true,
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
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("docker-options:report %s", state.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read docker options",
			"Unable to read docker options. "+err.Error(),
		)
		return
	}

	expectedName := fmt.Sprintf("Docker options %s", state.Phase.ValueString())
	expectedOption := state.Value.ValueString()
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	found := false
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		name := strings.TrimSpace(parts[0])
		if expectedName != name {
			continue
		}
		existingOptions := strings.TrimSpace(parts[1])
		if !strings.Contains(existingOptions, expectedOption) {
			resp.Diagnostics.AddError("Docker options doesn't contain expected option", "Docker options "+existingOptions+" doesn't contain expected option "+expectedOption)
			return
		}
		found = true
		break
	}
	if !found {
		resp.Diagnostics.AddError("Unable to find docker option", "Unable to find docker option for phase "+state.Phase.ValueString())
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

	// Add docker-option
	_, _, err := run(ctx, r.client, fmt.Sprintf("docker-options:add %s %s %s", plan.AppName.ValueString(), plan.Phase.ValueString(), plan.Value.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to add docker-option",
			"Unable to add docker-option. "+err.Error(),
		)
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
	// Retrieve values from plan
	var plan dockerOptionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state dockerOptionResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to change app name", "Unable to change app name")
		return
	}

	// Remove docker-option
	_, _, err := run(ctx, r.client, fmt.Sprintf("docker-options:remove %s %s %s", state.AppName.ValueString(), state.Phase.ValueString(), state.Value.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to remove docker-option",
			"Unable to remove docker-option. "+err.Error(),
		)
		return
	}

	// Add docker-option
	_, _, err = run(ctx, r.client, fmt.Sprintf("docker-options:add %s %s %s", plan.AppName.ValueString(), plan.Phase.ValueString(), plan.Value.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to add docker-option",
			"Unable to add docker-option. "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
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

	// Remove docker-option
	_, _, err := run(ctx, r.client, fmt.Sprintf("docker-options:remove %s %s %s", state.AppName.ValueString(), state.Phase.ValueString(), state.Value.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to remove docker-option",
			"Unable to remove docker-option. "+err.Error(),
		)
		return
	}
}

func (r *dockerOptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, " ")
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("phase"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), parts[2])...)
}
