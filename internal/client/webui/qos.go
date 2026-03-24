package webui

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/model"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client/parser"
)

const qosModeResourceID = "qos"

func (c *Client) GetQoSMode(ctx context.Context) (model.QoSMode, error) {
	state, err := c.getQoSBasicState(ctx)
	if err != nil {
		return model.QoSMode{}, err
	}

	return model.QoSMode{
		ID:   qosModeResourceID,
		Mode: state.Mode,
	}, nil
}

func (c *Client) GetPortQoSPriorities(ctx context.Context) ([]model.PortQoSPriority, error) {
	state, err := c.getQoSBasicState(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]model.PortQoSPriority, 0, state.PortCount)
	for idx := 0; idx < state.PortCount; idx++ {
		out = append(out, model.PortQoSPriority{
			PortID:     idx + 1,
			Priority:   intAt(state.Priorities, idx),
			TrunkGroup: intAt(state.TrunkGroups, idx),
		})
	}

	return out, nil
}

func (c *Client) GetPortBandwidthControls(ctx context.Context) ([]model.PortBandwidthControl, error) {
	state, err := c.getBandwidthControlState(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]model.PortBandwidthControl, 0, state.PortCount)
	for idx := 0; idx < state.PortCount; idx++ {
		out = append(out, model.PortBandwidthControl{
			PortID:          idx + 1,
			IngressRateKbps: state.Entries[idx].IngressRateKbps,
			EgressRateKbps:  state.Entries[idx].EgressRateKbps,
			TrunkGroup:      state.Entries[idx].TrunkGroup,
		})
	}

	return out, nil
}

func (c *Client) GetPortStormControls(ctx context.Context) ([]model.PortStormControl, error) {
	state, err := c.getStormControlState(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]model.PortStormControl, 0, state.PortCount)
	for idx := 0; idx < state.PortCount; idx++ {
		out = append(out, model.PortStormControl{
			PortID:     idx + 1,
			Enabled:    state.Entries[idx].Enabled,
			RateKbps:   state.Entries[idx].RateKbps,
			StormTypes: append([]int(nil), state.Entries[idx].StormTypes...),
			TrunkGroup: state.Entries[idx].TrunkGroup,
		})
	}

	return out, nil
}

func (c *Client) UpdateQoSMode(ctx context.Context, mode int) error {
	if mode < 0 || mode > 2 {
		return fmt.Errorf("qos mode must be one of 0, 1, or 2")
	}

	form := url.Values{
		"rd_qosmode": []string{fmt.Sprintf("%d", mode)},
		"qosmode":    []string{"Apply"},
	}

	_, err := c.doProtectedForm(ctx, "/qos_mode_set.cgi", form)
	return err
}

func (c *Client) SetPortQoSPriority(ctx context.Context, portID int, priority int) error {
	if priority < 1 || priority > 4 {
		return fmt.Errorf("priority must be in the range 1-4")
	}

	state, err := c.getQoSBasicState(ctx)
	if err != nil {
		return err
	}
	if state.Mode != 0 {
		return fmt.Errorf("port QoS priority can only be changed when qos mode is port-based")
	}

	selectedPorts, err := selectedPortsForTrunk(portID, state.TrunkGroups)
	if err != nil {
		return err
	}

	form := url.Values{
		"port_queue": []string{fmt.Sprintf("%d", priority-1)},
		"apply":      []string{"Apply"},
	}
	addSelectedPorts(form, selectedPorts)

	_, err = c.doProtectedForm(ctx, "/qos_port_priority_set.cgi", form)
	return err
}

