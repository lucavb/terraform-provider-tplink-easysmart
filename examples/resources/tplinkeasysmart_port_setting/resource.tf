resource "tplinkeasysmart_port_setting" "port2" {
  port_id             = 2
  enabled             = true
  speed_config        = 4
  flow_control_config = 1
}
