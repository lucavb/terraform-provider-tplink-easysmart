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
	_ resource.Resource                = (*portQoSPriorityResource)(nil)
	_ resource.ResourceWithConfigure   = (*portQoSPriorityResource)(nil)
	_ resource.ResourceWithImportState = (*portQoSPriorityResource)(nil)
)

type portQoSPriorityResource struct {
	client client.Client
}

type portQoSPriorityResourceModel struct {
	ID       types.String `tfsdk:"id"`
	PortID   types.Int64  `tfsdk:"port_id"`
	Priority types.Int64  `tfsdk:"priority"`
}

func NewPortQoSPriorityResource() resource.Resource {
	return &portQoSPriorityResource{}
}

func (r *portQoSPriorityResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_qos_priority"
}

func (r *portQoSPriorityResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manage one switch port's port-based QoS priority queue.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource identifier derived from port_id.",
			},
			"port_id": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Physical switch port number.",
			},
			"priority": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Port-based priority queue in the range 1-4.",
			},
		},
	}
}

func (r *portQoSPriorityResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *portQoSPriorityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portQoSPriorityResourceModel
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

func (r *portQoSPriorityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portQoSPriorityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, found, diags := r.readPriority(ctx, state.PortID.ValueInt64())
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

func (r *portQoSPriorityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan portQoSPriorityResourceModel
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

func (r *portQoSPriorityResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *portQoSPriorityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *portQoSPriorityResource) apply(ctx context.Context, plan portQoSPriorityResourceModel, diags *diag.Diagnostics) (portQoSPriorityResourceModel, bool) {
	if r.client == nil {
		diags.AddError("Unconfigured resource client", "The provider client was not configured.")
		return portQoSPriorityResourceModel{}, false
	}

	portID := plan.PortID.ValueInt64()
	if portID < 1 {
		diags.AddAttributeError(path.Root("port_id"), "Invalid port ID", "port_id must be greater than zero.")
		return portQoSPriorityResourceModel{}, false
	}

	priority := plan.Priority.ValueInt64()
	if priority < 1 || priority > 4 {
		diags.AddAttributeError(path.Root("priority"), "Invalid priority", "priority must be in the range 1-4.")
		return portQoSPriorityResourceModel{}, false
	}

	if err := r.client.SetPortQoSPriority(ctx, int(portID), int(priority)); err != nil {
		diags.AddError("Unable to apply port QoS priority", err.Error())
		return portQoSPriorityResourceModel{}, false
	}

	refreshed, found, refreshDiags := r.readPriority(ctx, portID)
	diags.Append(refreshDiags...)
	if diags.HasError() {
		return portQoSPriorityResourceModel{}, false
	}
	if !found {
		diags.AddError("Unable to refresh port QoS priority", fmt.Sprintf("Port %d was not found after apply.", portID))
		return portQoSPriorityResourceModel{}, false
	}

	return refreshed, true
}

func (r *portQoSPriorityResource) readPriority(ctx context.Context, portID int64) (portQoSPriorityResourceModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	priorities, err := r.client.GetPortQoSPriorities(ctx)
	if err != nil {
		diags.AddError("Unable to read port QoS priorities", err.Error())
		return portQoSPriorityResourceModel{}, false, diags
	}

	for _, entry := range priorities {
		if int64(entry.PortID) != portID {
			continue
		}

		return portQoSPriorityResourceModel{
			ID:       stringifyID(portID),
			PortID:   types.Int64Value(portID),
			Priority: types.Int64Value(int64(entry.Priority)),
		}, true, diags
	}

	return portQoSPriorityResourceModel{}, false, diags
}