func (c *Client) SetPortBandwidthControl(ctx context.Context, portID int, ingressRateKbps int, egressRateKbps int) error {
	if ingressRateKbps < 0 || ingressRateKbps > 1000000 {
		return fmt.Errorf("ingress rate must be in the range 0-1000000")
	}
	if egressRateKbps < 0 || egressRateKbps > 1000000 {
		return fmt.Errorf("egress rate must be in the range 0-1000000")
	}

	state, err := c.getBandwidthControlState(ctx)
	if err != nil {
		return err
	}

	selectedPorts, err := selectedPortsForTrunk(portID, bandwidthTrunkGroups(state.Entries))
	if err != nil {
		return err
	}

	form := url.Values{
		"igrRate": []string{fmt.Sprintf("%d", ingressRateKbps)},
		"egrRate": []string{fmt.Sprintf("%d", egressRateKbps)},
		"applay":  []string{"Apply"},
	}
	addSelectedPorts(form, selectedPorts)

	_, err = c.doProtectedForm(ctx, "/qos_bandwidth_set.cgi", form)
	return err
}

func (c *Client) SetPortStormControl(ctx context.Context, portID int, enabled bool, rateKbps int, stormTypes []int) error {
	if enabled {
		if rateKbps < 1 || rateKbps > 1000000 {
			return fmt.Errorf("storm-control rate must be in the range 1-1000000 when enabled")
		}
		if len(stormTypes) == 0 {
			return fmt.Errorf("at least one storm type must be selected when storm control is enabled")
		}
	}

	encodedStormTypes, err := encodeStormTypes(stormTypes)
	if err != nil {
		return err
	}

	state, err := c.getStormControlState(ctx)
	if err != nil {
		return err
	}

	selectedPorts, err := selectedPortsForTrunk(portID, stormTrunkGroups(state.Entries))
	if err != nil {
		return err
	}

	form := url.Values{
		"state":  []string{stormStateValue(enabled)},
		"applay": []string{"Apply"},
	}
	if enabled {
		form.Set("rate", fmt.Sprintf("%d", rateKbps))
		for _, stormType := range encodedStormTypes {
			form.Add("stormType", fmt.Sprintf("%d", stormType))
		}
	}
	addSelectedPorts(form, selectedPorts)

	_, err = c.doProtectedForm(ctx, "/qos_storm_set.cgi", form)
	return err
}

type qosBasicState struct {
	PortCount   int
	Mode        int
	TrunkGroups []int
	Priorities  []int
}

type bandwidthControlState struct {
	PortCount int
	Entries   []model.PortBandwidthControl
}

type stormControlState struct {
	PortCount int
	Entries   []model.PortStormControl
}

func (c *Client) getQoSBasicState(ctx context.Context) (qosBasicState, error) {
	body, err := c.fetchProtectedPage(ctx, "/QosBasicRpm.htm")
	if err != nil {
		return qosBasicState{}, err
	}

	portCount, err := parser.ExtractInt(body, "portNumber")
	if err != nil {
		return qosBasicState{}, err
	}

	mode, err := parser.ExtractInt(body, "qosMode")
	if err != nil {
		return qosBasicState{}, err
	}

	trunkGroupsRaw, err := parser.ExtractArray(body, "pTrunk")
	if err != nil {
		return qosBasicState{}, err
	}

	prioritiesRaw, err := parser.ExtractArray(body, "pPri")
	if err != nil {
		return qosBasicState{}, err
	}

	return qosBasicState{
		PortCount:   portCount,
		Mode:        mode,
		TrunkGroups: intSlice(trunkGroupsRaw),
		Priorities:  intSlice(prioritiesRaw),
	}, nil
}

func (c *Client) getBandwidthControlState(ctx context.Context) (bandwidthControlState, error) {
	body, err := c.fetchProtectedPage(ctx, "/QosBandWidthControlRpm.htm")
	if err != nil {
		return bandwidthControlState{}, err
	}

	portCount, err := parser.ExtractInt(body, "portNumber")
	if err != nil {
		return bandwidthControlState{}, err
	}

	raw, err := parser.ExtractArray(body, "bcInfo")
	if err != nil {
		return bandwidthControlState{}, err
	}

	values := intSlice(raw)
	entries := make([]model.PortBandwidthControl, 0, portCount)
	for idx := 0; idx < portCount; idx++ {
		base := idx * 3
		entries = append(entries, model.PortBandwidthControl{
			PortID:          idx + 1,
			IngressRateKbps: intAt(values, base),
			EgressRateKbps:  intAt(values, base+1),
			TrunkGroup:      intAt(values, base+2),
		})
	}

	return bandwidthControlState{
		PortCount: portCount,
		Entries:   entries,
	}, nil
}

