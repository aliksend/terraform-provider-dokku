package provider

import (
	"context"
	"regexp"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

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
	_ resource.Resource                = &checksResource{}
	_ resource.ResourceWithConfigure   = &checksResource{}
	_ resource.ResourceWithImportState = &checksResource{}
)

func NewChecksResource() resource.Resource {
	return &checksResource{}
}

type checksResource struct {
	client *dokkuclient.Client
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
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *checksResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"status": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled", "skipped"),
				},
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
	status, err := r.client.ChecksGet(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to get checks", "Unable to get checks. "+err.Error())
		return
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
	err := r.client.ChecksSet(ctx, plan.AppName.ValueString(), plan.Status.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to set checks", "Unable to set checks. "+err.Error())
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

	// Read checks
	status, err := r.client.ChecksGet(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to get checks", "Unable to get checks. "+err.Error())
		return
	}
	if status != state.Status.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("status"), "Checks has invalid status", "Checks has invalid status")
		return
	}

	// Set checks
	err = r.client.ChecksSet(ctx, state.AppName.ValueString(), plan.Status.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to set checks", "Unable to set checks. "+err.Error())
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
	err := r.client.ChecksSet(ctx, state.AppName.ValueString(), "enabled")
	if err != nil {
		resp.Diagnostics.AddError("Unable to set checks", "Unable to set checks. "+err.Error())
		return
	}
}

func (r *checksResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to app_name attribute
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_name"), req.ID)...)
}
