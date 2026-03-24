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
	_ resource.Resource                = (*lagResource)(nil)
	_ resource.ResourceWithConfigure   = (*lagResource)(nil)
	_ resource.ResourceWithImportState = (*lagResource)(nil)
)

type lagResource struct {
	client client.Client
}

type lagResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.Int64  `tfsdk:"group_id"`
	Ports   types.Set    `tfsdk:"ports"`
}

func NewLAGResource() resource.Resource {
	return &lagResource{}
}

func (r *lagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lag"
}

func (r *lagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manage one static LAG group and its member ports.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource identifier derived from group_id.",
			},
			"group_id": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Static LAG group number.",
			},
			"ports": resourceschema.SetAttribute{
				Required:    true,
				ElementType: types.Int64Type,
				Description: "Ports assigned to the LAG group.",
			},
		},
	}
}

func (r *lagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *lagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan lagResourceModel
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

func (r *lagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state lagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, found, diags := r.readLAG(ctx, state.GroupID.ValueInt64())
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

func (r *lagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan lagResourceModel
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

func (r *lagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state lagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client == nil {
		resp.Diagnostics.AddError("Unconfigured resource client", "The provider client was not configured.")
		return
	}

	if err := r.client.DeleteLAG(ctx, int(state.GroupID.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Unable to delete LAG", err.Error())
	}
}

func (r *lagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	groupID, ok := parseImportID(req.ID, "group_id", &resp.Diagnostics)
	if !ok {
		return
	}
	if groupID < 1 {
		resp.Diagnostics.AddAttributeError(path.Root("group_id"), "Invalid group ID", "group_id must be greater than zero.")
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
}

func (r *lagResource) apply(ctx context.Context, plan lagResourceModel, diags *diag.Diagnostics) (lagResourceModel, bool) {
	if r.client == nil {
		diags.AddError("Unconfigured resource client", "The provider client was not configured.")
		return lagResourceModel{}, false
	}

	groupID := plan.GroupID.ValueInt64()
	if groupID < 1 {
		diags.AddAttributeError(path.Root("group_id"), "Invalid group ID", "group_id must be greater than zero.")
		return lagResourceModel{}, false
	}

	ports, portDiags := intsFromSet(ctx, plan.Ports)
	diags.Append(portDiags...)
	if diags.HasError() {
		return lagResourceModel{}, false
	}
	ports = normalize.NormalizePorts(ports)

	if len(ports) < 2 {
		diags.AddAttributeError(path.Root("ports"), "Invalid LAG ports", "At least two ports must be selected for a LAG.")
		return lagResourceModel{}, false
	}

	if err := r.client.UpsertLAG(ctx, int(groupID), ports); err != nil {
		diags.AddError("Unable to apply LAG settings", err.Error())
		return lagResourceModel{}, false
	}

	refreshed, found, refreshDiags := r.readLAG(ctx, groupID)
	diags.Append(refreshDiags...)
	if diags.HasError() {
		return lagResourceModel{}, false
	}
	if !found {
		diags.AddError("Unable to refresh LAG", fmt.Sprintf("LAG group %d was not found after apply.", groupID))
		return lagResourceModel{}, false
	}

	return refreshed, true
}

func (r *lagResource) readLAG(ctx context.Context, groupID int64) (lagResourceModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	state, err := r.client.GetLAGs(ctx)
	if err != nil {
		diags.AddError("Unable to read LAG settings", err.Error())
		return lagResourceModel{}, false, diags
	}

	for _, group := range state.Groups {
		if int64(group.GroupID) != groupID {
			continue
		}
		if len(group.Ports) == 0 {
			return lagResourceModel{}, false, diags
		}

		ports, portDiags := int64SetFromInts(ctx, group.Ports)
		diags.Append(portDiags...)
		if diags.HasError() {
			return lagResourceModel{}, false, diags
		}

		return lagResourceModel{
			ID:      stringifyID(groupID),
			GroupID: types.Int64Value(groupID),
			Ports:   ports,
		}, true, diags
	}

	return lagResourceModel{}, false, diags
}
