package provider

import (
	"context"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &globalDomainResource{}
	_ resource.ResourceWithConfigure   = &globalDomainResource{}
	_ resource.ResourceWithImportState = &globalDomainResource{}
)

func NewGlobalDomainResource() resource.Resource {
	return &globalDomainResource{}
}

type globalDomainResource struct {
	client *dokkuclient.Client
}

type globalDomainResourceModel struct {
	Domain types.String `tfsdk:"domain"`
}

// Metadata returns the resource type name.
func (r *globalDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_domain"
}

// Configure adds the provider configured client to the resource.
func (r *globalDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *globalDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"domain": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *globalDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state globalDomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read domains
	exists, err := r.client.GlobalDomainExists(ctx, state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read domains",
			"Unable to read domains. "+err.Error(),
		)
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
func (r *globalDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan globalDomainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read domains
	exists, err := r.client.GlobalDomainExists(ctx, plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read domains",
			"Unable to read domains. "+err.Error(),
		)
		return
	}
	if exists {
		resp.Diagnostics.AddError("This global domain is already set", "This global domain is already set")
		return
	}

	// Add domain
	err = r.client.GlobalDomainAdd(ctx, plan.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create domain",
			"Unable to create domain. "+err.Error(),
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
func (r *globalDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Resource doesn't support Update", "Resource doesn't support Update")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *globalDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state globalDomainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read domains
	exists, err := r.client.GlobalDomainExists(ctx, state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read domains",
			"Unable to read domains. "+err.Error(),
		)
		return
	}
	if !exists {
		return
	}

	// Clear domains
	err = r.client.GlobalDomainRemove(ctx, state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete domain",
			"Unable to delete domain. "+err.Error(),
		)
		return
	}
}

func (r *globalDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to app_name attribute
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}
