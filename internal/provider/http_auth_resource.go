package provider

import (
	"context"
	"regexp"

	dokkuclient "terraform-provider-dokku/internal/provider/dokku_client"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
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
	_ resource.Resource                = &httpAuthResource{}
	_ resource.ResourceWithConfigure   = &httpAuthResource{}
	_ resource.ResourceWithImportState = &httpAuthResource{}
)

func NewHttpAuthResource() resource.Resource {
	return &httpAuthResource{}
}

type httpAuthResource struct {
	client *dokkuclient.Client
}

type httpAuthResourceModel struct {
	AppName types.String         `tfsdk:"app_name"`
	Users   map[string]userModel `tfsdk:"users"`
}

type userModel struct {
	Password types.String `tfsdk:"password"`
}

// Metadata returns the resource type name.
func (r *httpAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_http_auth"
}

// Configure adds the provider configured client to the resource.
func (r *httpAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *httpAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9-]*$`), "invalid app_name"),
				},
			},
			"users": schema.MapNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"password": schema.StringAttribute{
							Required:  true,
							Sensitive: true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
					},
				},
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.LengthAtLeast(1)),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *httpAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state httpAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check http auth enabled
	enabled, existingUsers, err := r.client.HttpAuthReport(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check http auth enabled", "Unable to check http auth enabled. "+err.Error())
		return
	}
	if !enabled {
		resp.State.RemoveResource(ctx)
		return
	}

	// Update users list
	users := make(map[string]userModel)
	for _, u := range existingUsers {
		password := ""
		if p, ok := state.Users[u]; ok {
			password = p.Password.ValueString()
		}
		users[u] = userModel{
			Password: basetypes.NewStringValue(password),
		}
	}
	state.Users = users

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *httpAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan httpAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	enabled, existingUsers, err := r.client.HttpAuthReport(ctx, plan.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check http auth enabled", "Unable to check http auth enabled. "+err.Error())
		return
	}
	if enabled {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "Http auth already enabled", "Http auth already enabled")
		return
	}

	err = r.client.HttpAuthEnable(ctx, plan.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to enable http-auth", "Unable to enable http-auth. "+err.Error())
		return
	}

	// remove all users if present
	for _, u := range existingUsers {
		err := r.client.HttpAuthRemoveUser(ctx, plan.AppName.ValueString(), u)
		if err != nil {
			resp.Diagnostics.AddError("Unable to remove user", "Unable to remove user. "+err.Error())
		}
	}

	for user, userData := range plan.Users {
		err := r.client.HttpAuthAddUser(ctx, plan.AppName.ValueString(), user, userData.Password.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to add http-auth user", "Unable to add http-auth user. "+err.Error())
		}
	}

	if resp.Diagnostics.HasError() {
		err := r.client.HttpAuthDisable(ctx, plan.AppName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to disable http-auth", "Unable to disable http-auth. "+err.Error())
		}
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
func (r *httpAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan httpAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state httpAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app_name"), "App name can't be changed", "App name can't be changed")
		return
	}

	for existingUser := range state.Users {
		found := false
		for plannedUser := range plan.Users {
			if plannedUser == existingUser {
				found = true
				break
			}
		}
		if !found {
			err := r.client.HttpAuthRemoveUser(ctx, state.AppName.ValueString(), existingUser)
			if err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("users").AtMapKey(existingUser), "Unable to remove user", "Unable to remove user. "+err.Error())
				return
			}
		}
	}

	for user, userData := range plan.Users {
		err := r.client.HttpAuthAddUser(ctx, plan.AppName.ValueString(), user, userData.Password.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to add http-auth user", "Unable to add http-auth user. "+err.Error())
			return
		}
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
func (r *httpAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state httpAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.HttpAuthDisable(ctx, state.AppName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to disable http-auth", "Unable to disable http-auth. "+err.Error())
		return
	}
}

func (r *httpAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to app_name attribute
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app_name"), req.ID)...)
}
