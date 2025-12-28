data "durantic_machines" "all" {}

# Output the total number of machines
output "total_machines" {
  description = "Total number of machines in the inventory"
  value       = length(data.durantic_machines.all.machines)
}

# Output online machines
output "online_machines" {
  description = "List of online machine hostnames"
  value       = [for m in data.durantic_machines.all.machines : m.hostname if m.is_online]
}

# Output machines in initrd
output "machines_in_initrd" {
  description = "Machines currently in installer initrd"
  value       = [for m in data.durantic_machines.all.machines : m.hostname if m.is_in_initrd]
}

# Output machines by role
output "web_servers" {
  description = "Machines with 'web' role"
  value = [
    for m in data.durantic_machines.all.machines :
    m.hostname if contains(m.role_names, "web")
  ]
}

# Create a map of hostname to UUID for easy reference
output "machine_map" {
  description = "Map of hostnames to UUIDs"
  value = {
    for m in data.durantic_machines.all.machines :
    m.hostname => m.uuid
  }
}
