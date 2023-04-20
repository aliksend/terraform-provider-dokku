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
	_ resource.Resource              = &postgresResource{}
	_ resource.ResourceWithConfigure = &postgresResource{}
)

func NewPostgresResource() resource.Resource {
	return &postgresResource{}
}

type postgresResource struct {
	client *goph.Client
}

type postgresResourceModel struct {
	ServiceName types.String `tfsdk:"service_name"`
}

// Metadata returns the resource type name.
func (r *postgresResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgres"
}

// Configure adds the provider configured client to the resource.
func (r *postgresResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *postgresResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"service_name": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *postgresResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state postgresResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("postgres:exists %s", state.ServiceName.ValueString()))
	if err != nil {
		if strings.Contains(stdout, fmt.Sprintf("Postgres service %s does not exist", state.ServiceName.ValueString())) {
			resp.Diagnostics.AddError("Unable to find postgres service", "Unable to find postgres service")
			return
		}

		resp.Diagnostics.AddError(
			"Unable to check postgres service existence",
			"Unable to check postgres service existence. "+err.Error(),
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
func (r *postgresResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan postgresResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create service is not exists
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("postgres:exists %s", plan.ServiceName.ValueString()))
	if err != nil {
		if strings.Contains(stdout, fmt.Sprintf("Postgres service %s does not exist", plan.ServiceName.ValueString())) {
			_, _, err := run(ctx, r.client, fmt.Sprintf("postgres:create %s", plan.ServiceName.ValueString()))
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to create postgres service",
					"Unable to create postgres service. "+err.Error(),
				)
				return
			}
		} else {
			resp.Diagnostics.AddError(
				"Unable to check postgres service existence",
				"Unable to check postgres service existence. "+err.Error(),
			)
			return
		}
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *postgresResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan postgresResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state postgresResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ServiceName.ValueString() != state.ServiceName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("service_name"), "Unable to change service name", "Unable to change service name")
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
func (r *postgresResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state postgresResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Destroy instance
	_, _, err := run(ctx, r.client, fmt.Sprintf("postgres:destroy %s --force", state.ServiceName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to destroy service",
			"Unable to destroy service. "+err.Error(),
		)
		return
	}
}
