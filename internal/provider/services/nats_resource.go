package services

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
	_ resource.Resource                = &natsResource{}
	_ resource.ResourceWithConfigure   = &natsResource{}
	_ resource.ResourceWithImportState = &natsResource{}
)

func NewNatsResource() resource.Resource {
	return &natsResource{}
}

type natsResource struct {
	client *dokkuclient.Client
}

type natsResourceModel struct {
	ServiceName   types.String `tfsdk:"service_name"`
	ConfigOptions types.String `tfsdk:"config_options"`
}

// Metadata returns the resource type name.
func (r *natsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nats"
}

// Configure adds the provider configured client to the resource.
func (r *natsResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *natsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"service_name": schema.StringAttribute{
				Required:    true,
				Description: "Service name to create",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9-]*$`), "invalid service_name"),
				},
			},
			"config_options": schema.StringAttribute{
				Optional:    true,
				Description: "Config options to create service with",
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
func (r *natsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state natsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.SimpleServiceExists(ctx, "nats", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check nats service existence", "Unable to check nats service existence. "+err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	info, err := r.client.SimpleServiceInfo(ctx, "nats", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to get nats service info", "Unable to get nats service info. "+err.Error())
		return
	}
	if info["Config options"] != "" {
		state.ConfigOptions = basetypes.NewStringValue(info["Config options"])
	} else {
		state.ConfigOptions = basetypes.NewStringNull()
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *natsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan natsResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create service is not exists
	exists, err := r.client.SimpleServiceExists(ctx, "nats", plan.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check nats service existence", "Unable to check nats service existence. "+err.Error())
		return
	}
	if exists {
		resp.Diagnostics.AddAttributeError(path.Root("service_name"), "Nats service already exists", "Nats service already exists")
		return
	}

	var args []string
	if !plan.ConfigOptions.IsNull() {
		args = append(args, "--config-options", plan.ConfigOptions.ValueString())
	}
	err = r.client.SimpleServiceCreate(ctx, "nats", plan.ServiceName.ValueString(), args...)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create nats service", "Unable to create nats service. "+err.Error())
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
func (r *natsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Resource doesn't support Update", "Resource doesn't support Update")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *natsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state natsResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.SimpleServiceExists(ctx, "nats", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check nats service existence", "Unable to check nats service existence. "+err.Error())
		return
	}
	if !exists {
		return
	}

	// Destroy instance
	err = r.client.SimpleServiceDestroy(ctx, "nats", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to destroy service", "Unable to destroy service. "+err.Error())
		return
	}
}

func (r *natsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to service_name attribute
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_name"), req.ID)...)
}
