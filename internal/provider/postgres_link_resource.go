package provider

import (
	"context"
	"strings"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &postgresLinkResource{}
	_ resource.ResourceWithConfigure   = &postgresLinkResource{}
	_ resource.ResourceWithImportState = &postgresLinkResource{}
)

func NewPostgresLinkResource() resource.Resource {
	return &postgresLinkResource{}
}

type postgresLinkResource struct {
	client *dokkuclient.Client
}

type postgresLinkResourceModel struct {
	AppName     types.String `tfsdk:"app_name"`
	ServiceName types.String `tfsdk:"service_name"`
}

// Metadata returns the resource type name.
func (r *postgresLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgres_link"
}

// Configure adds the provider configured client to the resource.
func (r *postgresLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *postgresLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *postgresLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state postgresLinkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.PostgresServiceExists(ctx, state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to check postgres service existence",
			"Unable to check postgres service existence. "+err.Error(),
		)
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Unable to find postgres service", "Unable to find postgres service")
		return
	}

	// Check link existence
	exists, err = r.client.PostgresLinkExists(ctx, state.ServiceName.ValueString(), state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to check postgres link existence",
			"Unable to check postgres link existence. "+err.Error(),
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
func (r *postgresLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan postgresLinkResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.PostgresServiceExists(ctx, plan.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to check postgres service existence",
			"Unable to check postgres service existence. "+err.Error(),
		)
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Unable to find postgres service", "Unable to find postgres service")
		return
	}

	// Check link existence
	exists, err = r.client.PostgresLinkExists(ctx, plan.ServiceName.ValueString(), plan.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to check postgres link existence",
			"Unable to check postgres link existence. "+err.Error(),
		)
		return
	}
	if exists {
		resp.Diagnostics.AddError("Service already linked to app", "Service already linked to app")
		return
	}

	// Create link
	err = r.client.PostgresLinkCreate(ctx, plan.ServiceName.ValueString(), plan.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create postgres link",
			"Unable to create postgres link. "+err.Error(),
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
func (r *postgresLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Resource doesn't support Update", "Resource doesn't support Update")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *postgresLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state postgresLinkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.PostgresServiceExists(ctx, state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to check postgres service existence",
			"Unable to check postgres service existence. "+err.Error(),
		)
		return
	}
	if !exists {
		return
	}

	// Check link existence
	exists, err = r.client.PostgresLinkExists(ctx, state.ServiceName.ValueString(), state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to check postgres link existence",
			"Unable to check postgres link existence. "+err.Error(),
		)
		return
	}
	if !exists {
		return
	}

	// Unlink service
	err = r.client.PostgresLinkRemove(ctx, state.ServiceName.ValueString(), state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to unlink service",
			"Unable to unlink service. "+err.Error(),
		)
		return
	}
}

func (r *postgresLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, " ")
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_name"), parts[1])...)
}
