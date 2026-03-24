package webui

import (
	"context"
	"fmt"
	"net/url"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/model"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/normalize"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/parser"
)

const igmpSnoopingResourceID = "igmp_snooping"

func (c *Client) GetIGMPSnooping(ctx context.Context) (model.IGMPSnooping, error) {
	state, err := c.getIGMPSnoopingState(ctx)
	if err != nil {
		return model.IGMPSnooping{}, err
	}

	return model.IGMPSnooping{
		ID:                       igmpSnoopingResourceID,
		Enabled:                  state.Enabled,
		ReportMessageSuppression: state.ReportMessageSuppression,
		Groups:                   state.Groups,
	}, nil
}

func (c *Client) UpdateIGMPSnooping(ctx context.Context, enabled bool, suppression bool) error {
	query := url.Values{
		"igmp_mode":     []string{boolString(enabled)},
		"reportSu_mode": []string{boolString(suppression)},
		"Apply":         []string{"Apply"},
	}

	_, err := c.doProtectedQuery(ctx, "/igmpSnooping.cgi", query)
	return err
}

func (c *Client) GetLAGs(ctx context.Context) (model.LAGInfo, error) {
	return c.getLAGState(ctx)
}

func (c *Client) UpsertLAG(ctx context.Context, groupID int, ports []int) error {
	state, err := c.getLAGState(ctx)
	if err != nil {
		return err
	}

	ports = normalize.NormalizePorts(ports)
	if err := validateLAGPorts(groupID, ports, state.MaxGroups, state.PortCount, state.PortsPerGroup); err != nil {
		return err
	}

	query := url.Values{
		"groupId":  []string{fmt.Sprintf("%d", groupID)},
		"setapply": []string{"Apply"},
	}
	for _, portID := range ports {
		query.Add("portid", fmt.Sprintf("%d", portID))
	}

	_, err = c.doProtectedQuery(ctx, "/port_trunk_set.cgi", query)
	return err
}

func (c *Client) DeleteLAG(ctx context.Context, groupID int) error {
	if groupID < 1 {
		return fmt.Errorf("group_id must be greater than zero")
	}

	query := url.Values{
		"chk_trunk": []string{fmt.Sprintf("%d", groupID)},
		"setDelete": []string{"Delete"},
	}

	_, err := c.doProtectedQuery(ctx, "/port_trunk_display.cgi", query)
	return err
}

type igmpSnoopingState struct {
	Enabled                  bool
	ReportMessageSuppression bool
	Groups                   []model.IGMPGroup
}

func (c *Client) getIGMPSnoopingState(ctx context.Context) (igmpSnoopingState, error) {
	body, err := c.fetchProtectedPage(ctx, "/IgmpSnoopingRpm.htm")
	if err != nil {
		return igmpSnoopingState{}, err
	}

	raw, err := parser.ExtractObject(body, "igmp_ds")
	if err != nil {
		return igmpSnoopingState{}, err
	}

	ips := intSlice(raw["ipStr"])
	vlans := intSlice(raw["vlanStr"])
	portMasks := intSlice(raw["portStr"])
	lagMasks := intSlice(raw["lagMbrs"])

	groupCount := intValue(raw["count"])
	groups := make([]model.IGMPGroup, 0, groupCount)
	for idx := 0; idx < groupCount; idx++ {
		ports, lags := decodeIGMPMembership(intAt(portMasks, idx), lagMasks)
		groups = append(groups, model.IGMPGroup{
			IPAddress: decodeIPAddress(intAt(ips, idx)),
			VLANID:    intAt(vlans, idx),
			Ports:     ports,
			LAGGroups: lags,
		})
	}

	return igmpSnoopingState{
		Enabled:                  intValue(raw["state"]) != 0,
		ReportMessageSuppression: intValue(raw["suppressionState"]) != 0,
		Groups:                   groups,
	}, nil
}

func (c *Client) getLAGState(ctx context.Context) (model.LAGInfo, error) {
	body, err := c.fetchProtectedPage(ctx, "/PortTrunkRpm.htm")
	if err != nil {
		return model.LAGInfo{}, err
	}

	raw, err := parser.ExtractObject(body, "trunk_conf")
	if err != nil {
		return model.LAGInfo{}, err
	}

	maxGroups := intValue(raw["maxTrunkNum"])
	portCount := intValue(raw["portNum"])
	portsPerGroup, err := parser.ExtractInt(body, "portNumPerTrunk")
	if err != nil {
		return model.LAGInfo{}, err
	}

	groups := make([]model.LAGGroup, 0, maxGroups)
	for groupID := 1; groupID <= maxGroups; groupID++ {
		key := fmt.Sprintf("portStr_g%d", groupID)
		ports := decodePortStringGroup(intSlice(raw[key]))
		groups = append(groups, model.LAGGroup{
			GroupID: groupID,
			Ports:   ports,
		})
	}

	return model.LAGInfo{
		MaxGroups:     maxGroups,
		PortCount:     portCount,
		PortsPerGroup: portsPerGroup,
		Groups:        groups,
	}, nil
}

func decodeIGMPMembership(portMask int, lagMasks []int) ([]int, []int) {
	remaining := portMask
	lags := make([]int, 0, len(lagMasks))
	for idx, lagMask := range lagMasks {
		if lagMask == 0 {
			continue
		}
		if remaining&lagMask != 0 {
			lags = append(lags, idx+1)
			remaining &^= lagMask
		}
	}

	ports := normalize.DecodePortBitmask(remaining, 32)
	return ports, lags
}

func decodeIPAddress(value int) string {
	return fmt.Sprintf(
		"%d.%d.%d.%d",
		(value>>24)&0xff,
		(value>>16)&0xff,
		(value>>8)&0xff,
		value&0xff,
	)
}

func decodePortStringGroup(values []int) []int {
	ports := make([]int, 0, len(values))
	for idx, value := range values {
		if value != 0 {
			ports = append(ports, idx+1)
		}
	}
	return ports
}

func validateLAGPorts(groupID int, ports []int, maxGroups int, portCount int, portsPerGroup int) error {
	if groupID < 1 || groupID > maxGroups {
		return fmt.Errorf("group_id must be between 1 and %d", maxGroups)
	}
	if len(ports) < 2 {
		return fmt.Errorf("at least two ports must be selected for a LAG")
	}
	if len(ports) > portsPerGroup {
		return fmt.Errorf("no more than %d ports may be selected for one LAG", portsPerGroup)
	}

	minPort := ((groupID - 1) * portsPerGroup) + 1
	maxPort := groupID * portsPerGroup
	if maxPort > portCount {
		maxPort = portCount
	}

	sorted := normalize.NormalizePorts(ports)
	for _, portID := range sorted {
		if portID < minPort || portID > maxPort {
			return fmt.Errorf("group_id %d only supports ports in the range %d-%d on this device", groupID, minPort, maxPort)
		}
	}
	return nil
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
