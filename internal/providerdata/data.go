package providerdata

import "github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"

type Data struct {
	SwitchClient client.Client
}

func (d *Data) Client() client.Client {
	return d.SwitchClient
}
