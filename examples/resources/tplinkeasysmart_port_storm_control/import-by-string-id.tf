import {
  to = tplinkeasysmart_port_storm_control.port2
  id = "2"
}

resource "tplinkeasysmart_port_storm_control" "port2" {
  port_id     = 2
  enabled     = true
  rate_kbps   = 1000
  storm_types = ["broadcast", "multicast"]
}
