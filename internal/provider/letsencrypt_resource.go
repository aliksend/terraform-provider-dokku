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
	_ resource.Resource              = &letsencryptResource{}
	_ resource.ResourceWithConfigure = &letsencryptResource{}
)

func NewLetsencryptResource() resource.Resource {
	return &letsencryptResource{}
}

type letsencryptResource struct {
	client *goph.Client
}

type letsencryptResourceModel struct {
	AppName types.String `tfsdk:"app_name"`
}

// Metadata returns the resource type name.
func (r *letsencryptResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_letsencrypt"
}

// Configure adds the provider configured client to the resource.
func (r *letsencryptResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *letsencryptResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *letsencryptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state letsencryptResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read letsencrypt status
	stdout, _, err := run(ctx, r.client, "letsencrypt:list")
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read letsencrypt status",
			"Unable to read letsencrypt status. "+err.Error(),
		)
		return
	}

	found := false
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		appName := strings.TrimSpace(parts[0])
		if appName == state.AppName.ValueString() {
			found = true
			break
		}
	}
	if !found {
		resp.Diagnostics.AddError(
			"Unable to find letsencrypt for app",
			"Unable to find letsencrypt for app",
		)
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
func (r *letsencryptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan letsencryptResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Enable letsencrypt
	_, _, err := run(ctx, r.client, fmt.Sprintf("letsencrypt:enable %s", plan.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to enable letsencrypt",
			"Unable to enable letsencrypt. "+err.Error(),
		)
		return
	}

	// Add cronjob for auto-renew
	_, _, err = run(ctx, r.client, "letsencrypt:cron-job --add")
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to add letsencrypt cronjob",
			"Unable to add letsencrypt cronjob. "+err.Error(),
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
func (r *letsencryptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan letsencryptResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state letsencryptResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to change app name", "Unable to change app name")
		return
	}

	// Nothing to update

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *letsencryptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state letsencryptResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Disable letsencrypt
	_, _, err := run(ctx, r.client, fmt.Sprintf("letsencrypt:disable %s", state.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to disable letsencrypt",
			"Unable to disable letsencrypt. "+err.Error(),
		)
		return
	}

}
