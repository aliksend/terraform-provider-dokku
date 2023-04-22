package provider

import (
	"context"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource              = &proxyPortResource{}
	_ resource.ResourceWithConfigure = &proxyPortResource{}
	// _ resource.ResourceWithImportState = &proxyPortResource{} // app_name + [scheme +] host_port
)

func NewProxyPortsResource() resource.Resource {
	return &proxyPortResource{}
}

type proxyPortResource struct {
	client *dokkuclient.Client
}

type proxyPortResourceModel struct {
	AppName       types.String `tfsdk:"app_name"`
	Scheme        types.String `tfsdk:"scheme"`
	HostPort      types.Int64  `tfsdk:"host_port"`
	ContainerPort types.Int64  `tfsdk:"container_port"`
}

// Metadata returns the resource type name.
func (r *proxyPortResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_proxy_port"
}

// Configure adds the provider configured client to the resource.
func (r *proxyPortResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *proxyPortResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
			},
			"scheme": schema.StringAttribute{
				Required: true,
			},
			"host_port": schema.Int64Attribute{
				Required: true,
			},
			"container_port": schema.Int64Attribute{
				Required: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *proxyPortResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state proxyPortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read proxy ports
	exists, scheme, containerPort, err := r.client.ProxyPortExists(ctx, state.AppName.ValueString(), state.HostPort.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read proxy ports",
			"Unable to read proxy ports. "+err.Error(),
		)
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Scheme = basetypes.NewStringValue(scheme)
	state.ContainerPort = basetypes.NewInt64Value(containerPort)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *proxyPortResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan proxyPortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read proxy ports
	exists, _, _, err := r.client.ProxyPortExists(ctx, plan.AppName.ValueString(), plan.HostPort.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read proxy ports",
			"Unable to read proxy ports. "+err.Error(),
		)
		return
	}
	if exists {
		resp.Diagnostics.AddError("Port already assigned for this app", "Port already assigned for this app")
		return
	}

	// Set proxy port
	err = r.client.ProxyPortAdd(ctx, plan.AppName.ValueString(), plan.Scheme.ValueString(), plan.HostPort.ValueInt64(), plan.ContainerPort.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to add proxy port",
			"Unable to add proxy port. "+err.Error(),
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
func (r *proxyPortResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan proxyPortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state proxyPortResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to change app name", "Unable to change app name")
		return
	}

	// Read proxy ports
	exists, _, _, err := r.client.ProxyPortExists(ctx, state.AppName.ValueString(), state.HostPort.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read proxy ports",
			"Unable to read proxy ports. "+err.Error(),
		)
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Proxy port not exists", "Proxy port not exists")
		return
	}

	// Unset proxy port
	err = r.client.ProxyPortRemove(ctx, state.AppName.ValueString(), state.HostPort.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to remove proxy port",
			"Unable to remove proxy port. "+err.Error(),
		)
		return
	}

	// Set proxy port
	err = r.client.ProxyPortAdd(ctx, plan.AppName.ValueString(), plan.Scheme.ValueString(), plan.HostPort.ValueInt64(), plan.ContainerPort.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to add proxy port",
			"Unable to add proxy port. "+err.Error(),
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
func (r *proxyPortResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state proxyPortResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Read proxy ports
	exists, _, _, err := r.client.ProxyPortExists(ctx, state.AppName.ValueString(), state.HostPort.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read proxy ports",
			"Unable to read proxy ports. "+err.Error(),
		)
		return
	}
	if !exists {
		return
	}

	// Unset proxy port
	err = r.client.ProxyPortRemove(ctx, state.AppName.ValueString(), state.HostPort.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to remove proxy port",
			"Unable to remove proxy port. "+err.Error(),
		)
		return
	}
}
