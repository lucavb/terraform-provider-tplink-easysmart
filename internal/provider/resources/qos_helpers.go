package resources

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const qosModeID = "qos"

var (
	qosModeNameToValue = map[string]int{
		"port_based":       0,
		"dot1p_based":      1,
		"dscp_dot1p_based": 2,
	}
	qosModeValueToName = map[int]string{
		0: "port_based",
		1: "dot1p_based",
		2: "dscp_dot1p_based",
	}
	stormTypeNameToValue = map[string]int{
		"ul_frame":  1,
		"multicast": 2,
		"broadcast": 4,
	}
	stormTypeValueToName = map[int]string{
		1: "ul_frame",
		2: "multicast",
		4: "broadcast",
	}
)

func qosModeValue(name string) (int, bool) {
	value, ok := qosModeNameToValue[name]
	return value, ok
}

func qosModeName(value int) (string, bool) {
	name, ok := qosModeValueToName[value]
	return name, ok
}

func stormTypeSetFromValues(ctx context.Context, values []int) (types.Set, diag.Diagnostics) {
	names := make([]string, 0, len(values))
	for _, value := range values {
		name, ok := stormTypeValueToName[value]
		if !ok {
			var diags diag.Diagnostics
			diags.AddError("Unsupported storm type value", fmt.Sprintf("Observed unsupported storm type bit %d.", value))
			return types.Set{}, diags
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return types.SetValueFrom(ctx, types.StringType, names)
}

func stormTypeValuesFromSet(ctx context.Context, set types.Set, attribute string) ([]int, diag.Diagnostics) {
	var names []string
	diags := set.ElementsAs(ctx, &names, false)
	if diags.HasError() {
		return nil, diags
	}

	values := make([]int, 0, len(names))
	seen := make(map[int]struct{}, len(names))
	for _, name := range names {
		value, ok := stormTypeNameToValue[name]
		if !ok {
			diags.AddAttributeError(
				path.Root(attribute),
				"Invalid storm type",
				fmt.Sprintf("storm_types entries must be one of ul_frame, multicast, or broadcast; got %q.", name),
			)
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	sort.Ints(values)
	return values, diags
}
