package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
)

var (
	_ resource.Resource              = (*portSettingResource)(nil)
	_ resource.ResourceWithConfigure = (*portSettingResource)(nil)
)

type portSettingResource struct {
	client client.Client
}

type portSettingResourceModel struct {
	ID                types.String `tfsdk:"id"`
	PortID            types.Int64  `tfsdk:"port_id"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	SpeedConfig       types.Int64  `tfsdk:"speed_config"`
	FlowControlConfig types.Int64  `tfsdk:"flow_control_config"`
}

func NewPortSettingResource() resource.Resource {
	return &portSettingResource{}
}

func (r *portSettingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_setting"
}

func (r *portSettingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manage one switch port's configurable status, speed, and flow control values.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource identifier derived from port_id.",
			},
			"port_id": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Physical switch port number.",
			},
			"enabled": resourceschema.BoolAttribute{
				Required:    true,
				Description: "Administrative enable state for the port.",
			},
			"speed_config": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Configured speed and duplex enum from the device page model. Valid observed values are 1 through 6.",
			},
			"flow_control_config": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Configured flow-control enum from the device page model. Valid values are 0 or 1.",
			},
		},
	}
}

func (r *portSettingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *portSettingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portSettingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, ok := r.apply(ctx, portSettingResourceModel{}, plan, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portSettingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portSettingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, found, diags := r.readPort(ctx, state.PortID.ValueInt64())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *portSettingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state portSettingResourceModel
	var plan portSettingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, ok := r.apply(ctx, state, plan, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
}

func (r *portSettingResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *portSettingResource) apply(ctx context.Context, prior portSettingResourceModel, plan portSettingResourceModel, diags *diag.Diagnostics) (portSettingResourceModel, bool) {
	if r.client == nil {
		diags.AddError("Unconfigured resource client", "The provider client was not configured.")
		return portSettingResourceModel{}, false
	}

	portID := plan.PortID.ValueInt64()
	if portID < 1 {
		diags.AddAttributeError(path.Root("port_id"), "Invalid port ID", "port_id must be greater than zero.")
		return portSettingResourceModel{}, false
	}

	speed := int(plan.SpeedConfig.ValueInt64())
	if speed < 1 || speed > 6 {
		diags.AddAttributeError(path.Root("speed_config"), "Invalid speed_config", "speed_config must be one of the observed device enum values 1-6.")
		return portSettingResourceModel{}, false
	}

	flow := int(plan.FlowControlConfig.ValueInt64())
	if flow < 0 || flow > 1 {
		diags.AddAttributeError(path.Root("flow_control_config"), "Invalid flow_control_config", "flow_control_config must be 0 or 1.")
		return portSettingResourceModel{}, false
	}

	var enabledArg *bool
	var speedArg *int
	var flowArg *int

	enabled := plan.Enabled.ValueBool()
	if prior.ID.IsNull() || prior.Enabled.IsNull() || prior.Enabled.ValueBool() != enabled {
		enabledArg = &enabled
	}
	if prior.ID.IsNull() || prior.SpeedConfig.IsNull() || prior.SpeedConfig.ValueInt64() != plan.SpeedConfig.ValueInt64() {
		speedArg = &speed
	}
	if prior.ID.IsNull() || prior.FlowControlConfig.IsNull() || prior.FlowControlConfig.ValueInt64() != plan.FlowControlConfig.ValueInt64() {
		flowArg = &flow
	}

	if err := r.client.UpdatePortSettings(ctx, int(portID), enabledArg, speedArg, flowArg); err != nil {
		diags.AddError("Unable to apply port settings", err.Error())
		return portSettingResourceModel{}, false
	}

	refreshed, found, refreshDiags := r.readPort(ctx, portID)
	diags.Append(refreshDiags...)
	if diags.HasError() {
		return portSettingResourceModel{}, false
	}
	if !found {
		diags.AddError("Unable to refresh port settings", fmt.Sprintf("Port %d was not found after apply.", portID))
		return portSettingResourceModel{}, false
	}

	return refreshed, true
}

func (r *portSettingResource) readPort(ctx context.Context, portID int64) (portSettingResourceModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	ports, err := r.client.GetPorts(ctx)
	if err != nil {
		diags.AddError("Unable to read ports", err.Error())
		return portSettingResourceModel{}, false, diags
	}

	for _, port := range ports {
		if int64(port.ID) != portID {
			continue
		}
		return portSettingResourceModel{
			ID:                stringifyID(portID),
			PortID:            types.Int64Value(portID),
			Enabled:           types.BoolValue(port.Enabled),
			SpeedConfig:       types.Int64Value(int64(port.SpeedConfig)),
			FlowControlConfig: types.Int64Value(int64(port.FlowControlConfig)),
		}, true, diags
	}

	return portSettingResourceModel{}, false, diags
}
