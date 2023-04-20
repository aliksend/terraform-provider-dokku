package provider

import (
	"context"
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

// TODO global domains support

var (
	_ resource.Resource              = &domainResource{}
	_ resource.ResourceWithConfigure = &domainResource{}
)

func NewDomainResource() resource.Resource {
	return &domainResource{}
}

type domainResource struct {
	client *goph.Client
}

type domainResourceModel struct {
	AppName types.String `tfsdk:"app"`
	Domains types.Set    `tfsdk:"domains"`
}

// Metadata returns the resource type name.
func (r *domainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domains"
}

// Configure adds the provider configured client to the resource.
func (r *domainResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*goph.Client)
}

// Schema defines the schema for the resource.
func (r *domainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"app": schema.StringAttribute{
				Required: true,
			},
			"domains": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *domainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state domainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read domains
	stdout, _, err := run(ctx, r.client, fmt.Sprintf("domains:report %s", state.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create domain",
			"Unable to create domain. "+err.Error(),
		)
		return
	}

	lines := strings.Split(stdout, "\n")
	var domains []string
	for _, line := range lines {
		parts := strings.Split(line, ":")
		key := strings.TrimSpace(parts[0])
		if key == "Domains app vhosts" {
			domainList := strings.TrimSpace(parts[1])
			if domainList != "" {
				domains = strings.Split(domainList, " ")
			}
		}
	}

	var stateData []attr.Value
	for _, v := range domains {
		stateData = append(stateData, basetypes.NewStringValue(v))
	}
	state.Domains = basetypes.NewSetValueMust(types.StringType, stateData)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *domainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan domainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set domains
	r.setDomains(ctx, plan, &resp.Diagnostics)
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
func (r *domainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan domainResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state domainResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AppName.ValueString() != state.AppName.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("app"), "Unable to change app name", "Unable to change app name")
		return
	}

	// Unset domains
	var domainsToRemove []string
	for _, existing := range state.Domains.Elements() {
		//nolint:forcetypeassert
		existingDomain := existing.(types.String)
		found := false
		for _, planned := range plan.Domains.Elements() {
			//nolint:forcetypeassert
			plannedDomain := planned.(types.String)
			if plannedDomain.ValueString() == existingDomain.ValueString() {
				found = true
				break
			}
		}
		// If exsting domain not present in planned keys then remove
		if !found {
			domainsToRemove = append(domainsToRemove, existingDomain.ValueString())
		}
	}
	if len(domainsToRemove) != 0 {
		_, _, err := run(ctx, r.client, fmt.Sprintf("domains:remove %s %s", state.AppName.ValueString(), strings.Join(domainsToRemove, " ")))
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to create domains",
				"Unable to create domains. "+err.Error(),
			)
			return
		}
	}

	// Set domains
	r.setDomains(ctx, plan, &resp.Diagnostics)
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
func (r *domainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state domainResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Clear domains
	_, _, err := run(ctx, r.client, fmt.Sprintf("domains:clear %s", state.AppName.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete domain",
			"Unable to delete domain. "+err.Error(),
		)
		return
	}
}

func (r *domainResource) setDomains(ctx context.Context, state domainResourceModel, d *diag.Diagnostics) {
	var valuesArr []string
	for _, v := range state.Domains.Elements() {
		//nolint:forcetypeassert
		strValue := v.(types.String)
		valuesArr = append(valuesArr, strValue.ValueString())
	}

	var err error
	if len(valuesArr) == 0 {
		d.AddError("Config must contain at least one value", "Config must contain at least one value")
		return
	}

	_, _, err = run(ctx, r.client, fmt.Sprintf("domains:set %s %s", state.AppName.ValueString(), strings.Join(valuesArr, " ")))
	if err != nil {
		d.AddError(
			"Unable to create domain",
			"Unable to create domain. "+err.Error(),
		)
		return
	}
}
