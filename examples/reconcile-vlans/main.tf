terraform {
  required_providers {
    sops = {
      source = "carlpett/sops"
    }

    tplinkeasysmart = {
      source  = "registry.terraform.io/lucavb/tplink-easysmart"
      version = "= 0.1.0"
    }
  }
}

data "sops_file" "switch" {
  source_file = "${path.module}/secrets.sops.yaml"
}

locals {
  switch = yamldecode(data.sops_file.switch.raw)
}

provider "tplinkeasysmart" {
  host          = local.switch.host
  username      = local.switch.username
  password      = local.switch.password
  insecure_http = true
}

data "tplinkeasysmart_system_info" "this" {}

data "tplinkeasysmart_vlans" "this" {}

data "tplinkeasysmart_port_pvids" "this" {}

output "switch_identity" {
  value = {
    description = data.tplinkeasysmart_system_info.this.description
    firmware    = data.tplinkeasysmart_system_info.this.firmware
    hardware    = data.tplinkeasysmart_system_info.this.hardware
    ip          = data.tplinkeasysmart_system_info.this.ip
    mac         = data.tplinkeasysmart_system_info.this.mac
  }
}

output "vlan_inventory" {
  value = data.tplinkeasysmart_vlans.this
}

output "port_pvid_inventory" {
  value = data.tplinkeasysmart_port_pvids.this
}
