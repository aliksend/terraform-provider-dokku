package provider

import (
	"context"
	"strings"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &storageResource{}
	_ resource.ResourceWithConfigure   = &storageResource{}
	_ resource.ResourceWithImportState = &storageResource{}
)

func NewStorageResource() resource.Resource {
	return &storageResource{}
}

type storageResource struct {
	client *dokkuclient.Client
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
	r.client = req.ProviderData.(*dokkuclient.Client)
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
	exists, mountPath, err := r.client.StorageExists(ctx, state.AppName.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read storage",
			"Unable to read storage. "+err.Error(),
		)
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	state.MountPath = basetypes.NewStringValue(mountPath)

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

	exists, _, err := r.client.StorageExists(ctx, plan.AppName.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read storage",
			"Unable to read storage. "+err.Error(),
		)
		return
	}
	if exists {
		resp.Diagnostics.AddError("Storage already mounted", "Storage already mounted")
		return
	}

	// Ensure storage
	err = r.client.StorageEnsure(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to ensure storage",
			"Unable to ensure storage. "+err.Error(),
		)
		return
	}

	// Mount storage
	err = r.client.StorageMount(ctx, plan.AppName.ValueString(), plan.Name.ValueString(), plan.MountPath.ValueString())
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

	exists, _, err := r.client.StorageExists(ctx, state.AppName.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read storage",
			"Unable to read storage. "+err.Error(),
		)
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Storage not mounted", "Storage not mounted")
		return
	}

	// Unmount storage
	err = r.client.StorageUnmount(ctx, state.AppName.ValueString(), state.Name.ValueString(), state.MountPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to unmount storage",
			"Unable to unmount storage. "+err.Error(),
		)
		return
	}

	// Ensure storage
	err = r.client.StorageEnsure(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to ensure storage",
			"Unable to ensure storage. "+err.Error(),
		)
		return
	}

	// Mount storage
	err = r.client.StorageMount(ctx, plan.AppName.ValueString(), plan.Name.ValueString(), plan.MountPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to mount storage",
			"Unable to mount storage. "+err.Error(),
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

	exists, _, err := r.client.StorageExists(ctx, state.AppName.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read storage",
			"Unable to read storage. "+err.Error(),
		)
		return
	}
	if !exists {
		return
	}

	// Umount storage
	err = r.client.StorageUnmount(ctx, state.AppName.ValueString(), state.Name.ValueString(), state.MountPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to unmount storage",
			"Unable to unmount storage. "+err.Error(),
		)
		return
	}
}

func (r *storageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, " ")
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_name"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
}
