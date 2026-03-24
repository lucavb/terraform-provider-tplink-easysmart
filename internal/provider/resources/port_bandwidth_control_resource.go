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
	_ resource.Resource                = (*portBandwidthControlResource)(nil)
	_ resource.ResourceWithConfigure   = (*portBandwidthControlResource)(nil)
	_ resource.ResourceWithImportState = (*portBandwidthControlResource)(nil)
)

type portBandwidthControlResource struct {
	client client.Client
}

type portBandwidthControlResourceModel struct {
	ID              types.String `tfsdk:"id"`
	PortID          types.Int64  `tfsdk:"port_id"`
	IngressRateKbps types.Int64  `tfsdk:"ingress_rate_kbps"`
	EgressRateKbps  types.Int64  `tfsdk:"egress_rate_kbps"`
}

func NewPortBandwidthControlResource() resource.Resource {
	return &portBandwidthControlResource{}
}

func (r *portBandwidthControlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_bandwidth_control"
}

func (r *portBandwidthControlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manage one switch port's ingress and egress bandwidth-control rates.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource identifier derived from port_id.",
			},
			"port_id": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Physical switch port number.",
			},
			"ingress_rate_kbps": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Ingress rate limit in Kbps. Set to `0` for unlimited.",
			},
			"egress_rate_kbps": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Egress rate limit in Kbps. Set to `0` for unlimited.",
			},
		},
	}
}

func (r *portBandwidthControlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *portBandwidthControlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portBandwidthControlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, ok := r.apply(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portBandwidthControlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portBandwidthControlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, found, diags := r.readBandwidthControl(ctx, state.PortID.ValueInt64())
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

func (r *portBandwidthControlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan portBandwidthControlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, ok := r.apply(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *portBandwidthControlResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *portBandwidthControlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	portID, ok := parseImportID(req.ID, "port_id", &resp.Diagnostics)
	if !ok {
		return
	}
	if portID < 1 {
		resp.Diagnostics.AddAttributeError(path.Root("port_id"), "Invalid port ID", "port_id must be greater than zero.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("port_id"), portID)...)
}

func (r *portBandwidthControlResource) apply(ctx context.Context, plan portBandwidthControlResourceModel, diags *diag.Diagnostics) (portBandwidthControlResourceModel, bool) {
	if r.client == nil {
		diags.AddError("Unconfigured resource client", "The provider client was not configured.")
		return portBandwidthControlResourceModel{}, false
	}

	portID := plan.PortID.ValueInt64()
	if portID < 1 {
		diags.AddAttributeError(path.Root("port_id"), "Invalid port ID", "port_id must be greater than zero.")
		return portBandwidthControlResourceModel{}, false
	}

	ingress := plan.IngressRateKbps.ValueInt64()
	if ingress < 0 || ingress > 1000000 {
		diags.AddAttributeError(path.Root("ingress_rate_kbps"), "Invalid ingress rate", "ingress_rate_kbps must be in the range 0-1000000.")
		return portBandwidthControlResourceModel{}, false
	}

	egress := plan.EgressRateKbps.ValueInt64()
	if egress < 0 || egress > 1000000 {
		diags.AddAttributeError(path.Root("egress_rate_kbps"), "Invalid egress rate", "egress_rate_kbps must be in the range 0-1000000.")
		return portBandwidthControlResourceModel{}, false
	}

	if err := r.client.SetPortBandwidthControl(ctx, int(portID), int(ingress), int(egress)); err != nil {
		diags.AddError("Unable to apply bandwidth control", err.Error())
		return portBandwidthControlResourceModel{}, false
	}

	refreshed, found, refreshDiags := r.readBandwidthControl(ctx, portID)
	diags.Append(refreshDiags...)
	if diags.HasError() {
		return portBandwidthControlResourceModel{}, false
	}
	if !found {
		diags.AddError("Unable to refresh bandwidth control", fmt.Sprintf("Port %d was not found after apply.", portID))
		return portBandwidthControlResourceModel{}, false
	}

	return refreshed, true
}

func (r *portBandwidthControlResource) readBandwidthControl(ctx context.Context, portID int64) (portBandwidthControlResourceModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	entries, err := r.client.GetPortBandwidthControls(ctx)
	if err != nil {
		diags.AddError("Unable to read bandwidth control settings", err.Error())
		return portBandwidthControlResourceModel{}, false, diags
	}

	for _, entry := range entries {
		if int64(entry.PortID) != portID {
			continue
		}

		return portBandwidthControlResourceModel{
			ID:              stringifyID(portID),
			PortID:          types.Int64Value(portID),
			IngressRateKbps: types.Int64Value(int64(entry.IngressRateKbps)),
			EgressRateKbps:  types.Int64Value(int64(entry.EgressRateKbps)),
		}, true, diags
	}

	return portBandwidthControlResourceModel{}, false, diags
}