func (c *Client) getStormControlState(ctx context.Context) (stormControlState, error) {
	body, err := c.fetchProtectedPage(ctx, "/QosStormControlRpm.htm")
	if err != nil {
		return stormControlState{}, err
	}

	portCount, err := parser.ExtractInt(body, "portNumber")
	if err != nil {
		return stormControlState{}, err
	}

	raw, err := parser.ExtractArray(body, "scInfo")
	if err != nil {
		return stormControlState{}, err
	}

	values := intSlice(raw)
	entries := make([]model.PortStormControl, 0, portCount)
	for idx := 0; idx < portCount; idx++ {
		base := idx * 3
		rate := intAt(values, base)
		stormMask := intAt(values, base+1)

		entries = append(entries, model.PortStormControl{
			PortID:     idx + 1,
			Enabled:    rate != 0 && stormMask != 0,
			RateKbps:   rate,
			StormTypes: decodeStormTypes(stormMask),
			TrunkGroup: intAt(values, base+2),
		})
	}

	return stormControlState{
		PortCount: portCount,
		Entries:   entries,
	}, nil
}

func selectedPortsForTrunk(portID int, trunkGroups []int) ([]int, error) {
	if portID < 1 || portID > len(trunkGroups) {
		return nil, fmt.Errorf("port_id must be between 1 and %d", len(trunkGroups))
	}

	trunkGroup := trunkGroups[portID-1]
	if trunkGroup == 0 {
		return []int{portID}, nil
	}

	selected := make([]int, 0, len(trunkGroups))
	for idx, candidateGroup := range trunkGroups {
		if candidateGroup == trunkGroup {
			selected = append(selected, idx+1)
		}
	}
	return selected, nil
}

func bandwidthTrunkGroups(entries []model.PortBandwidthControl) []int {
	trunkGroups := make([]int, 0, len(entries))
	for _, entry := range entries {
		trunkGroups = append(trunkGroups, entry.TrunkGroup)
	}
	return trunkGroups
}

func stormTrunkGroups(entries []model.PortStormControl) []int {
	trunkGroups := make([]int, 0, len(entries))
	for _, entry := range entries {
		trunkGroups = append(trunkGroups, entry.TrunkGroup)
	}
	return trunkGroups
}

func addSelectedPorts(form url.Values, ports []int) {
	for _, portID := range ports {
		form.Set(fmt.Sprintf("sel_%d", portID), "1")
	}
}

func decodeStormTypes(mask int) []int {
	var stormTypes []int
	for _, candidate := range []int{1, 2, 4} {
		if mask&candidate != 0 {
			stormTypes = append(stormTypes, candidate)
		}
	}
	return stormTypes
}

func encodeStormTypes(stormTypes []int) ([]int, error) {
	seen := make(map[int]struct{}, len(stormTypes))
	encoded := make([]int, 0, len(stormTypes))
	for _, stormType := range stormTypes {
		if stormType != 1 && stormType != 2 && stormType != 4 {
			return nil, fmt.Errorf("storm type must be one of 1, 2, or 4")
		}
		if _, ok := seen[stormType]; ok {
			continue
		}
		seen[stormType] = struct{}{}
		encoded = append(encoded, stormType)
	}
	sort.Ints(encoded)
	return encoded, nil
}

func stormStateValue(enabled bool) string {
	if enabled {
		return "1"
	}
	return "0"
}
