import {
  to = tplinkeasysmart_igmp_snooping.switch
  id = "igmp_snooping"
}

resource "tplinkeasysmart_igmp_snooping" "switch" {
  enabled                    = true
  report_message_suppression = false
}
