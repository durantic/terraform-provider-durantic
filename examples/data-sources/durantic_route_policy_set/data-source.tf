# Copyright (c) HashiCorp, Inc.

# Look up a route policy set by name.
data "durantic_route_policy_set" "upstream_in" {
  name = "upstream-in"
}

# Route policy sets can also be looked up by UUID.
data "durantic_route_policy_set" "by_uuid" {
  uuid = "a8ecf9c8-1721-424d-ba0f-87917dfc03d8"
}

# Use an existing policy set as the import policy of an external BGP peer.
resource "durantic_external_bgp_peer" "upstream" {
  machine_uuid       = data.durantic_machine.gateway.uuid
  name               = "upstream-transit"
  peer_ip            = "192.0.2.1"
  peer_asn           = 64512
  import_policy_uuid = data.durantic_route_policy_set.upstream_in.uuid
}

data "durantic_machine" "gateway" {
  hostname = "gateway-01"
}

output "upstream_in_rule_count" {
  value = length(data.durantic_route_policy_set.upstream_in.rules)
}
