# Copyright (c) HashiCorp, Inc.

data "durantic_machine" "gateway" {
  hostname = "gateway-01"
}

# Minimal example — only required fields
resource "durantic_external_bgp_peer" "minimal" {
  machine_uuid = data.durantic_machine.gateway.uuid
  name         = "upstream-transit"
  peer_ip      = "192.0.2.1"
  peer_asn     = 64512
}

# Full example with an MD5 password and import/export route policies
resource "durantic_route_policy_set" "upstream_in" {
  name = "upstream-in"
}

resource "durantic_route_policy_set" "upstream_out" {
  name = "upstream-out"
}

resource "durantic_external_bgp_peer" "example" {
  machine_uuid = data.durantic_machine.gateway.uuid
  name         = "upstream-transit"
  peer_ip      = "192.0.2.1"
  peer_asn     = 64512
  local_ip     = "192.0.2.2"
  enabled      = true
  password     = var.bgp_md5_password

  import_policy_uuid = durantic_route_policy_set.upstream_in.uuid
  export_policy_uuid = durantic_route_policy_set.upstream_out.uuid
}
