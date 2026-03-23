package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/providerdata"
)

func configureClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) client.Client {
	if req.ProviderData == nil {
		return nil
	}

	providerData, ok := req.ProviderData.(*providerdata.Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", "The provider data type was not recognized.")
		return nil
	}

	return providerData.Client()
}

func int64SetFromInts(ctx context.Context, values []int) (types.Set, diag.Diagnostics) {
	int64s := make([]int64, 0, len(values))
	for _, value := range values {
		int64s = append(int64s, int64(value))
	}
	return types.SetValueFrom(ctx, types.Int64Type, int64s)
}

func intsFromSet(ctx context.Context, set types.Set) ([]int, diag.Diagnostics) {
	var values []int64
	diags := set.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, diags
	}

	out := make([]int, 0, len(values))
	for _, value := range values {
		out = append(out, int(value))
	}
	return out, diags
}

func stringifyID(value int64) types.String {
	return types.StringValue(fmt.Sprintf("%d", value))
}

func parseImportID(id string, attribute string, diags *diag.Diagnostics) (int64, bool) {
	value, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		diags.AddAttributeError(
			path.Root(attribute),
			"Invalid import identifier",
			fmt.Sprintf("Expected %s import ID to be a base-10 integer, got %q.", attribute, id),
		)
		return 0, false
	}

	return value, true
}
