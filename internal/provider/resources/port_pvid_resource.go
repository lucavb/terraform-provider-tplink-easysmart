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
	_ resource.Resource                = (*portPVIDResource)(nil)
	_ resource.ResourceWithConfigure   = (*portPVIDResource)(nil)
	_ resource.ResourceWithImportState = (*portPVIDResource)(nil)
)

type portPVIDResource struct {
	client client.Client
}

type portPVIDResourceModel struct {
	ID     types.String `tfsdk:"id"`
	PortID types.Int64  `tfsdk:"port_id"`
	PVID   types.Int64  `tfsdk:"pvid"`
}

func NewPortPVIDResource() resource.Resource {
	return &portPVIDResource{}
}

func (r *portPVIDResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_pvid"
}

func (r *portPVIDResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manage the 802.1Q PVID for one switch port.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource identifier derived from port_id.",
			},
			"port_id": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Physical switch port number.",
			},
			"pvid": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Existing VLAN ID assigned as the port PVID. The VLAN must already exist on the switch.",
			},
		},
	}
}

func (r *portPVIDResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *portPVIDResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portPVIDResourceModel
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

func (r *portPVIDResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portPVIDResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, found, diags := r.readPortPVID(ctx, state.PortID.ValueInt64())
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

func (r *portPVIDResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan portPVIDResourceModel
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

func (r *portPVIDResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *portPVIDResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *portPVIDResource) apply(ctx context.Context, plan portPVIDResourceModel, diags *diag.Diagnostics) (portPVIDResourceModel, bool) {
	if r.client == nil {
		diags.AddError("Unconfigured resource client", "The provider client was not configured.")
		return portPVIDResourceModel{}, false
	}

	portID := plan.PortID.ValueInt64()
	if portID < 1 {
		diags.AddAttributeError(path.Root("port_id"), "Invalid port ID", "port_id must be greater than zero.")
		return portPVIDResourceModel{}, false
	}

	pvid := plan.PVID.ValueInt64()
	if pvid < 1 || pvid > 4094 {
		diags.AddAttributeError(path.Root("pvid"), "Invalid PVID", "pvid must be in the range 1-4094.")
		return portPVIDResourceModel{}, false
	}

	table, err := r.client.GetVLANs(ctx)
	if err != nil {
		diags.AddError("Unable to validate VLAN table", err.Error())
		return portPVIDResourceModel{}, false
	}

	foundVLAN := false
	for _, vlan := range table.VLANs {
		if int64(vlan.ID) == pvid {
			foundVLAN = true
			break
		}
	}
	if !foundVLAN {
		diags.AddError("Missing VLAN for PVID", fmt.Sprintf("VLAN %d does not exist on the switch.", pvid))
		return portPVIDResourceModel{}, false
	}

	if err := r.client.SetPortPVID(ctx, int(portID), int(pvid)); err != nil {
		diags.AddError("Unable to apply port PVID", err.Error())
		return portPVIDResourceModel{}, false
	}

	refreshed, found, refreshDiags := r.readPortPVID(ctx, portID)
	diags.Append(refreshDiags...)
	if diags.HasError() {
		return portPVIDResourceModel{}, false
	}
	if !found {
		diags.AddError("Unable to refresh port PVID", fmt.Sprintf("Port %d was not found after apply.", portID))
		return portPVIDResourceModel{}, false
	}

	return refreshed, true
}

func (r *portPVIDResource) readPortPVID(ctx context.Context, portID int64) (portPVIDResourceModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	pvids, err := r.client.GetPVIDs(ctx)
	if err != nil {
		diags.AddError("Unable to read port PVIDs", err.Error())
		return portPVIDResourceModel{}, false, diags
	}

	for _, entry := range pvids {
		if int64(entry.PortID) != portID {
			continue
		}
		return portPVIDResourceModel{
			ID:     stringifyID(portID),
			PortID: types.Int64Value(portID),
			PVID:   types.Int64Value(int64(entry.PVID)),
		}, true, diags
	}

	return portPVIDResourceModel{}, false, diags
}
