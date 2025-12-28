# Note: Machines must be imported first as they are auto-discovered
# See import.sh for import instructions

resource "durantic_machine" "example" {
  # UUID is set automatically via import

  # Assign to mesh network
  mesh_network_uuid = "network-uuid-here"

  # Configure roles
  role_names = ["web", "frontend"]

  # Advertise routes
  advertised_routes = ["10.0.0.0/24", "192.168.1.0/24"]

  # Set target disk for provisioning
  target_disk = "/dev/sda"

  # Docker registry authentication (sensitive)
  # docker_registry_auth = jsonencode({
  #   "registry.example.com" = {
  #     username = "user"
  #     password = "pass"
  #   }
  # })
}

# Example with minimal configuration
resource "durantic_machine" "minimal" {
  # Only set roles
  role_names = ["database"]
}

# Example using mesh network reference
resource "durantic_mesh_network" "main" {
  name         = "main-network"
  network_cidr = "10.0.0.0/16"
}

resource "durantic_machine" "with_network" {
  mesh_network_uuid = durantic_mesh_network.main.uuid
  role_names        = ["app"]
}

# Output machine information
output "machine_uuid" {
  description = "Machine UUID"
  value       = durantic_machine.example.uuid
}

output "machine_hostname" {
  description = "Machine hostname"
  value       = durantic_machine.example.hostname
}

output "machine_ip" {
  description = "WireGuard IP address"
  value       = durantic_machine.example.wg_ip_address
}

output "needs_provisioning" {
  description = "Whether machine needs provisioning"
  value       = durantic_machine.example.needs_provisioning
}
