import {
  to = tplinkeasysmart_vlan_8021q.users
  id = "20"
}

resource "tplinkeasysmart_vlan_8021q" "users" {
  vlan_id        = 20
  name           = "Users"
  tagged_ports   = [1]
  untagged_ports = [2, 3]
}
