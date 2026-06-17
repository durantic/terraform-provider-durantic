# Minimal example — policy set with no rules (default accept-all)
resource "durantic_route_policy_set" "minimal" {
  name = "my-policy"
}

# Full example — policy set with multiple rules
resource "durantic_route_policy_set" "example" {
  name           = "ingress-filter"
  description    = "Accept RFC1918 prefixes, reject everything else"
  default_action = "reject"
  advanced_mode  = false

  rules {
    sequence       = 10
    action         = "accept"
    match_type     = "prefix_list"
    match_prefixes = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
    description    = "Allow private address space"
  }

  rules {
    sequence          = 20
    action            = "accept"
    match_type        = "community"
    match_communities = ["65000:100"]
    description       = "Allow routes tagged with community 65000:100"
  }
}
