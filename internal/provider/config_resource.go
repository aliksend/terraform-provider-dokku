package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/melbahja/goph"
)

var (
	_ resource.Resource              = &configResource{}
	_ resource.ResourceWithConfigure = &configResource{}
)

func NewConfigResource() resource.Resource {
	return &configResource{}
}

type configResource struct {
	client *goph.Client
}

type configResourceModel struct {
	AppName types.String `tfsdk:"app_name"`
	Name    types.String `tfsdk:"name"`
	Value   types.String `tfsdk:"value"`
}

// Metadata returns the resource type name.
func (r *configResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

// Configure adds the provider configured client to the resource.
func (r *configResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *configResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"value": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *configResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state configResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read config value
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("config:get %s %s", state.AppName.ValueString(), state.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read config",
			"Unable to read config. "+err.Error(),
		)
		return
	}

	state.Value = basetypes.NewStringValue(strings.TrimSuffix(stdout, "\n"))

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *configResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan configResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set config
	// TODO --no-restart ?
	_, _, err := run(ctx, r.client, fmt.Sprintf("config:set --encoded %s %s=%q", plan.AppName.ValueString(), plan.Name.ValueString(), base64.StdEncoding.EncodeToString([]byte(plan.Value.ValueString()))))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set config value",
			"Unable to set config value. "+err.Error(),
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
func (r *configResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan configResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state configResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to change app name", "Unable to change app name")
		return
	}

	// Set config value
	// TODO --no-restart?
	_, _, err := run(ctx, r.client, fmt.Sprintf("config:set --encoded %s %s=%q", plan.AppName.ValueString(), plan.Name.ValueString(), base64.StdEncoding.EncodeToString([]byte(plan.Value.ValueString()))))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to set config value",
			"Unable to set config value. "+err.Error(),
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
func (r *configResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state configResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Clear config
	_, _, err := run(ctx, r.client, fmt.Sprintf("config:unset %s %s", state.AppName.ValueString(), state.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to unset config value",
			"Unable to unset config value. "+err.Error(),
		)
		return
	}
}
