package webui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/model"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/normalize"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/parser"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/session"
)

type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

func New(cfg client.Config) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}
	if httpClient.Timeout == 0 {
		httpClient.Timeout = 10 * time.Second
	}

	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		username:   cfg.Username,
		password:   cfg.Password,
		httpClient: httpClient,
	}
}

func (c *Client) Authenticate(ctx context.Context) error {
	form := url.Values{
		"username":  []string{c.username},
		"password":  []string{c.password},
		"cpassword": []string{""},
		"logon":     []string{"Login"},
	}

	body, err := c.doForm(ctx, "/logon.cgi", form)
	if err != nil {
		return err
	}

	status, code := session.ClassifyLoginResponse(body)
	if status != session.LoginStatusSuccess {
		return fmt.Errorf("%w: logonInfo=%d", client.ErrLoginFailed, code)
	}

	tflog.Debug(ctx, "switch login succeeded", map[string]any{
		"base_url": c.baseURL,
	})

	return nil
}

func (c *Client) GetSystemInfo(ctx context.Context) (model.SystemInfo, error) {
	body, err := c.fetchProtectedPage(ctx, "/SystemInfoRpm.htm")
	if err != nil {
		return model.SystemInfo{}, err
	}

	raw, err := parser.ExtractObject(body, "info_ds")
	if err != nil {
		return model.SystemInfo{}, err
	}

	return model.SystemInfo{
		Description: firstString(raw["descriStr"]),
		MAC:         firstString(raw["macStr"]),
		IP:          firstString(raw["ipStr"]),
		Netmask:     firstString(raw["netmaskStr"]),
		Gateway:     firstString(raw["gatewayStr"]),
		Firmware:    firstString(raw["firmwareStr"]),
		Hardware:    firstString(raw["hardwareStr"]),
	}, nil
}

func (c *Client) GetPorts(ctx context.Context) ([]model.Port, error) {
	body, err := c.fetchProtectedPage(ctx, "/PortSettingRpm.htm")
	if err != nil {
		return nil, err
	}

	maxPortNum, err := parser.ExtractInt(body, "max_port_num")
	if err != nil {
		return nil, err
	}

	raw, err := parser.ExtractObject(body, "all_info")
	if err != nil {
		return nil, err
	}

	state := intSlice(raw["state"])
	trunkInfo := intSlice(raw["trunk_info"])
	speedCfg := intSlice(raw["spd_cfg"])
	speedAct := intSlice(raw["spd_act"])
	flowCfg := intSlice(raw["fc_cfg"])
	flowAct := intSlice(raw["fc_act"])

	ports := make([]model.Port, 0, maxPortNum)
	for idx := 0; idx < maxPortNum; idx++ {
		ports = append(ports, model.Port{
			ID:                idx + 1,
			Enabled:           intAt(state, idx) != 0,
			TrunkMember:       intAt(trunkInfo, idx) != 0,
			SpeedConfig:       intAt(speedCfg, idx),
			SpeedActual:       intAt(speedAct, idx),
			FlowControlConfig: intAt(flowCfg, idx),
			FlowControlActual: intAt(flowAct, idx),
		})
	}

	return ports, nil
}

func (c *Client) GetManagementIP(ctx context.Context) (model.ManagementIP, error) {
	body, err := c.fetchProtectedPage(ctx, "/IpSettingRpm.htm")
	if err != nil {
		return model.ManagementIP{}, err
	}

	raw, err := parser.ExtractObject(body, "ip_ds")
	if err != nil {
		return model.ManagementIP{}, err
	}

	return model.ManagementIP{
		State:   intValue(raw["state"]),
		VLAN:    intValue(raw["vlan"]),
		MaxVLAN: intValue(raw["maxVlan"]),
		IP:      firstString(raw["ipStr"]),
		Netmask: firstString(raw["netmaskStr"]),
		Gateway: firstString(raw["gatewayStr"]),
	}, nil
}

