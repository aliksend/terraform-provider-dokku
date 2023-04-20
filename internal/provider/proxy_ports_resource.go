package provider

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/melbahja/goph"
)

var (
	_ resource.Resource              = &proxyPortResource{}
	_ resource.ResourceWithConfigure = &proxyPortResource{}
)

func NewProxyPortsResource() resource.Resource {
	return &proxyPortResource{}
}

type proxyPortResource struct {
	client *goph.Client
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
	r.client = req.ProviderData.(*goph.Client)
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
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("proxy:ports %s", state.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read proxy ports",
			"Unable to read proxy ports. "+err.Error(),
		)
		return
	}

	found := false
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	for _, line := range lines {
		parts := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(line), -1)
		scheme := strings.TrimSpace(parts[0])
		hostPortStr := strings.TrimSpace(parts[1])
		containerPortStr := strings.TrimSpace(parts[2])

		hostPort, err := strconv.ParseInt(hostPortStr, 10, 64)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("host_port"), "Invalid value", "Invalid value. "+err.Error())
		}
		containerPort, err := strconv.ParseInt(containerPortStr, 10, 64)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("container_port"), "Invalid value", "Invalid value. "+err.Error())
		}
		if resp.Diagnostics.HasError() {
			return
		}

		if state.HostPort.ValueInt64() == hostPort {
			state.Scheme = basetypes.NewStringValue(scheme)
			state.ContainerPort = basetypes.NewInt64Value(containerPort)
			found = true
			break
		}
	}
	if !found {
		resp.Diagnostics.AddError("Unable to find port", "Unable to find port")
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
func (r *proxyPortResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan proxyPortResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set proxy port
	_, _, err := run(ctx, r.client, fmt.Sprintf("proxy:ports-add %s %s:%d:%d", plan.AppName.ValueString(), plan.Scheme.ValueString(), plan.HostPort.ValueInt64(), plan.ContainerPort.ValueInt64()))
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

	// Unset proxy port
	_, _, err := run(ctx, r.client, fmt.Sprintf("proxy:ports-remove %s %d", state.AppName.ValueString(), state.HostPort.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to remove proxy port",
			"Unable to remove proxy port. "+err.Error(),
		)
		return
	}

	// Set proxy port
	_, _, err = run(ctx, r.client, fmt.Sprintf("proxy:ports-add %s %s:%d:%d", plan.AppName.ValueString(), plan.Scheme.ValueString(), plan.HostPort.ValueInt64(), plan.ContainerPort.ValueInt64()))
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

	// Unset proxy port
	_, _, err := run(ctx, r.client, fmt.Sprintf("proxy:ports-remove %s %d", state.AppName.ValueString(), state.HostPort.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to remove proxy port",
			"Unable to remove proxy port. "+err.Error(),
		)
		return
	}
}
