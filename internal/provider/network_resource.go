package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/melbahja/goph"
)

var (
	_ resource.Resource              = &networkResource{}
	_ resource.ResourceWithConfigure = &networkResource{}
)

func NewNetworkResource() resource.Resource {
	return &networkResource{}
}

type networkResource struct {
	client *goph.Client
}

type networkResourceModel struct {
	AppName types.String `tfsdk:"app_name"`
	Type    types.String `tfsdk:"type"`
	Name    types.String `tfsdk:"name"`
}

// Metadata returns the resource type name.
func (r *networkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network"
}

// Configure adds the provider configured client to the resource.
func (r *networkResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *networkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
			},
			"type": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *networkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state networkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check network existence
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("network:exists %s", state.Name.ValueString()))
	if err != nil {
		if strings.Contains(stdout, "Network does not exist") {
			resp.Diagnostics.AddError("Unable to find network", "Unable to find network")
			return
		}

		resp.Diagnostics.AddError(
			"Unable to check network existence",
			"Unable to check network existence. "+err.Error(),
		)
		return
	}

	// Check network attached to app
	stdout, _, err = run(ctx, r.client, fmt.Sprintf("network:report %s --network-%s", state.AppName.ValueString(), state.Type.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get network report",
			"Unable to get network report. "+err.Error(),
		)
		return
	}
	networkName := strings.TrimSuffix(stdout, "\n")
	if networkName == "" {
		resp.Diagnostics.AddError("No network set", "No network set")
		return
	}
	state.Name = basetypes.NewStringValue(networkName)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *networkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan networkResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.ensureAndSetNetwork(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
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
func (r *networkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan networkResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state networkResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to change app name", "Unable to change app name")
		return
	}

	// Unset network
	_, _, err := run(ctx, r.client, fmt.Sprintf("network:set %s %s", state.AppName.ValueString(), state.Type.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to unset network",
			"Unable to unset network. "+err.Error(),
		)
		return
	}

	r.ensureAndSetNetwork(ctx, plan, &resp.Diagnostics)
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
func (r *networkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state networkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Unset network
	_, _, err := run(ctx, r.client, fmt.Sprintf("network:set %s %s", state.AppName.ValueString(), state.Type.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to unset network",
			"Unable to unset network. "+err.Error(),
		)
		return
	}
}

func (r *networkResource) ensureAndSetNetwork(ctx context.Context, plan networkResourceModel, d *diag.Diagnostics) {

	// Ensure network
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("network:exists %s", plan.Name.ValueString()))
	if err != nil {
		if strings.Contains(stdout, "Network does not exist") {
			_, _, err := run(ctx, r.client, fmt.Sprintf("network:create %s", plan.Name.ValueString()))
			if err != nil {
				d.AddError(
					"Unable to create network",
					"Unable to create network. "+err.Error(),
				)
				return
			}
		} else {
			d.AddError(
				"Unable to check network existence",
				"Unable to check network existence. "+err.Error(),
			)
			return
		}
	}

	// Set network for app
	_, _, err = run(ctx, r.client, fmt.Sprintf("network:set %s %s %s", plan.AppName.ValueString(), plan.Type.ValueString(), plan.Name.ValueString()))
	if err != nil {
		d.AddError(
			"Unable to set network",
			"Unable to set network. "+err.Error(),
		)
		return
	}
}
