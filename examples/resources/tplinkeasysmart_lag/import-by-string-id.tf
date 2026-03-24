import {
  to = tplinkeasysmart_lag.lag1
  id = "1"
}

resource "tplinkeasysmart_lag" "lag1" {
  group_id = 1
  ports    = [1, 2]
}
