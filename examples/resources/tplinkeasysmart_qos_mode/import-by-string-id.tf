import {
  to = tplinkeasysmart_qos_mode.switch
  id = "qos"
}

resource "tplinkeasysmart_qos_mode" "switch" {
  mode = "port_based"
}
