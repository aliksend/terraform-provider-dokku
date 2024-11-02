package provider

import (
	"context"
	"regexp"
	"strings"

	dokkuclient "github.com/aliksend/terraform-provider-dokku/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource              = &nginxConfigResource{}
	_ resource.ResourceWithConfigure = &nginxConfigResource{}
)

func NewNginxConfigResource() resource.Resource {
	return &nginxConfigResource{}
}

type nginxConfigResource struct {
	client *dokkuclient.Client
}

type nginxConfigResourceModel struct {
	AppName types.String            `tfsdk:"app_name"`
	Global  types.Bool              `tfsdk:"global"`
	Config  map[string]types.String `tfsdk:"config"`
}

// Metadata returns the resource type name.
func (r *nginxConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nginx_config"
}

// Configure adds the provider configured client to the resource.
func (r *nginxConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *nginxConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: strings.Join([]string{
			"dokku nginx config",
			"Configure nginx for app or globally",
			"https://dokku.com/docs/networking/proxies/nginx/",
		}, "\n  "),
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Optional:    true,
				Description: "App name to apply nginx config to. You must specify either app_name or global",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9-]*$`), "invalid app_name"),
				},
			},
			"global": schema.BoolAttribute{
				Optional:    true,
				Description: "Apply nginx config globaly to server. You must specify either app_name or global",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.MapAttribute{
				Required:    true,
				Description: "Nginx config",
				ElementType: types.StringType,
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.LengthAtLeast(1)),
					mapvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
		},
	}
}

func (r nginxConfigResourceModel) appName() (appName string, ok bool) {
	if !r.Global.IsNull() {
		if !r.AppName.IsNull() {
			return "", false
		}

		return "--global", true
	}

	if !r.AppName.IsNull() {
		return r.AppName.ValueString(), true
	}

	return "", false
}

// Read refreshes the Terraform state with the latest data.
func (r *nginxConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state nginxConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check
	appName, ok := state.appName()
	if !ok {
		msg := "You must specify either app_name or global"
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), msg, msg)
		resp.Diagnostics.AddAttributeError(path.Root("global"), msg, msg)
		return
	}

	// Read
	for property := range state.Config {
		value, err := r.client.NginxConfigGetValue(ctx, appName, property)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read nginxConfig property"+property, "Unable to read nginxConfig property"+property+". "+err.Error())
			return
		}
		state.Config[property] = basetypes.NewStringValue(value)
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *nginxConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan nginxConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check
	appName, ok := plan.appName()
	if !ok {
		msg := "You must specify either app_name or global"
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), msg, msg)
		resp.Diagnostics.AddAttributeError(path.Root("global"), msg, msg)
		return
	}

	// Create
	for property := range plan.Config {
		err := r.client.NginxConfigSetValue(ctx, appName, property, plan.Config[property].ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to set nginxConfig property", "Unable to set nginxConfig property. "+err.Error())
			return
		}
	}

	err := r.client.ProxyBuildConfig(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError("Unable to rebuild nginxConfig for app", "Unable to rebuild nginxConfig for app. "+err.Error())
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
func (r *nginxConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan nginxConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state nginxConfigResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check
	appName, ok := plan.appName()
	if !ok {
		msg := "You must specify either app_name or global"
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), msg, msg)
		resp.Diagnostics.AddAttributeError(path.Root("global"), msg, msg)
		return
	}

	// Update
	var updatedProperties []string
	for property := range plan.Config {
		err := r.client.NginxConfigSetValue(ctx, appName, property, plan.Config[property].ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to set nginxConfig property", "Unable to set nginxConfig property. "+err.Error())
			return
		}
		updatedProperties = append(updatedProperties, property)
	}
	for property := range state.Config {
		presentInPlan := false
		for _, p := range updatedProperties {
			if property == p {
				presentInPlan = true
				break
			}
		}
		if !presentInPlan {
			err := r.client.NginxConfigResetValue(ctx, appName, property)
			if err != nil {
				resp.Diagnostics.AddError("Unable to reset nginxConfig property", "Unable to reset nginxConfig property. "+err.Error())
				return
			}
		}
	}

	err := r.client.ProxyBuildConfig(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError("Unable to rebuild nginxConfig for app", "Unable to rebuild nginxConfig for app. "+err.Error())
		return
	}

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
func (r *nginxConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state nginxConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check
	appName, ok := state.appName()
	if !ok {
		msg := "You must specify either app_name or global"
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), msg, msg)
		resp.Diagnostics.AddAttributeError(path.Root("global"), msg, msg)
		return
	}

	for property := range state.Config {
		err := r.client.NginxConfigResetValue(ctx, appName, property)
		if err != nil {
			resp.Diagnostics.AddError("Unable to reset nginxConfig property", "Unable to reset nginxConfig property. "+err.Error())
			return
		}
	}

	err := r.client.ProxyBuildConfig(ctx, appName)
	if err != nil {
		resp.Diagnostics.AddError("Unable to rebuild nginxConfig for app", "Unable to rebuild nginxConfig for app. "+err.Error())
		return
	}
}
