# Look up an existing machine by hostname.
data "durantic_machine" "gateway" {
  hostname = "gateway-01"
}

# The Cluster Wizard uses discovered IP addresses as selectable public IPs.
output "gateway_public_ips" {
  value = data.durantic_machine.gateway.public_ip_addresses
}

output "gateway_mesh_ip" {
  value = data.durantic_machine.gateway.wg_ip_address
}
