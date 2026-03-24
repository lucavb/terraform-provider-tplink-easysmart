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
	_ resource.Resource                = (*portStormControlResource)(nil)
	_ resource.ResourceWithConfigure   = (*portStormControlResource)(nil)
	_ resource.ResourceWithImportState = (*portStormControlResource)(nil)
)

type portStormControlResource struct {
	client client.Client
}

type portStormControlResourceModel struct {
	ID         types.String `tfsdk:"id"`
	PortID     types.Int64  `tfsdk:"port_id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	RateKbps   types.Int64  `tfsdk:"rate_kbps"`
	StormTypes types.Set    `tfsdk:"storm_types"`
}

func NewPortStormControlResource() resource.Resource {
	return &portStormControlResource{}
}

func (r *portStormControlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_storm_control"
}

func (r *portStormControlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resourceschema.Schema{
		Description: "Manage one switch port's storm-control rate and enabled storm types.",
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
				Description: "Whether storm control is enabled for the port.",
			},
			"rate_kbps": resourceschema.Int64Attribute{
				Required:    true,
				Description: "Total rate limit in Kbps. Use `0` when `enabled = false`.",
			},
			"storm_types": resourceschema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Storm types included in the limit. Supported values are `ul_frame`, `multicast`, and `broadcast`. Use an empty set when `enabled = false`.",
			},
		},
	}
}

func (r *portStormControlResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

func (r *portStormControlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan portStormControlResourceModel
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

func (r *portStormControlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state portStormControlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, found, diags := r.readStormControl(ctx, state.PortID.ValueInt64())
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

func (r *portStormControlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan portStormControlResourceModel
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

func (r *portStormControlResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *portStormControlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *portStormControlResource) apply(ctx context.Context, plan portStormControlResourceModel, diags *diag.Diagnostics) (portStormControlResourceModel, bool) {
	if r.client == nil {
		diags.AddError("Unconfigured resource client", "The provider client was not configured.")
		return portStormControlResourceModel{}, false
	}

	portID := plan.PortID.ValueInt64()
	if portID < 1 {
		diags.AddAttributeError(path.Root("port_id"), "Invalid port ID", "port_id must be greater than zero.")
		return portStormControlResourceModel{}, false
	}

	stormTypes, stormTypeDiags := stormTypeValuesFromSet(ctx, plan.StormTypes, "storm_types")
	diags.Append(stormTypeDiags...)
	if diags.HasError() {
		return portStormControlResourceModel{}, false
	}

	enabled := plan.Enabled.ValueBool()
	rate := plan.RateKbps.ValueInt64()
	if enabled {
		if rate < 1 || rate > 1000000 {
			diags.AddAttributeError(path.Root("rate_kbps"), "Invalid storm-control rate", "rate_kbps must be in the range 1-1000000 when enabled is true.")
			return portStormControlResourceModel{}, false
		}
		if len(stormTypes) == 0 {
			diags.AddAttributeError(path.Root("storm_types"), "Missing storm types", "storm_types must contain at least one entry when enabled is true.")
			return portStormControlResourceModel{}, false
		}
	} else {
		if rate != 0 {
			diags.AddAttributeError(path.Root("rate_kbps"), "Invalid disabled storm-control rate", "rate_kbps must be 0 when enabled is false.")
			return portStormControlResourceModel{}, false
		}
		if len(stormTypes) != 0 {
			diags.AddAttributeError(path.Root("storm_types"), "Invalid disabled storm types", "storm_types must be empty when enabled is false.")
			return portStormControlResourceModel{}, false
		}
	}

	if err := r.client.SetPortStormControl(ctx, int(portID), enabled, int(rate), stormTypes); err != nil {
		diags.AddError("Unable to apply storm control", err.Error())
		return portStormControlResourceModel{}, false
	}

	refreshed, found, refreshDiags := r.readStormControl(ctx, portID)
	diags.Append(refreshDiags...)
	if diags.HasError() {
		return portStormControlResourceModel{}, false
	}
	if !found {
		diags.AddError("Unable to refresh storm control", fmt.Sprintf("Port %d was not found after apply.", portID))
		return portStormControlResourceModel{}, false
	}

	return refreshed, true
}

func (r *portStormControlResource) readStormControl(ctx context.Context, portID int64) (portStormControlResourceModel, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	entries, err := r.client.GetPortStormControls(ctx)
	if err != nil {
		diags.AddError("Unable to read storm-control settings", err.Error())
		return portStormControlResourceModel{}, false, diags
	}

	for _, entry := range entries {
		if int64(entry.PortID) != portID {
			continue
		}

		stormTypes, stormTypeDiags := stormTypeSetFromValues(ctx, entry.StormTypes)
		diags.Append(stormTypeDiags...)
		if diags.HasError() {
			return portStormControlResourceModel{}, false, diags
		}

		rate := entry.RateKbps
		if !entry.Enabled {
			rate = 0
		}

		return portStormControlResourceModel{
			ID:         stringifyID(portID),
			PortID:     types.Int64Value(portID),
			Enabled:    types.BoolValue(entry.Enabled),
			RateKbps:   types.Int64Value(int64(rate)),
			StormTypes: stormTypes,
		}, true, diags
	}

	return portStormControlResourceModel{}, false, diags
}
