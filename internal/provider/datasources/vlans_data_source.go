package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/normalize"
)

var (
	_ datasource.DataSource              = (*vlansDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*vlansDataSource)(nil)
)

type vlansDataSource struct {
	client client.Client
}

type vlansDataSourceModel struct {
	Enabled   types.Bool  `tfsdk:"enabled"`
	PortNum   types.Int64 `tfsdk:"port_num"`
	VLANCount types.Int64 `tfsdk:"vlan_count"`
	MaxVLANs  types.Int64 `tfsdk:"max_vlans"`
	VLANs     types.List  `tfsdk:"vlans"`
}

func NewVLANsDataSource() datasource.DataSource {
	return &vlansDataSource{}
}

func (d *vlansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vlans"
}

func (d *vlansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		Description: "Read the 802.1Q VLAN inventory from the switch.",
		Attributes: map[string]datasourceschema.Attribute{
			"enabled":    datasourceschema.BoolAttribute{Computed: true, Description: "Whether 802.1Q VLAN mode is enabled on the switch."},
			"port_num":   datasourceschema.Int64Attribute{Computed: true, Description: "Number of ports reported by the VLAN page."},
			"vlan_count": datasourceschema.Int64Attribute{Computed: true, Description: "Current number of VLAN entries reported by the switch."},
			"max_vlans":  datasourceschema.Int64Attribute{Computed: true, Description: "Maximum VLAN entries supported by the switch UI."},
			"vlans": datasourceschema.ListNestedAttribute{
				Computed: true,
				Description: "Current VLAN table entries reported by the switch.",
				NestedObject: datasourceschema.NestedAttributeObject{
					Attributes: map[string]datasourceschema.Attribute{
						"vlan_id":        datasourceschema.Int64Attribute{Computed: true, Description: "Numeric VLAN identifier."},
						"name":           datasourceschema.StringAttribute{Computed: true, Description: "VLAN display name reported by the switch."},
						"tagged_ports":   datasourceschema.ListAttribute{Computed: true, ElementType: types.Int64Type, Description: "Tagged member ports for the VLAN."},
						"untagged_ports": datasourceschema.ListAttribute{Computed: true, ElementType: types.Int64Type, Description: "Untagged member ports for the VLAN."},
					},
				},
			},
		},
	}
}

func (d *vlansDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req, resp)
}

func (d *vlansDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	table, err := d.client.GetVLANs(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read VLAN inventory", err.Error())
		return
	}

	values := make([]attr.Value, 0, len(table.VLANs))
	for _, vlan := range table.VLANs {
		taggedPorts, diags := int64ListFromInts(ctx, normalize.NormalizePorts(vlan.TaggedPorts))
		resp.Diagnostics.Append(diags...)
		untaggedPorts, diags := int64ListFromInts(ctx, normalize.NormalizePorts(vlan.UntaggedPorts))
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		vlanValue, diags := types.ObjectValue(vlanObjectType().AttrTypes, map[string]attr.Value{
			"vlan_id":        types.Int64Value(int64(vlan.ID)),
			"name":           types.StringValue(vlan.Name),
			"tagged_ports":   taggedPorts,
			"untagged_ports": untaggedPorts,
		})
		resp.Diagnostics.Append(diags...)
		values = append(values, vlanValue)
	}

	vlanList, diags := types.ListValue(vlanObjectType(), values)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := vlansDataSourceModel{
		Enabled:   types.BoolValue(table.Enabled),
		PortNum:   types.Int64Value(int64(table.PortNum)),
		VLANCount: types.Int64Value(int64(table.Count)),
		MaxVLANs:  types.Int64Value(int64(table.MaxVLANs)),
		VLANs:     vlanList,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func vlanObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"vlan_id":        types.Int64Type,
			"name":           types.StringType,
			"tagged_ports":   types.ListType{ElemType: types.Int64Type},
			"untagged_ports": types.ListType{ElemType: types.Int64Type},
		},
	}
}
