package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/melbahja/goph"
)

var (
	_ resource.Resource              = &storageResource{}
	_ resource.ResourceWithConfigure = &storageResource{}
)

const hostStoragePrefix = "/var/lib/dokku/data/storage/"

func NewStorageResource() resource.Resource {
	return &storageResource{}
}

type storageResource struct {
	client *goph.Client
}

type storageResourceModel struct {
	AppName   types.String `tfsdk:"app_name"`
	Name      types.String `tfsdk:"name"`
	MountPath types.String `tfsdk:"mount_path"`
}

// Metadata returns the resource type name.
func (r *storageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage"
}

// Configure adds the provider configured client to the resource.
func (r *storageResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *storageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"mount_path": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *storageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state storageResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read storage
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("storage:list %s", state.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read storage",
			"Unable to read storage. "+err.Error(),
		)
		return
	}

	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	found := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		hostpath := strings.TrimSpace(parts[0])
		if hostpath[:len(hostStoragePrefix)] != hostStoragePrefix {
			continue
		}
		if state.Name.ValueString() != hostpath[len(hostStoragePrefix):] {
			continue
		}
		state.MountPath = basetypes.NewStringValue(parts[1])
		found = true
		break
	}
	if !found {
		resp.Diagnostics.AddError("Unable to find storage", "Unable to find storage with name "+state.Name.ValueString())
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
func (r *storageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan storageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Ensure storage
	_, _, err := run(ctx, r.client, fmt.Sprintf("storage:ensure-directory %s", plan.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to ensure storage",
			"Unable to ensure storage. "+err.Error(),
		)
		return
	}

	// Mount storage
	_, _, err = run(ctx, r.client, fmt.Sprintf("storage:mount %s %s:%s", plan.AppName.ValueString(), hostStoragePrefix+plan.Name.ValueString(), plan.MountPath.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to mount storage",
			"Unable to mount storage. "+err.Error(),
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
func (r *storageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan storageResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state storageResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Unable to change app name", "Unable to change app name")
		return
	}
	if plan.Name.ValueString() != state.Name.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Unable to change storage name", "Unable to change storage name")
		return
	}

	// Unmount storage
	_, _, err := run(ctx, r.client, fmt.Sprintf("storage:unmount %s %s:%s", state.AppName.ValueString(), hostStoragePrefix+state.Name.ValueString(), state.MountPath.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to unmount storage",
			"Unable to unmount storage. "+err.Error(),
		)
		return
	}

	// Ensure storage
	_, _, err = run(ctx, r.client, fmt.Sprintf("storage:ensure-directory %s", plan.Name.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to ensure storage",
			"Unable to ensure storage. "+err.Error(),
		)
		return
	}

	// Mount storage
	_, _, err = run(ctx, r.client, fmt.Sprintf("storage:mount %s %s:%s", plan.AppName.ValueString(), hostStoragePrefix+plan.Name.ValueString(), plan.MountPath.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create storage",
			"Unable to create storage. "+err.Error(),
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
func (r *storageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state storageResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Umount storage
	_, _, err := run(ctx, r.client, fmt.Sprintf("storage:unmount %s %s:%s", state.AppName.ValueString(), hostStoragePrefix+state.Name.ValueString(), state.MountPath.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to unmount storage",
			"Unable to unmount storage. "+err.Error(),
		)
		return
	}
}
