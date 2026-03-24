package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/webui"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/provider/datasources"
	providerresources "github.com/lucavb/terraform-provider-tplink-easysmart/internal/provider/resources"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/providerdata"
)

var _ provider.Provider = (*tplinkEasySmartProvider)(nil)

type tplinkEasySmartProvider struct {
	version string
}

type providerModel struct {
	Host         types.String `tfsdk:"host"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	InsecureHTTP types.Bool   `tfsdk:"insecure_http"`
	Timeout      types.Int64  `tfsdk:"timeout_seconds"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &tplinkEasySmartProvider{version: version}
	}
}

func (p *tplinkEasySmartProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tplinkeasysmart"
	resp.Version = p.version
}

func (p *tplinkEasySmartProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{
		Description: "Terraform provider for TP-Link Easy Smart switches.",
		Attributes: map[string]providerschema.Attribute{
			"host": providerschema.StringAttribute{
				Required:    true,
				Description: "Switch hostname or base URL.",
			},
			"username": providerschema.StringAttribute{
				Required:    true,
				Description: "Web UI username.",
			},
			"password": providerschema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Web UI password.",
			},
			"insecure_http": providerschema.BoolAttribute{
				Optional:    true,
				Description: "Use http:// when the host does not include a scheme. Defaults to true for legacy Easy Smart devices.",
			},
			"timeout_seconds": providerschema.Int64Attribute{
				Optional:    true,
				Description: "HTTP timeout in seconds. Defaults to 10.",
			},
		},
	}
}

func (p *tplinkEasySmartProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Host.IsUnknown() || config.Username.IsUnknown() || config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown provider configuration",
			"The provider cannot create a switch client while configuration values are unknown.",
		)
		return
	}

	insecureHTTP := true
	if !config.InsecureHTTP.IsNull() {
		insecureHTTP = config.InsecureHTTP.ValueBool()
	}

	timeout := 10 * time.Second
	if !config.Timeout.IsNull() {
		timeout = time.Duration(config.Timeout.ValueInt64()) * time.Second
	}

	baseURL := buildBaseURL(config.Host.ValueString(), insecureHTTP)
	httpClient := &http.Client{Timeout: timeout}

	switchClient := webui.New(client.Config{
		BaseURL:    baseURL,
		Username:   config.Username.ValueString(),
		Password:   config.Password.ValueString(),
		Timeout:    timeout,
		HTTPClient: httpClient,
	})

	if err := switchClient.Authenticate(ctx); err != nil {
		resp.Diagnostics.AddError(
			"Unable to authenticate to the switch",
			fmt.Sprintf("Failed to authenticate against %s: %s", baseURL, err),
		)
		return
	}

	tflog.Info(ctx, "configured TP-Link Easy Smart provider", map[string]any{
		"base_url": baseURL,
	})

	data := &providerdata.Data{SwitchClient: switchClient}
	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *tplinkEasySmartProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		providerresources.NewVLAN8021QResource,
		providerresources.NewPortPVIDResource,
		providerresources.NewPortSettingResource,
		providerresources.NewIGMPSnoopingResource,
		providerresources.NewLAGResource,
		providerresources.NewQoSModeResource,
		providerresources.NewPortQoSPriorityResource,
		providerresources.NewPortBandwidthControlResource,
		providerresources.NewPortStormControlResource,
	}
}

func (p *tplinkEasySmartProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewSystemInfoDataSource,
		datasources.NewPortsDataSource,
		datasources.NewVLANsDataSource,
		datasources.NewPortPVIDsDataSource,
	}
}

func buildBaseURL(host string, insecureHTTP bool) string {
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return strings.TrimRight(host, "/")
	}

	scheme := "https"
	if insecureHTTP {
		scheme = "http"
	}

	return scheme + "://" + strings.TrimRight(host, "/")
}
