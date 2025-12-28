# Provision a machine in rebuild mode (install OS)
resource "durantic_machine_provision" "rebuild_web" {
  machine_uuid = "machine-uuid-here" # Replace with actual machine UUID
  mode         = "rebuild"
}

# Provision a machine in discovery mode
resource "durantic_machine_provision" "discover" {
  machine_uuid = "machine-uuid-here"
  mode         = "discover"
}

# Clear provisioning flag without rebooting
resource "durantic_machine_provision" "clear_flag" {
  machine_uuid = "machine-uuid-here"
  mode         = "clear"
}

# Use triggers to force re-provisioning when configuration changes
resource "durantic_machine" "web" {
  role_names = ["web", "frontend"]
}

resource "durantic_machine_provision" "rebuild_on_config_change" {
  machine_uuid = durantic_machine.web.uuid
  mode         = "rebuild"

  # Force re-provision when machine configuration changes
  triggers = {
    config_version = durantic_machine.web.updated_at
    roles          = join(",", durantic_machine.web.role_names)
  }
}

# Use triggers with timestamp to force periodic rebuilds
resource "durantic_machine_provision" "periodic_rebuild" {
  machine_uuid = "machine-uuid-here"
  mode         = "rebuild"

  # Change this value to trigger a new rebuild
  triggers = {
    rebuild_version = "v1"
  }
}

# Example with multiple provision modes for different machines
locals {
  machines = {
    web1 = {
      uuid = "web1-uuid"
      mode = "rebuild"
    }
    web2 = {
      uuid = "web2-uuid"
      mode = "discover"
    }
    db1 = {
      uuid = "db1-uuid"
      mode = "rebuild"
    }
  }
}

resource "durantic_machine_provision" "fleet" {
  for_each = local.machines

  machine_uuid = each.value.uuid
  mode         = each.value.mode

  triggers = {
    deployment_id = "deploy-123"
  }
}

# Output provisioning status
output "provision_id" {
  description = "Provisioning action ID"
  value       = durantic_machine_provision.rebuild_web.id
}

output "last_provisioned" {
  description = "When the machine was last provisioned"
  value       = durantic_machine_provision.rebuild_web.last_provisioned
}

output "provision_status" {
  description = "Status of the provisioning action"
  value       = durantic_machine_provision.rebuild_web.status
}
