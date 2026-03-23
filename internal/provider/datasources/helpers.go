package datasources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/providerdata"
)

func configureClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) client.Client {
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

func int64ListFromInts(ctx context.Context, values []int) (types.List, diag.Diagnostics) {
	int64s := make([]int64, 0, len(values))
	for _, value := range values {
		int64s = append(int64s, int64(value))
	}

	return types.ListValueFrom(ctx, types.Int64Type, int64s)
}
