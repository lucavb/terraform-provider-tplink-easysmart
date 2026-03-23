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
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/normalize"
)

var (
	_ resource.Resource                = (*vlan8021qResource)(nil)
	_ resource.ResourceWithConfigure   = (*vlan8021qResource)(nil)
	_ resource.ResourceWithImportState = (*vlan8021qResource)(nil)
)

type vlan8021qResource struct {
	client client.Client
}

type vlan8021qResourceModel struct {
	ID            types.String `tfsdk:"id"`
	VLANID        types.Int64  `tfsdk:"vlan_id"`
	Name          types.String `tfsdk:"name"`
	TaggedPorts   types.Set    `tfsdk:"tagged_ports"`
	UntaggedPorts types.Set    `tfsdk:"untagged_ports"`
}

func NewVLAN8021QResource() resource.Resource {
	return &vlan8021qResource{}
}

func (r *vlan8021qResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vlan_8021q"
}

func (r *vlan8021qResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manage one 802.1Q VLAN entry on the switch.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource identifier derived from vlan_id.",
			},
			"vlan_id": resourceschema.Int64Attribute{
				Required:    true,
				Description: "VLAN ID in the range 1-4094.",
			},
			"name": resourceschema.StringAttribute{
				Required:    true,
				Description: "VLAN display name.",
			},
			"tagged_ports": resourceschema.SetAttribute{
				Required:    true,
				ElementType: types.Int64Type,
				Description: "Ports that should be tagged members of the VLAN.",
			},
			"untagged_ports": resourceschema.SetAttribute{
				Required:    true,
				ElementType: types.Int64Type,
				Description: "Ports that should be untagged members of the VLAN.",
			},
		},
	}
}

func (r *vlan8021qResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *vlan8021qResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vlan8021qResourceModel
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

func (r *vlan8021qResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vlan8021qResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, found, diags := r.readVLAN(ctx, state.VLANID.ValueInt64())
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

func (r *vlan8021qResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vlan8021qResourceModel
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

func (r *vlan8021qResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vlan8021qResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteVLAN(ctx, int(state.VLANID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Unable to delete VLAN", err.Error())
	}
}

func (r *vlan8021qResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	vlanID, ok := parseImportID(req.ID, "vlan_id", &resp.Diagnostics)
	if !ok {
		return
	}
	if vlanID < 1 || vlanID > 4094 {
		resp.Diagnostics.AddAttributeError(path.Root("vlan_id"), "Invalid VLAN ID", "vlan_id must be in the range 1-4094.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vlan_id"), vlanID)...)
}

func (r *vlan8021qResource) apply(ctx context.Context, plan vlan8021qResourceModel, diags *diag.Diagnostics) (vlan8021qResourceModel, bool) {
	if r.client == nil {
		diags.AddError("Unconfigured resource client", "The provider client was not configured.")
		return vlan8021qResourceModel{}, false
	}

	vlanID := plan.VLANID.ValueInt64()
	if vlanID < 1 || vlanID > 4094 {
		diags.AddAttributeError(path.Root("vlan_id"), "Invalid VLAN ID", "vlan_id must be in the range 1-4094.")
		return vlan8021qResourceModel{}, false
	}

	taggedPorts, taggedDiags := intsFromSet(ctx, plan.TaggedPorts)
	diags.Append(taggedDiags...)
	untaggedPorts, untaggedDiags := intsFromSet(ctx, plan.UntaggedPorts)
	diags.Append(untaggedDiags...)
	if diags.HasError() {
		return vlan8021qResourceModel{}, false
	}

	if hasOverlappingPorts(taggedPorts, untaggedPorts) {
		diags.AddError("Invalid VLAN membership", "tagged_ports and untagged_ports must not overlap.")
		return vlan8021qResourceModel{}, false
	}

	if err := r.client.UpsertVLAN(ctx, int(vlanID), plan.Name.ValueString(), taggedPorts, untaggedPorts); err != nil {
		diags.AddError("Unable to apply VLAN", err.Error())
		return vlan8021qResourceModel{}, false
	}

	refreshed, found, refreshDiags := r.readVLAN(ctx, vlanID)
	diags.Append(refreshDiags...)
	if diags.HasError() {
		return vlan8021qResourceModel{}, false
	}
	if !found {
		diags.AddError("Unable to refresh VLAN", fmt.Sprintf("VLAN %d was not found after apply.", vlanID))
		return vlan8021qResourceModel{}, false
	}

	return refreshed, true
}

func (r *vlan8021qResource) readVLAN(ctx context.Context, vlanID int64) (vlan8021qResourceModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	table, err := r.client.GetVLANs(ctx)
	if err != nil {
		diags.AddError("Unable to read VLAN table", err.Error())
		return vlan8021qResourceModel{}, false, diags
	}

	for _, vlan := range table.VLANs {
		if int64(vlan.ID) != vlanID {
			continue
		}

		taggedSet, taggedDiags := int64SetFromInts(ctx, normalize.NormalizePorts(vlan.TaggedPorts))
		diags.Append(taggedDiags...)
		untaggedSet, untaggedDiags := int64SetFromInts(ctx, normalize.NormalizePorts(vlan.UntaggedPorts))
		diags.Append(untaggedDiags...)

		return vlan8021qResourceModel{
			ID:            stringifyID(vlanID),
			VLANID:        types.Int64Value(vlanID),
			Name:          types.StringValue(vlan.Name),
			TaggedPorts:   taggedSet,
			UntaggedPorts: untaggedSet,
		}, true, diags
	}

	return vlan8021qResourceModel{}, false, diags
}

func hasOverlappingPorts(left []int, right []int) bool {
	for _, port := range left {
		for _, candidate := range right {
			if port == candidate {
				return true
			}
		}
	}
	return false
}
