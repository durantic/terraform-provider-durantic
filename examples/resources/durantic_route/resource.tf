# Minimal example — only required fields, all defaults apply
resource "durantic_route" "minimal" {
  name     = "internal-route"
  prefixes = ["10.0.0.0/8"]
}

# Full example with all optional fields configured
resource "durantic_route" "example" {
  name     = "prod-route"
  enabled  = true
  prefixes = ["10.0.0.0/8", "192.168.0.0/16"]

  machine_uuids = [
    "a8ecf9c8-1721-424d-ba0f-87917dfc03d8",
  ]
}
