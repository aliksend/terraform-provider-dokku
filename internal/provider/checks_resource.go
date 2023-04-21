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
	_ resource.Resource                = &checksResource{}
	_ resource.ResourceWithConfigure   = &checksResource{}
	_ resource.ResourceWithImportState = &checksResource{}
)

func NewChecksResource() resource.Resource {
	return &checksResource{}
}

type checksResource struct {
	client *goph.Client
}

type checksResourceModel struct {
	AppName types.String `tfsdk:"app_name"`
	Status  types.String `tfsdk:"status"`
}

// Metadata returns the resource type name.
func (r *checksResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_checks"
}

// Configure adds the provider configured client to the resource.
func (r *checksResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *checksResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
			},
			"status": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *checksResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state checksResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read checks
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("checks:report %s", state.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read checks",
			"Unable to read checks. "+err.Error(),
		)
		return
	}

	status := "enabled"
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		title := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch title {
		case "Checks disabled list":
			if value == "_all_" {
				status = "disabled"
				break
			}
		case "Checks skipped list":
			if value == "_all_" {
				status = "skipped"
				break
			}
		}
	}
	state.Status = basetypes.NewStringValue(status)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *checksResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan checksResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set checks
	r.setChecks(ctx, plan.AppName.ValueString(), plan.Status.ValueString(), &resp.Diagnostics)
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
func (r *checksResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan checksResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state checksResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to change app name", "Unable to change app name")
		return
	}

	// Set checks
	r.setChecks(ctx, state.AppName.ValueString(), plan.Status.ValueString(), &resp.Diagnostics)
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
func (r *checksResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state checksResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set checks
	r.setChecks(ctx, state.AppName.ValueString(), "enabled", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *checksResource) setChecks(ctx context.Context, appName string, status string, d *diag.Diagnostics) {
	var action string
	switch status {
	case "enabled":
		action = "enable"
	case "disabled":
		action = "disable"
	case "skipped":
		action = "skip"
	default:
		d.AddAttributeError(path.Root("status"), "Invalid status value", "Invalid status value. Valid values are: enabled, disabled, skipped")
		return
	}

	_, _, err := run(ctx, r.client, fmt.Sprintf("checks:%s %s", action, appName))
	if err != nil {
		d.AddError(
			"Unable to read checks",
			"Unable to read checks. "+err.Error(),
		)
		return
	}
}

func (r *checksResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to app_name attribute
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_name"), req.ID)...)
}
