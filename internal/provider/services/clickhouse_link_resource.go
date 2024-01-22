package services

import (
	"context"
	"regexp"
	"strings"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &clickhouseLinkResource{}
	_ resource.ResourceWithConfigure   = &clickhouseLinkResource{}
	_ resource.ResourceWithImportState = &clickhouseLinkResource{}
)

func NewClickhouseLinkResource() resource.Resource {
	return &clickhouseLinkResource{}
}

type clickhouseLinkResource struct {
	client *dokkuclient.Client
}

type clickhouseLinkResourceModel struct {
	AppName     types.String `tfsdk:"app_name"`
	ServiceName types.String `tfsdk:"service_name"`
	Alias       types.String `tfsdk:"alias"`
}

// Metadata returns the resource type name.
func (r *clickhouseLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clickhouse_link"
}

// Configure adds the provider configured client to the resource.
func (r *clickhouseLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *clickhouseLinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required:    true,
				Description: "App name to apply link service to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9-]*$`), "invalid app_name"),
				},
			},
			"service_name": schema.StringAttribute{
				Required:    true,
				Description: "Service name to link",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9-]*$`), "invalid service_name"),
				},
			},
			"alias": schema.StringAttribute{
				Optional:    true,
				Description: "Alias is dokku's resource alias to provide as env XXXX_URL",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Z]*$`), "invalid alias"),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *clickhouseLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state clickhouseLinkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.SimpleServiceExists(ctx, "clickhouse", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check clickhouse service existence", "Unable to check clickhouse service existence. "+err.Error())
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Unable to find clickhouse service", "Unable to find clickhouse service")
		return
	}

	// Check link existence
	exists, err = r.client.SimpleServiceLinkExists(ctx, "clickhouse", state.ServiceName.ValueString(), state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check clickhouse link existence", "Unable to check clickhouse link existence. "+err.Error())
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
func (r *clickhouseLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan clickhouseLinkResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.SimpleServiceExists(ctx, "clickhouse", plan.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check clickhouse service existence", "Unable to check clickhouse service existence. "+err.Error())
		return
	}
	if !exists {
		resp.Diagnostics.AddAttributeError(path.Root("service_name"), "Unable to find clickhouse service", "Unable to find clickhouse service")
		return
	}

	// Check link existence
	exists, err = r.client.SimpleServiceLinkExists(ctx, "clickhouse", plan.ServiceName.ValueString(), plan.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check clickhouse link existence", "Unable to check clickhouse link existence. "+err.Error())
		return
	}
	if exists {
		resp.Diagnostics.AddError("Service already linked to app", "Service already linked to app")
		return
	}

	args := make([]string, 0)
	if !plan.Alias.IsNull() {
		args = append(args, dokkuclient.DoubleLineArg("alias", plan.Alias.ValueString()))
	}

	// Create link
	err = r.client.SimpleServiceLinkCreate(ctx, "clickhouse", plan.ServiceName.ValueString(), plan.AppName.ValueString(), args...)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create clickhouse link", "Unable to create clickhouse link. "+err.Error())
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
func (r *clickhouseLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Resource doesn't support Update", "Resource doesn't support Update")
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *clickhouseLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state clickhouseLinkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.SimpleServiceExists(ctx, "clickhouse", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check clickhouse service existence", "Unable to check clickhouse service existence. "+err.Error())
		return
	}
	if !exists {
		return
	}

	// Check link existence
	exists, err = r.client.SimpleServiceLinkExists(ctx, "clickhouse", state.ServiceName.ValueString(), state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check clickhouse link existence", "Unable to check clickhouse link existence. "+err.Error())
		return
	}
	if !exists {
		return
	}

	// Unlink service
	err = r.client.SimpleServiceLinkRemove(ctx, "clickhouse", state.ServiceName.ValueString(), state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to unlink service from app", "Unable to unlink service from app. "+err.Error())
		return
	}
}

func (r *clickhouseLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, " ")
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_name"), parts[1])...)
	if len(parts) == 3 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("alias"), parts[2])...)
	}
}
