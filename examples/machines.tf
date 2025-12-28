# Machine Management Examples
#
# This file demonstrates all machine-related resources and data sources
# in the Durantic Terraform provider.

# Configure the Durantic Provider
terraform {
  required_providers {
    durantic = {
      source = "registry.durantic.io/durantic/durantic"
    }
  }
}

provider "durantic" {
  # API endpoint - defaults to https://api.stage.durantic.dev if not set
  # endpoint = "https://api.stage.durantic.dev"

  # API token - required for authentication
  # Set via DURANTIC_API_TOKEN environment variable (recommended)
  # api_token = var.durantic_api_token
}

# ============================================================================
# DATA SOURCES - Query machine inventory
# ============================================================================

# Example 1: List all machines
data "durantic_machines" "all" {}

# Example 2: Get specific machine details
data "durantic_machine" "web_server" {
  uuid = "550e8400-e29b-41d4-a716-446655440000" # Replace with actual UUID
}

# ============================================================================
# RESOURCES - Manage machine configuration and provisioning
# ============================================================================

# Example 3: Create a mesh network for machines
resource "durantic_mesh_network" "production" {
  name         = "production-network"
  network_cidr = "10.0.0.0/16"
  is_default   = true
}

# Example 4: Import and configure a machine
# IMPORTANT: Machines must be imported first (they are auto-discovered)
# Run: terraform import durantic_machine.web1 <machine-uuid>
resource "durantic_machine" "web1" {
  # UUID is set via import

  # Assign to mesh network
  mesh_network_uuid = durantic_mesh_network.production.uuid

  # Assign roles
  role_names = ["web", "frontend", "nginx"]

  # Advertise routes
  advertised_routes = ["10.0.1.0/24"]

  # Set target disk for provisioning
  target_disk = "/dev/sda"
}

# Example 5: Machine with minimal configuration
resource "durantic_machine" "db1" {
  # Only roles, other fields use defaults
  role_names = ["database", "postgresql"]
}

# Example 6: Machine with Docker registry authentication
resource "durantic_machine" "app1" {
  role_names        = ["app"]
  mesh_network_uuid = durantic_mesh_network.production.uuid

  # Docker registry credentials (sensitive)
  docker_registry_auth = jsonencode({
    "registry.example.com" = {
      username = "deployer"
      password = var.docker_password
    }
  })
}

# Example 7: Provision a machine (rebuild OS)
resource "durantic_machine_provision" "rebuild_web1" {
  machine_uuid = durantic_machine.web1.uuid
  mode         = "rebuild"

  # Re-provision when configuration changes
  triggers = {
    config_hash = sha256(jsonencode({
      roles   = durantic_machine.web1.role_names
      network = durantic_machine.web1.mesh_network_uuid
    }))
  }
}

# Example 8: Provision machine in discovery mode
resource "durantic_machine_provision" "discover_new" {
  machine_uuid = "new-machine-uuid" # Machine in discovery
  mode         = "discover"
}

# Example 9: Clear provisioning flag
resource "durantic_machine_provision" "clear_flag" {
  machine_uuid = durantic_machine.db1.uuid
  mode         = "clear"
}

# Example 10: Provision multiple machines with for_each
locals {
  web_servers = {
    web1 = "web1-machine-uuid"
    web2 = "web2-machine-uuid"
    web3 = "web3-machine-uuid"
  }
}

resource "durantic_machine_provision" "web_fleet" {
  for_each = local.web_servers

  machine_uuid = each.value
  mode         = "rebuild"

  triggers = {
    deployment_id = "deploy-v2.1.0"
    timestamp     = timestamp()
  }
}

# ============================================================================
# OUTPUTS - Extract useful information
# ============================================================================

# Machine inventory statistics
output "inventory_stats" {
  description = "Machine inventory statistics"
  value = {
    total_machines    = length(data.durantic_machines.all.machines)
    online_machines   = length([for m in data.durantic_machines.all.machines : m if m.is_online])
    machines_in_initrd = length([for m in data.durantic_machines.all.machines : m if m.is_in_initrd])
  }
}

# Machines by role
output "machines_by_role" {
  description = "Machines grouped by role"
  value = {
    web = [
      for m in data.durantic_machines.all.machines :
      m.hostname if contains(m.role_names, "web")
    ]
    database = [
      for m in data.durantic_machines.all.machines :
      m.hostname if contains(m.role_names, "database")
    ]
    app = [
      for m in data.durantic_machines.all.machines :
      m.hostname if contains(m.role_names, "app")
    ]
  }
}

# Specific machine details
output "web1_details" {
  description = "Web server 1 configuration"
  value = {
    uuid              = durantic_machine.web1.uuid
    hostname          = durantic_machine.web1.hostname
    wg_ip             = durantic_machine.web1.wg_ip_address
    roles             = durantic_machine.web1.role_names
    needs_provisioning = durantic_machine.web1.needs_provisioning
    is_online         = durantic_machine.web1.is_online
  }
}

# Provisioning status
output "web1_provision_status" {
  description = "Web server 1 provisioning status"
  value = {
    provision_id      = durantic_machine_provision.rebuild_web1.id
    last_provisioned  = durantic_machine_provision.rebuild_web1.last_provisioned
    status            = durantic_machine_provision.rebuild_web1.status
  }
}

# Machine to IP mapping
output "machine_ips" {
  description = "Mapping of machine hostnames to IP addresses"
  value = {
    for m in data.durantic_machines.all.machines :
    m.hostname => m.wg_ip_address
  }
}

# ============================================================================
# VARIABLES (Optional)
# ============================================================================

variable "docker_password" {
  description = "Docker registry password"
  type        = string
  sensitive   = true
  default     = ""
}

# ============================================================================
# USAGE NOTES
# ============================================================================

# 1. Import machines before managing them:
#    terraform import durantic_machine.web1 <machine-uuid>
#    terraform import durantic_machine.db1 <machine-uuid>
#    terraform import durantic_machine.app1 <machine-uuid>

# 2. To get machine UUIDs, use the data source:
#    data "durantic_machines" "all" {}
#    Then check: data.durantic_machines.all.machines[*].uuid

# 3. Provisioning modes:
#    - "rebuild": Install OS (sets needs_provisioning=true, reboots)
#    - "discover": Discovery mode (sets needs_provisioning=false)
#    - "clear": Clear provisioning flag (no reboot)

# 4. Triggers force re-provisioning:
#    - Change any trigger value to provision again
#    - Use timestamp() for time-based provisioning
#    - Use config hashes to provision on config changes

# 5. Machine configuration updates:
#    - Changes to role_names, mesh_network_uuid, etc. update without provisioning
#    - To rebuild after config change, update provision triggers

# 6. Sensitive data:
#    - docker_registry_auth is marked as sensitive
#    - Stored encrypted in state file
#    - Use variables with sensitive = true
