package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
)

var (
	_ datasource.DataSource              = (*portsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*portsDataSource)(nil)
)

type portsDataSource struct {
	client client.Client
}

type portsDataSourceModel struct {
	Ports types.List `tfsdk:"ports"`
}

func NewPortsDataSource() datasource.DataSource {
	return &portsDataSource{}
}

func (d *portsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ports"
}

func (d *portsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		Description: "Read per-port configuration and effective state.",
		Attributes: map[string]datasourceschema.Attribute{
			"ports": datasourceschema.ListNestedAttribute{
				Computed: true,
				Description: "Port inventory as reported by the switch web UI.",
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"id":                  datasourceschema.Int64Attribute{Computed: true, Description: "Physical switch port number."},
						"enabled":             datasourceschema.BoolAttribute{Computed: true, Description: "Administrative enable state for the port."},
						"trunk_member":        datasourceschema.BoolAttribute{Computed: true, Description: "Whether the device reports the port as a trunk member."},
						"speed_config":        datasourceschema.Int64Attribute{Computed: true, Description: "Configured speed and duplex enum reported by the device page model."},
						"speed_actual":        datasourceschema.Int64Attribute{Computed: true, Description: "Effective speed and duplex enum reported by the device page model."},
						"flow_control_config": datasourceschema.Int64Attribute{Computed: true, Description: "Configured flow-control enum reported by the device page model."},
						"flow_control_actual": datasourceschema.Int64Attribute{Computed: true, Description: "Effective flow-control enum reported by the device page model."},
					},
				},
			},
		},
	}
}

func (d *portsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req, resp)
}

func (d *portsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	ports, err := d.client.GetPorts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read port settings", err.Error())
		return
	}

	values := make([]attr.Value, 0, len(ports))
	for _, port := range ports {
		portValue, diags := types.ObjectValue(portObjectType().AttrTypes, map[string]attr.Value{
			"id":                  types.Int64Value(int64(port.ID)),
			"enabled":             types.BoolValue(port.Enabled),
			"trunk_member":        types.BoolValue(port.TrunkMember),
			"speed_config":        types.Int64Value(int64(port.SpeedConfig)),
			"speed_actual":        types.Int64Value(int64(port.SpeedActual)),
			"flow_control_config": types.Int64Value(int64(port.FlowControlConfig)),
			"flow_control_actual": types.Int64Value(int64(port.FlowControlActual)),
		})
		resp.Diagnostics.Append(diags...)
		values = append(values, portValue)
	}

	listValue, diags := types.ListValue(portObjectType(), values)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := portsDataSourceModel{
		Ports: listValue,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func portObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":                  types.Int64Type,
			"enabled":             types.BoolType,
			"trunk_member":        types.BoolType,
			"speed_config":        types.Int64Type,
			"speed_actual":        types.Int64Type,
			"flow_control_config": types.Int64Type,
			"flow_control_actual": types.Int64Type,
		},
	}
}
