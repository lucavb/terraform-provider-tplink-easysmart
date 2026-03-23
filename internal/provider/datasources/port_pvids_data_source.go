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
	_ datasource.DataSource              = (*portPVIDsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*portPVIDsDataSource)(nil)
)

type portPVIDsDataSource struct {
	client client.Client
}

type portPVIDsDataSourceModel struct {
	PVIDs types.List `tfsdk:"pvids"`
}

func NewPortPVIDsDataSource() datasource.DataSource {
	return &portPVIDsDataSource{}
}

func (d *portPVIDsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_port_pvids"
}

func (d *portPVIDsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		Description: "Read the 802.1Q PVID assigned to each switch port.",
		Attributes: map[string]datasourceschema.Attribute{
			"pvids": datasourceschema.ListNestedAttribute{
				Computed: true,
				Description: "Per-port PVID assignments reported by the switch.",
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"port_id": datasourceschema.Int64Attribute{Computed: true, Description: "Physical switch port number."},
						"pvid":    datasourceschema.Int64Attribute{Computed: true, Description: "Current PVID assigned to the port."},
					},
				},
			},
		},
	}
}

func (d *portPVIDsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req, resp)
}

func (d *portPVIDsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	pvids, err := d.client.GetPVIDs(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read port PVID inventory", err.Error())
		return
	}

	values := make([]attr.Value, 0, len(pvids))
	for _, entry := range pvids {
		value, diags := types.ObjectValue(portPVIDObjectType().AttrTypes, map[string]attr.Value{
			"port_id": types.Int64Value(int64(entry.PortID)),
			"pvid":    types.Int64Value(int64(entry.PVID)),
		})
		resp.Diagnostics.Append(diags...)
		values = append(values, value)
	}

	listValue, diags := types.ListValue(portPVIDObjectType(), values)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := portPVIDsDataSourceModel{
		PVIDs: listValue,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func portPVIDObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"port_id": types.Int64Type,
			"pvid":    types.Int64Type,
		},
	}
}
