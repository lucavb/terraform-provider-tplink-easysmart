import {
  to = tplinkeasysmart_port_qos_priority.port2
  id = "2"
}

resource "tplinkeasysmart_port_qos_priority" "port2" {
  port_id  = 2
  priority = 3
}