func (c *Client) GetVLANs(ctx context.Context) (model.VLANTable, error) {
	body, err := c.fetchProtectedPage(ctx, "/Vlan8021QRpm.htm")
	if err != nil {
		return model.VLANTable{}, err
	}

	raw, err := parser.ExtractObject(body, "qvlan_ds")
	if err != nil {
		return model.VLANTable{}, err
	}

	vids := intSlice(raw["vids"])
	names := stringSlice(raw["names"])
	tagMasks := intSlice(raw["tagMbrs"])
	untagMasks := intSlice(raw["untagMbrs"])
	portNum := intValue(raw["portNum"])

	vlans := make([]model.VLAN, 0, len(vids))
	for idx, vid := range vids {
		vlans = append(vlans, model.VLAN{
			ID:            vid,
			Name:          stringAt(names, idx),
			TaggedPorts:   normalize.DecodePortBitmask(intAt(tagMasks, idx), portNum),
			UntaggedPorts: normalize.DecodePortBitmask(intAt(untagMasks, idx), portNum),
		})
	}

	return model.VLANTable{
		Enabled:  intValue(raw["state"]) != 0,
		PortNum:  portNum,
		Count:    intValue(raw["count"]),
		MaxVLANs: intValue(raw["maxVids"]),
		VLANs:    vlans,
	}, nil
}

func (c *Client) GetPVIDs(ctx context.Context) ([]model.PortPVID, error) {
	body, err := c.fetchProtectedPage(ctx, "/Vlan8021QPvidRpm.htm")
	if err != nil {
		return nil, err
	}

	for _, candidate := range []string{"pvid_ds", "qpvid_ds", "qvlan_pvid_ds"} {
		raw, parseErr := parser.ExtractObject(body, candidate)
		if parseErr != nil {
			continue
		}

		for _, key := range []string{"pvid", "pvids", "portPvid", "port_pvid"} {
			values := intSlice(raw[key])
			if len(values) == 0 {
				continue
			}

			out := make([]model.PortPVID, 0, len(values))
			for idx, pvid := range values {
				out = append(out, model.PortPVID{
					PortID: idx + 1,
					PVID:   pvid,
				})
			}
			return out, nil
		}
	}

	for _, candidate := range []string{"pvid", "pvids", "portPvid", "port_pvid"} {
		values, parseErr := parser.ExtractArray(body, candidate)
		if parseErr != nil {
			continue
		}

		ints := intSlice(values)
		out := make([]model.PortPVID, 0, len(ints))
		for idx, pvid := range ints {
			out = append(out, model.PortPVID{
				PortID: idx + 1,
				PVID:   pvid,
			})
		}
		return out, nil
	}

	return nil, fmt.Errorf("%w: pvid page model not yet observed", client.ErrUnsupportedPageModel)
}

func (c *Client) UpsertVLAN(ctx context.Context, vlanID int, name string, taggedPorts []int, untaggedPorts []int) error {
	table, err := c.GetVLANs(ctx)
	if err != nil {
		return err
	}
	if !table.Enabled {
		return fmt.Errorf("802.1Q VLAN mode is disabled")
	}

	tagged := normalize.NormalizePorts(taggedPorts)
	untagged := normalize.NormalizePorts(untaggedPorts)
	if overlappingPorts(tagged, untagged) {
		return fmt.Errorf("tagged and untagged port memberships must not overlap")
	}

	query := url.Values{
		"vid":       []string{fmt.Sprintf("%d", vlanID)},
		"vname":     []string{name},
		"qvlan_add": []string{"Add/Modify"},
	}
	for port := 1; port <= table.PortNum; port++ {
		selector := "2"
		switch {
		case containsPort(untagged, port):
			selector = "0"
		case containsPort(tagged, port):
			selector = "1"
		}
		query.Set(fmt.Sprintf("selType_%d", port), selector)
	}

	_, err = c.doProtectedQuery(ctx, "/qvlanSet.cgi", query)
	return err
}

