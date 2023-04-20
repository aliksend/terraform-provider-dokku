package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/melbahja/goph"
)

var (
	_ resource.Resource              = &configResource{}
	_ resource.ResourceWithConfigure = &configResource{}
)

func NewConfigResource() resource.Resource {
	return &configResource{}
}

type configResource struct {
	client *goph.Client
}

type configResourceModel struct {
	AppName types.String `tfsdk:"app"`
	Data    types.Map    `tfsdk:"data"`
}

// Metadata returns the resource type name.
func (r *configResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

// Configure adds the provider configured client to the resource.
func (r *configResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *configResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app": schema.StringAttribute{
				Required: true,
			},
			"data": schema.MapAttribute{
				ElementType: types.StringType,
				Required:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *configResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state configResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read config
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("config:export --format=json %s", state.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read config",
			"Unable to read config. "+err.Error(),
		)
		return
	}

	var data map[string]string
	err = json.Unmarshal([]byte(stdout), &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to parse config",
			"Unable to parse config. "+err.Error(),
		)
		return
	}

	stateData := make(map[string]attr.Value)
	for k, v := range data {
		stateData[k] = basetypes.NewStringValue(v)
	}
	state.Data = basetypes.NewMapValueMust(types.StringType, stateData)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *configResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan configResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set config
	r.setConfig(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
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
func (r *configResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan configResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state configResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app"), "Unable to change app name", "Unable to change app name")
		return
	}

	// Unset config
	var keysToUnset []string
	for existingKey := range state.Data.Elements() {
		found := false
		for plannedKey := range plan.Data.Elements() {
			if plannedKey == existingKey {
				found = true
				break
			}
		}
		// If exsting key not present in planned keys then unset
		if !found {
			keysToUnset = append(keysToUnset, existingKey)
		}
	}
	if len(keysToUnset) != 0 {
		_, _, err := run(ctx, r.client, fmt.Sprintf("config:unset %s %s", state.AppName.ValueString(), strings.Join(keysToUnset, " ")))
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to create config",
				"Unable to create config. "+err.Error(),
			)
			return
		}
	}

	// Set config
	r.setConfig(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *configResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state configResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Clear config
	_, _, err := run(ctx, r.client, fmt.Sprintf("config:clear %s", state.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete config",
			"Unable to delete config. "+err.Error(),
		)
		return
	}
}

func (r *configResource) setConfig(ctx context.Context, state configResourceModel, d *diag.Diagnostics) {
	var valuesArr []string
	for k, v := range state.Data.Elements() {
		//nolint:forcetypeassert
		strValue := v.(types.String)
		valuesArr = append(valuesArr, fmt.Sprintf("%s=%q", k, base64.StdEncoding.EncodeToString([]byte(strValue.ValueString()))))
	}

	var err error
	if len(valuesArr) == 0 {
		d.AddError("Config must contain at least one value", "Config must contain at least one value")
		return
	}

	// TODO --no-restart ?
	_, _, err = run(ctx, r.client, fmt.Sprintf("config:set --encoded %s %s", state.AppName.ValueString(), strings.Join(valuesArr, " ")))
	if err != nil {
		d.AddError(
			"Unable to create config",
			"Unable to create config. "+err.Error(),
		)
		return
	}
}
