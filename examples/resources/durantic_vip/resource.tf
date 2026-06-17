# Minimal example — only required fields, all defaults apply
resource "durantic_vip" "minimal" {
  name    = "web-vip"
  address = "203.0.113.10"
}

# Full example with all optional fields configured
resource "durantic_vip" "example" {
  name    = "prod-vip"
  enabled = true
  address = "203.0.113.10"

  health_check_type                = "http"
  health_check_target              = "/healthz"
  health_check_interval_seconds    = 10
  health_check_timeout_seconds     = 5
  health_check_healthy_threshold   = 2
  health_check_unhealthy_threshold = 3
  health_check_holdoff_seconds     = 30

  machine_uuids = [
    "a8ecf9c8-1721-424d-ba0f-87917dfc03d8",
  ]
}
