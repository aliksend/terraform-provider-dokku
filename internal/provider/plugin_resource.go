package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/melbahja/goph"
)

var (
	_ resource.Resource              = &pluginResource{}
	_ resource.ResourceWithConfigure = &pluginResource{}
	// _ resource.ResourceWithImportState = &pluginResource{} // no way to read URL from
)

func NewPluginResource() resource.Resource {
	return &pluginResource{}
}

type pluginResource struct {
	client *goph.Client
}

type pluginResourceModel struct {
	Name types.String `tfsdk:"name"`
	URL  types.String `tfsdk:"url"`
}

// Metadata returns the resource type name.
func (r *pluginResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin"
}

// Configure adds the provider configured client to the resource.
func (r *pluginResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *pluginResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *pluginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state pluginResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read plugin
	found := r.isPluginInstalled(ctx, state.Name.ValueString(), &resp.Diagnostics)
	if !found {
		resp.Diagnostics.AddError("Unable to find plugin", "Unable to find plugin with name "+state.Name.ValueString())
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
func (r *pluginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan pluginResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Невозможно установить плагин потому что это требует root-прав

	// // Install plugin
	// _, _, err := run(ctx, r.client, fmt.Sprintf("plugin:install %s %s", plan.URL.ValueString(), plan.Name.ValueString()))
	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Unable to install plugin",
	// 		"Unable to install plugin. "+err.Error(),
	// 	)
	// 	return
	// }

	// Поэтому просто проверяем что плагин установлен и, если это не так, то выкидываем ошибку
	found := r.isPluginInstalled(ctx, plan.Name.ValueString(), &resp.Diagnostics)
	if !found {
		resp.Diagnostics.AddError("Plugin not installed", fmt.Sprintf("Plugin not installed. Run `sudo plugin:install %s %s` manually.", plan.URL.ValueString(), plan.Name.ValueString()))
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
func (r *pluginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan pluginResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state pluginResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Name.ValueString() != state.Name.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Unable to change plugin name", "Unable to change plugin name")
		return
	}
	if plan.URL.ValueString() != state.URL.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Unable to change plugin url", "Unable to change plugin url")
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *pluginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state pluginResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Не заставляем удалять плагин

	// // Uninstall plugin
	// _, _, err := run(ctx, r.client, fmt.Sprintf("plugin:uninstall %s", state.Name.ValueString()))
	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Unable to uninstall plugin",
	// 		"Unable to uninstall plugin. "+err.Error(),
	// 	)
	// 	return
	// }
}

func (r *pluginResource) isPluginInstalled(ctx context.Context, pluginNameToFind string, d *diag.Diagnostics) bool {
	stdout, _, err := run(ctx, r.client, "plugin:list")
	if err != nil {
		d.AddError(
			"Unable to read plugin",
			"Unable to read plugin. "+err.Error(),
		)
		return false
	}

	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	found := false
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), " ")
		pluginName := strings.TrimSpace(parts[0])
		if pluginNameToFind != pluginName {
			continue
		}
		found = true
		break
	}
	return found
}
