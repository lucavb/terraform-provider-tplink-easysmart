package providerdata

import (
	"sync"

	"github.com/lucavb/terraform-provider-tplink-easysmart/internal/client"
)

type Data struct {
	SwitchClient   client.Client
	vlanMutationMu sync.Mutex
}

func (d *Data) Client() client.Client {
	return d.SwitchClient
}

// VLANMutationLock serializes VLAN table changes. The switch Web UI exposes a
// single shared table, so concurrent Add/Modify requests can overwrite each
// other's view of that table.
func (d *Data) VLANMutationLock() *sync.Mutex {
	return &d.vlanMutationMu
}
