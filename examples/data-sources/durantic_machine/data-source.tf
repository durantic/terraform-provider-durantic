data "durantic_machine" "example" {
  uuid = "550e8400-e29b-41d4-a716-446655440000" # Replace with actual machine UUID
}

# Output basic machine information
output "machine_info" {
  description = "Basic machine information"
  value = {
    hostname          = data.durantic_machine.example.hostname
    uuid              = data.durantic_machine.example.uuid
    is_online         = data.durantic_machine.example.is_online
    needs_provisioning = data.durantic_machine.example.needs_provisioning
  }
}

# Output network configuration
output "machine_network" {
  description = "Machine network configuration"
  value = {
    mesh_network_name = data.durantic_machine.example.mesh_network_name
    mesh_network_cidr = data.durantic_machine.example.mesh_network_cidr
    wg_ip_address     = data.durantic_machine.example.wg_ip_address
    advertised_routes = data.durantic_machine.example.advertised_routes
  }
}

# Output machine roles
output "machine_roles" {
  description = "Assigned roles"
  value       = data.durantic_machine.example.role_names
}

# Use machine data to configure other resources
resource "local_file" "machine_config" {
  content = jsonencode({
    hostname   = data.durantic_machine.example.hostname
    ip_address = data.durantic_machine.example.wg_ip_address
    roles      = data.durantic_machine.example.role_names
  })
  filename = "${path.module}/machine-${data.durantic_machine.example.hostname}.json"
}