func (c *Client) DeleteVLAN(ctx context.Context, vlanID int) error {
	query := url.Values{
		"selVlans":  []string{fmt.Sprintf("%d", vlanID)},
		"qvlan_del": []string{"Delete"},
	}
	_, err := c.doProtectedQuery(ctx, "/qvlanSet.cgi", query)
	return err
}

func (c *Client) SetPortPVID(ctx context.Context, portID int, pvid int) error {
	query := url.Values{
		"pbm":  []string{fmt.Sprintf("%d", normalize.EncodePortBitmask([]int{portID}))},
		"pvid": []string{fmt.Sprintf("%d", pvid)},
	}
	_, err := c.doProtectedQuery(ctx, "/vlanPvidSet.cgi", query)
	return err
}

func (c *Client) UpdatePortSettings(ctx context.Context, portID int, enabled *bool, speedConfig *int, flowControlConfig *int) error {
	query := url.Values{
		"portid":      []string{fmt.Sprintf("%d", portID)},
		"state":       []string{portSettingBoolValue(enabled)},
		"speed":       []string{portSettingIntValue(speedConfig)},
		"flowcontrol": []string{portSettingIntValue(flowControlConfig)},
		"apply":       []string{"Apply"},
	}
	_, err := c.doProtectedQuery(ctx, "/port_setting.cgi", query)
	return err
}

func (c *Client) fetchProtectedPage(ctx context.Context, path string) (string, error) {
	body, err := c.doRequest(ctx, http.MethodGet, path, "", nil)
	if err != nil {
		return "", err
	}

	if session.IsLoginPage(body) {
		return "", client.ErrPageReturnedLogin
	}

	return body, nil
}

func (c *Client) doProtectedQuery(ctx context.Context, path string, query url.Values) (string, error) {
	body, err := c.doRequest(ctx, http.MethodGet, path+"?"+query.Encode(), "", nil)
	if err != nil {
		return "", err
	}
	if session.IsLoginPage(body) {
		return "", client.ErrPageReturnedLogin
	}
	return body, nil
}

func (c *Client) doProtectedForm(ctx context.Context, path string, form url.Values) (string, error) {
	body, err := c.doForm(ctx, path, form)
	if err != nil {
		return "", err
	}
	if session.IsLoginPage(body) {
		return "", client.ErrPageReturnedLogin
	}
	return body, nil
}

func (c *Client) doForm(ctx context.Context, path string, form url.Values) (string, error) {
	return c.doRequest(ctx, http.MethodPost, path, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
}

func (c *Client) doRequest(ctx context.Context, method string, path string, contentType string, body io.Reader) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return "", err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, path)
	}

	return string(raw), nil
}

func firstString(value any) string {
	values := stringSlice(value)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func stringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		str, ok := item.(string)
		if !ok {
			continue
		}
		out = append(out, str)
	}
	return out
}

func intSlice(value any) []int {
	items, ok := value.([]any)
	if !ok {
		return nil
	}

	out := make([]int, 0, len(items))
	for _, item := range items {
		out = append(out, intValue(item))
	}
	return out
}

func intValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return 0
	}
}

func intAt(values []int, idx int) int {
	if idx < 0 || idx >= len(values) {
		return 0
	}
	return values[idx]
}

func stringAt(values []string, idx int) string {
	if idx < 0 || idx >= len(values) {
		return ""
	}
	return values[idx]
}

func containsPort(ports []int, port int) bool {
	for _, candidate := range ports {
		if candidate == port {
			return true
		}
	}
	return false
}

func overlappingPorts(left []int, right []int) bool {
	for _, port := range left {
		if containsPort(right, port) {
			return true
		}
	}
	return false
}

func portSettingBoolValue(value *bool) string {
	if value == nil {
		return "7"
	}
	if *value {
		return "1"
	}
	return "0"
}

func portSettingIntValue(value *int) string {
	if value == nil {
		return "7"
	}
	return fmt.Sprintf("%d", *value)
}
