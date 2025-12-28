# Example 1: Using environment variables (recommended for security)
# Set these before running terraform:
#   export DURANTIC_API_TOKEN="your-api-token-here"
#   export DURANTIC_ENDPOINT="https://api.stage.durantic.dev"  # optional

terraform {
  required_providers {
    durantic = {
      source  = "registry.durantic.io/durantic/durantic"
      version = "~> 1.0"
    }
  }
}

provider "durantic" {
  # Both endpoint and api_token will be read from environment variables
  # DURANTIC_ENDPOINT defaults to https://api.stage.durantic.dev if not set
  # DURANTIC_API_TOKEN is required
}

# Example 2: Explicit endpoint configuration
provider "durantic" {
  endpoint = "https://api.stage.durantic.dev"
  # api_token still read from DURANTIC_API_TOKEN environment variable
}

# Example 3: Using variables (not recommended for api_token - use env vars instead)
variable "durantic_endpoint" {
  description = "Durantic API endpoint URL"
  type        = string
  default     = "https://api.stage.durantic.dev"
}

variable "durantic_api_token" {
  description = "Durantic API token for authentication"
  type        = string
  sensitive   = true
}

provider "durantic" {
  endpoint  = var.durantic_endpoint
  api_token = var.durantic_api_token
}

# Example 4: Multiple provider configurations (aliases)
# Default provider
provider "durantic" {
  # Uses default endpoint and DURANTIC_API_TOKEN from environment
}

# Staging environment provider
provider "durantic" {
  alias    = "staging"
  endpoint = "https://api.stage.durantic.dev"
  # Uses DURANTIC_API_TOKEN from environment
}

# Production environment provider
provider "durantic" {
  alias    = "production"
  endpoint = "https://api.durantic.io"
  # Uses DURANTIC_API_TOKEN from environment (or set different token)
}

# Using aliased providers
resource "durantic_mesh_network" "staging_network" {
  provider = durantic.staging

  name         = "staging-network"
  network_cidr = "10.1.0.0/16"
}

resource "durantic_mesh_network" "production_network" {
  provider = durantic.production

  name         = "production-network"
  network_cidr = "10.2.0.0/16"
  is_default   = true
}

# Example 5: Using terraform.tfvars for configuration
# Create a terraform.tfvars file:
# durantic_endpoint = "https://api.stage.durantic.dev"
# durantic_api_token = "your-token-here"  # Better to use env var

# Example 6: Local development configuration
provider "durantic" {
  endpoint = "http://localhost:8000"
  # api_token from DURANTIC_API_TOKEN environment variable
}
