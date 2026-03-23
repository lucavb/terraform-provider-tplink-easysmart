import {
  to = tplinkeasysmart_port_pvid.port2
  id = "2"
}

resource "tplinkeasysmart_port_pvid" "port2" {
  port_id = 2
  pvid    = 20
}
