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
	_ resource.Resource                = (*qosModeResource)(nil)
	_ resource.ResourceWithConfigure   = (*qosModeResource)(nil)
	_ resource.ResourceWithImportState = (*qosModeResource)(nil)
)

type qosModeResource struct {
	client client.Client
}

type qosModeResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Mode types.String `tfsdk:"mode"`
}

func NewQoSModeResource() resource.Resource {
	return &qosModeResource{}
}

func (r *qosModeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_qos_mode"
}

func (r *qosModeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manage the switch-wide QoS mode.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource identifier for the singleton QoS mode object.",
			},
			"mode": resourceschema.StringAttribute{
				Required:    true,
				Description: "QoS mode. Supported values are `port_based`, `dot1p_based`, and `dscp_dot1p_based`.",
			},
		},
	}
}

func (r *qosModeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *qosModeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan qosModeResourceModel
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

func (r *qosModeResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	state, ok := r.readQoSMode(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *qosModeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan qosModeResourceModel
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

func (r *qosModeResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *qosModeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != qosModeID {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Invalid import identifier",
			fmt.Sprintf("Expected import ID %q for qos_mode, got %q.", qosModeID, req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *qosModeResource) apply(ctx context.Context, plan qosModeResourceModel, diags *diag.Diagnostics) (qosModeResourceModel, bool) {
	if r.client == nil {
		diags.AddError("Unconfigured resource client", "The provider client was not configured.")
		return qosModeResourceModel{}, false
	}

	mode, ok := qosModeValue(plan.Mode.ValueString())
	if !ok {
		diags.AddAttributeError(
			path.Root("mode"),
			"Invalid QoS mode",
			fmt.Sprintf("mode must be one of %q, %q, or %q.", "port_based", "dot1p_based", "dscp_dot1p_based"),
		)
		return qosModeResourceModel{}, false
	}

	if err := r.client.UpdateQoSMode(ctx, mode); err != nil {
		diags.AddError("Unable to apply QoS mode", err.Error())
		return qosModeResourceModel{}, false
	}

	state, refreshed := r.readQoSMode(ctx, diags)
	return state, refreshed
}

func (r *qosModeResource) readQoSMode(ctx context.Context, diags *diag.Diagnostics) (qosModeResourceModel, bool) {
	mode, err := r.client.GetQoSMode(ctx)
	if err != nil {
		diags.AddError("Unable to read QoS mode", err.Error())
		return qosModeResourceModel{}, false
	}

	modeName, ok := qosModeName(mode.Mode)
	if !ok {
		diags.AddError("Unsupported QoS mode", fmt.Sprintf("Observed unsupported QoS mode value %d.", mode.Mode))
		return qosModeResourceModel{}, false
	}

	return qosModeResourceModel{
		ID:   types.StringValue(qosModeID),
		Mode: types.StringValue(modeName),
	}, true
}
