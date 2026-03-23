# Terraform Provider TP-Link Easy Smart

`terraform-provider-tplink-easysmart` is a community Terraform provider for TP-Link Easy Smart switches that expose the legacy web UI used by devices such as the TL-SG108PE family.

This provider is intentionally scoped as an MVP around the safest, best-understood switch surfaces:

- system information
- per-port settings
- 802.1Q VLAN inventory and management
- per-port PVID inventory and management

## Status

This provider is pre-1.0 software. It is intended for careful, operator-driven use on real hardware. Features that can disrupt management connectivity or require additional reverse-engineering remain intentionally out of scope.

## Requirements

- Terraform or OpenTofu compatible with modern Terraform providers
- network reachability to the switch web UI
- switch credentials for the web UI

## Installation

```hcl
terraform {
  required_providers {
    tplinkeasysmart = {
      source  = "lucavb/tplink-easysmart"
      version = "~> 0.1"
    }
  }
}
```

## Provider Configuration

```hcl
provider "tplinkeasysmart" {
  host          = "10.0.2.1"
  username      = "admin"
  password      = var.switch_password
  insecure_http = true
  timeout_seconds = 10
}
```

Arguments:

- `host`: switch hostname, IP, or full base URL
- `username`: web UI username
- `password`: web UI password
- `insecure_http`: use `http://` when `host` does not include a scheme; defaults to `true` for legacy Easy Smart devices
- `timeout_seconds`: HTTP timeout in seconds; defaults to `10`

## Supported Resources

- `tplinkeasysmart_vlan_8021q`
- `tplinkeasysmart_port_pvid`
- `tplinkeasysmart_port_setting`

## Supported Data Sources

- `tplinkeasysmart_system_info`
- `tplinkeasysmart_ports`
- `tplinkeasysmart_vlans`
- `tplinkeasysmart_port_pvids`

## Example

```hcl
provider "tplinkeasysmart" {
  host          = "10.0.2.1"
  username      = "admin"
  password      = var.switch_password
  insecure_http = true
}

data "tplinkeasysmart_system_info" "switch" {}

resource "tplinkeasysmart_vlan_8021q" "users" {
  vlan_id        = 20
  name           = "Users"
  tagged_ports   = [1]
  untagged_ports = [2, 3]
}

resource "tplinkeasysmart_port_pvid" "port2" {
  port_id = 2
  pvid    = 20
}
```

## Hardware Scope

This provider has been developed against the TP-Link Easy Smart web UI model and is currently targeted at switches that expose the same page structure and handlers. Tested support should be expanded over time, but until then you should assume compatibility is best for the TL-SG108PE-class devices used during development.

## Safety Notes

- Avoid managing uplink or management ports until you have proven a no-op plan on your switch.
- Prefer importing or reconciling live state before making the first write.
- Do not assume unimplemented UI features are safe to automate yet.

## Local Examples

- `examples/connection-check`: verify connectivity and provider configuration against a live switch
- `examples/reconcile-vlans`: inventory and import existing VLAN/PVID state before making changes

## Development

```sh
make test
make lint
make testacc
```

Acceptance tests require live switch credentials via `TPLINK_EASYSMART_*` environment variables.

## License

MIT. See `LICENSE`.
