terraform {
  required_providers {
    tplinkeasysmart = {
      source  = "registry.terraform.io/lucavb/tplink-easysmart"
      version = "= 0.1.0"
    }
  }
}

provider "tplinkeasysmart" {
  host          = "10.0.2.1"
  username      = "admin"
  password      = var.switch_password
  insecure_http = true
}

variable "switch_password" {
  type      = string
  sensitive = true
}
