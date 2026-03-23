package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
)

var (
	_ datasource.DataSource              = (*systemInfoDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*systemInfoDataSource)(nil)
)

type systemInfoDataSource struct {
	client client.Client
}

type systemInfoDataSourceModel struct {
	Description types.String `tfsdk:"description"`
	MAC         types.String `tfsdk:"mac"`
	IP          types.String `tfsdk:"ip"`
	Netmask     types.String `tfsdk:"netmask"`
	Gateway     types.String `tfsdk:"gateway"`
	Firmware    types.String `tfsdk:"firmware"`
	Hardware    types.String `tfsdk:"hardware"`
}

func NewSystemInfoDataSource() datasource.DataSource {
	return &systemInfoDataSource{}
}

func (d *systemInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_info"
}

func (d *systemInfoDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasourceschema.Schema{
		Description: "Read basic system inventory information from the switch.",
		Attributes: map[string]datasourceschema.Attribute{
			"description": datasourceschema.StringAttribute{Computed: true, Description: "Switch description string reported by the device."},
			"mac":         datasourceschema.StringAttribute{Computed: true, Description: "Base MAC address reported by the switch."},
			"ip":          datasourceschema.StringAttribute{Computed: true, Description: "Current management IP address reported by the switch."},
			"netmask":     datasourceschema.StringAttribute{Computed: true, Description: "Current management subnet mask reported by the switch."},
			"gateway":     datasourceschema.StringAttribute{Computed: true, Description: "Current management default gateway reported by the switch."},
			"firmware":    datasourceschema.StringAttribute{Computed: true, Description: "Firmware version reported by the switch."},
			"hardware":    datasourceschema.StringAttribute{Computed: true, Description: "Hardware revision reported by the switch."},
		},
	}
}

func (d *systemInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req, resp)
}

func (d *systemInfoDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	info, err := d.client.GetSystemInfo(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read system info", err.Error())
		return
	}

	state := systemInfoDataSourceModel{
		Description: types.StringValue(info.Description),
		MAC:         types.StringValue(info.MAC),
		IP:          types.StringValue(info.IP),
		Netmask:     types.StringValue(info.Netmask),
		Gateway:     types.StringValue(info.Gateway),
		Firmware:    types.StringValue(info.Firmware),
		Hardware:    types.StringValue(info.Hardware),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
