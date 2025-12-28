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
  # Can also be set via DURANTIC_ENDPOINT environment variable
  # endpoint = "https://api.stage.durantic.dev"

  # API token - required for authentication
  # Can also be set via DURANTIC_API_TOKEN environment variable (recommended)
  # api_token = var.durantic_api_token
}

# Example 1: Basic mesh network
resource "durantic_mesh_network" "basic" {
  name         = "basic-network"
  network_cidr = "10.0.0.0/16"
}

# Example 2: Mesh network with explicit is_default
resource "durantic_mesh_network" "production" {
  name         = "production-network"
  network_cidr = "10.1.0.0/16"
  is_default   = true
}

# Example 3: Multiple mesh networks with different CIDR ranges
resource "durantic_mesh_network" "development" {
  name         = "development-network"
  network_cidr = "10.2.0.0/16"
  is_default   = false
}

resource "durantic_mesh_network" "staging" {
  name         = "staging-network"
  network_cidr = "10.3.0.0/16"
  is_default   = false
}

# Example 4: Using smaller subnet
resource "durantic_mesh_network" "small" {
  name         = "small-network"
  network_cidr = "172.16.0.0/24"
}

# Example 5: Using outputs to reference computed attributes
output "production_network_id" {
  description = "UUID of the production network"
  value       = durantic_mesh_network.production.uuid
}

output "production_network_stats" {
  description = "Statistics for the production network"
  value = {
    uuid          = durantic_mesh_network.production.uuid
    available_ips = durantic_mesh_network.production.available_ip_count
    machine_count = durantic_mesh_network.production.machine_count
    created_at    = durantic_mesh_network.production.created_at
    updated_at    = durantic_mesh_network.production.updated_at
  }
}

# Example 6: Using variables for configuration
variable "environment" {
  description = "Environment name"
  type        = string
  default     = "dev"
}

variable "network_cidr_base" {
  description = "Base CIDR for network allocation"
  type        = string
  default     = "10.10.0.0/16"
}

resource "durantic_mesh_network" "dynamic" {
  name         = "${var.environment}-mesh-network"
  network_cidr = var.network_cidr_base
  is_default   = var.environment == "production" ? true : false
}

# Example 7: Using data source for reference (if implemented)
# data "durantic_mesh_network" "existing" {
#   uuid = "550e8400-e29b-41d4-a716-446655440000"
# }

# Example 8: Import existing mesh network
# To import an existing mesh network:
# terraform import durantic_mesh_network.imported 550e8400-e29b-41d4-a716-446655440000
