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

const igmpSnoopingID = "igmp_snooping"

var (
	_ resource.Resource                = (*igmpSnoopingResource)(nil)
	_ resource.ResourceWithConfigure   = (*igmpSnoopingResource)(nil)
	_ resource.ResourceWithImportState = (*igmpSnoopingResource)(nil)
)

type igmpSnoopingResource struct {
	client client.Client
}

type igmpSnoopingResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Enabled                  types.Bool   `tfsdk:"enabled"`
	ReportMessageSuppression types.Bool   `tfsdk:"report_message_suppression"`
}

func NewIGMPSnoopingResource() resource.Resource {
	return &igmpSnoopingResource{}
}

func (r *igmpSnoopingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_igmp_snooping"
}

func (r *igmpSnoopingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manage the switch-wide IGMP snooping settings.",
		Attributes: map[string]resourceschema.Attribute{
			"id": resourceschema.StringAttribute{
				Computed:    true,
				Description: "Terraform resource identifier for the singleton IGMP snooping object.",
			},
			"enabled": resourceschema.BoolAttribute{
				Required:    true,
				Description: "Whether IGMP snooping is enabled on the switch.",
			},
			"report_message_suppression": resourceschema.BoolAttribute{
				Required:    true,
				Description: "Whether IGMP report message suppression is enabled.",
			},
		},
	}
}

func (r *igmpSnoopingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *igmpSnoopingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan igmpSnoopingResourceModel
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

func (r *igmpSnoopingResource) Read(ctx context.Context, _ resource.ReadRequest, resp *resource.ReadResponse) {
	state, ok := r.readIGMPSnooping(ctx, &resp.Diagnostics)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *igmpSnoopingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan igmpSnoopingResourceModel
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

func (r *igmpSnoopingResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *igmpSnoopingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != igmpSnoopingID {
		resp.Diagnostics.AddAttributeError(
			path.Root("id"),
			"Invalid import identifier",
			fmt.Sprintf("Expected import ID %q for igmp_snooping, got %q.", igmpSnoopingID, req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *igmpSnoopingResource) apply(ctx context.Context, plan igmpSnoopingResourceModel, diags *diag.Diagnostics) (igmpSnoopingResourceModel, bool) {
	if r.client == nil {
		diags.AddError("Unconfigured resource client", "The provider client was not configured.")
		return igmpSnoopingResourceModel{}, false
	}

	if err := r.client.UpdateIGMPSnooping(ctx, plan.Enabled.ValueBool(), plan.ReportMessageSuppression.ValueBool()); err != nil {
		diags.AddError("Unable to apply IGMP snooping settings", err.Error())
		return igmpSnoopingResourceModel{}, false
	}

	state, ok := r.readIGMPSnooping(ctx, diags)
	return state, ok
}

func (r *igmpSnoopingResource) readIGMPSnooping(ctx context.Context, diags *diag.Diagnostics) (igmpSnoopingResourceModel, bool) {
	state, err := r.client.GetIGMPSnooping(ctx)
	if err != nil {
		diags.AddError("Unable to read IGMP snooping settings", err.Error())
		return igmpSnoopingResourceModel{}, false
	}

	return igmpSnoopingResourceModel{
		ID:                       types.StringValue(igmpSnoopingID),
		Enabled:                  types.BoolValue(state.Enabled),
		ReportMessageSuppression: types.BoolValue(state.ReportMessageSuppression),
	}, true
}
