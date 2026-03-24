resource "tplinkeasysmart_lag" "lag1" {
  group_id = 1
  ports    = [1, 2]
}
