package services

import (
	"context"
	"fmt"
	"regexp"

	dokkuclient "github.com/aliksend/terraform-provider-dokku/provider/dokku_client"

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
	_ resource.Resource                = &rabbitMQResource{}
	_ resource.ResourceWithConfigure   = &rabbitMQResource{}
	_ resource.ResourceWithImportState = &rabbitMQResource{}
)

func NewRabbitMQResource() resource.Resource {
	return &rabbitMQResource{}
}

type rabbitMQResource struct {
	client *dokkuclient.Client
}

type rabbitMQResourceModel struct {
	ServiceName types.String `tfsdk:"service_name"`
	Image       types.String `tfsdk:"image"`
	Expose      types.String `tfsdk:"expose"`
}

// Metadata returns the resource type name.
func (r *rabbitMQResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rabbitmq"
}

// Configure adds the provider configured client to the resource.
func (r *rabbitMQResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	//nolint:forcetypeassert
	r.client = req.ProviderData.(*dokkuclient.Client)
}

// Schema defines the schema for the resource.
func (r *rabbitMQResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"service_name": schema.StringAttribute{
				Required:    true,
				Description: "Service name to create",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z][a-z0-9-]*$`), "invalid service_name"),
				},
			},
			"image": schema.StringAttribute{
				Optional:    true,
				Description: "Image to use in `image:version` format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^(.+):(.+)$`), "invalid image"),
				},
			},
			"expose": schema.StringAttribute{
				Optional:    true,
				Description: "Port or IP:Port to expose service on",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$|^\d+\.\d+\.\d+\.\d+:\d+$`), "invalid expose"),
				},
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *rabbitMQResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state rabbitMQResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.SimpleServiceExists(ctx, "rabbitmq", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check rabbitMQ service existence", "Unable to check rabbitMQ service existence. "+err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	info, err := r.client.SimpleServiceInfo(ctx, "rabbitmq", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to get rabbitmq service info", "Unable to get rabbitmq service info. "+err.Error())
		return
	}
	infoExposedPorts := info["Exposed ports"]
	if infoExposedPorts != "" && infoExposedPorts != "-" {
		r := regexp.MustCompile(`^\d+->(.+)$`)
		m := r.FindStringSubmatch(infoExposedPorts)
		if len(m) != 2 {
			resp.Diagnostics.AddError("Unsupported format of rabbitmq service Exposed ports", "Unsupported format of rabbitmq service Exposed ports: "+infoExposedPorts)
			return
		}

		state.Expose = basetypes.NewStringValue(m[1])
	} else {
		state.Expose = basetypes.NewStringNull()
	}
	infoVersion := info["Version"]
	state.Image = basetypes.NewStringValue(infoVersion)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *rabbitMQResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan rabbitMQResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create service is not exists
	exists, err := r.client.SimpleServiceExists(ctx, "rabbitmq", plan.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check rabbitMQ service existence", "Unable to check rabbitMQ service existence. "+err.Error())
		return
	}
	if exists {
		resp.Diagnostics.AddAttributeError(path.Root("service_name"), "RabbitMQ service already exists", "RabbitMQ service already exists")
		return
	}

	var args []string

	if !plan.Image.IsNull() {
		r := regexp.MustCompile(`^(.+):(.+)$`)
		m := r.FindStringSubmatch(plan.Image.ValueString())
		if len(m) == 3 {
			args = append(args, fmt.Sprintf("--image %s", m[1]), fmt.Sprintf("--image-version %s", m[2]))
		} else {
			resp.Diagnostics.AddError("Invalid image format", "Invalid image format")
		}
	}

	err = r.client.SimpleServiceCreate(ctx, "rabbitmq", plan.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create rabbitMQ service", "Unable to create rabbitMQ service. "+err.Error())
		return
	}

	if !plan.Expose.IsNull() {
		err := r.client.SimpleServiceExpose(ctx, "rabbitmq", plan.ServiceName.ValueString(), plan.Expose.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to expose rabbitmq service", "Unable to expose rabbitmq service. "+err.Error())
			return
		}
	}

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *rabbitMQResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan rabbitMQResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state rabbitMQResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ServiceName.ValueString() != state.ServiceName.ValueString() {
		resp.Diagnostics.AddError("service_name can't be changed", "service_name can't be changed")
	}

	if !plan.Expose.IsNull() {
		if !plan.Expose.Equal(state.Expose) {
			err := r.client.SimpleServiceUnexpose(ctx, "rabbitmq", state.ServiceName.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Unable to unexpose rabbitmq service", "Unable to unexpose rabbitmq service. "+err.Error())
				return
			}

			err = r.client.SimpleServiceExpose(ctx, "rabbitmq", plan.ServiceName.ValueString(), plan.Expose.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Unable to expose rabbitmq service", "Unable to expose rabbitmq service. "+err.Error())
				return
			}
		}
	} else if !state.Expose.IsNull() {
		err := r.client.SimpleServiceUnexpose(ctx, "rabbitmq", state.ServiceName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to unexpose rabbitmq service", "Unable to unexpose rabbitmq service. "+err.Error())
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
func (r *rabbitMQResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state rabbitMQResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check service existence
	exists, err := r.client.SimpleServiceExists(ctx, "rabbitmq", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to check rabbitMQ service existence", "Unable to check rabbitMQ service existence. "+err.Error())
		return
	}
	if !exists {
		return
	}

	// Destroy instance
	err = r.client.SimpleServiceDestroy(ctx, "rabbitmq", state.ServiceName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to destroy service", "Unable to destroy service. "+err.Error())
		return
	}
}

func (r *rabbitMQResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to service_name attribute
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_name"), req.ID)...)
}
