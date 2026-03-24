import {
  to = tplinkeasysmart_port_bandwidth_control.port2
  id = "2"
}

resource "tplinkeasysmart_port_bandwidth_control" "port2" {
  port_id           = 2
  ingress_rate_kbps = 100000
  egress_rate_kbps  = 50000
}
