# Terraform Provider for Durantic

[![Tests](https://github.com/durantic/terraform-provider-durantic/actions/workflows/test.yml/badge.svg)](https://github.com/durantic/terraform-provider-durantic/actions/workflows/test.yml)

A Terraform provider for managing [Durantic](https://durantic.io) infrastructure resources.

**Registry address:** `registry.terraform.io/durantic/durantic`

## Requirements

| Tool      | Version  |
|-----------|----------|
| Terraform | >= 1.0   |
| Go        | >= 1.26 *(contributors only)* |

## Provider Configuration

### Schema

| Attribute             | Type   | Required | Description |
|-----------------------|--------|----------|-------------|
| `endpoint`            | string | No       | Durantic API endpoint URL. Defaults to `https://api.demo.durantic.dev`. |
| `api_token`           | string | No*      | API token for authentication. Marked sensitive. *Required at runtime. |
| `insecure_skip_verify`| bool   | No       | Skip TLS certificate verification. Defaults to `false`. **Development/testing only.** |

### Environment Variables

| Variable                      | Overrides attribute     |
|-------------------------------|-------------------------|
| `DURANTIC_ENDPOINT`           | `endpoint`              |
| `DURANTIC_API_TOKEN`          | `api_token`             |
| `DURANTIC_INSECURE_SKIP_VERIFY` | `insecure_skip_verify` |

`DURANTIC_API_TOKEN` is required; the provider will fail to configure if it is absent from both the attribute and the environment.

### Example

```hcl
terraform {
  required_providers {
    durantic = {
      source  = "durantic/durantic"
      version = "~> 1.0"
    }
  }
}

# Recommended: supply credentials via environment variables
provider "durantic" {}

# Or set explicitly (avoid committing api_token to source control)
provider "durantic" {
  endpoint  = "https://api.demo.durantic.dev"
  api_token = var.durantic_api_token
}
```

## API Client Dependency (controlplane-client-go)

The provider uses an OpenAPI-generated Go client:

- **Module:** `github.com/durantic/controlplane-client-go/durantic`
- **Authentication:** Bearer token via `Authorization` header

## Resources

| Resource               | Description |
|------------------------|-------------|
| `durantic_machine_config` | Manages desired configuration for an existing Durantic machine — mesh network assignment, role names, tunnel settings, and provisioning-related config flags. |
| `durantic_machine_deployment` | Manages configuration and OS provisioning for an existing Durantic machine — applies role, mesh, and tunnel settings and triggers a full OS install, blocking until the provision run reaches a terminal state. |
| `durantic_machine_role` | Manages a Durantic machine role — a named configuration template (cloud-init data, merge priority, mesh requirement, optional VIP association) applied to machines. |
| `durantic_mesh_network` | Manages a Durantic mesh network — a WireGuard-based overlay network with a defined CIDR block, default flag, and route reflector mode. |
| `durantic_route`        | Manages a Durantic route — a named set of network prefixes with optional machine associations and enable/disable control. |
| `durantic_vip`          | Manages a Durantic Virtual IP (VIP) — an IP address with configurable health checks and optional machine associations. |
| `durantic_registry_credential` | Manages a Durantic registry credential — authentication details (URL, username, password) for a container image registry. |
| `durantic_secrets_backend` | Manages a Durantic secrets backend — an external secrets store (Vault or HTTP) that supplies secrets to workloads. |
| `durantic_secret`       | Manages a Durantic account secret — a named sensitive value stored encrypted. The value is write-only (never returned by the API). |
| `durantic_variable`     | Manages a Durantic account variable — a named key/value pair available to workloads. |
| `durantic_route_policy_set` | Manages a Durantic route policy set — an ordered list of BGP route policy rules with nested `rules` blocks. |

## Data Sources

| Data Source       | Description |
|-------------------|-------------|
| `durantic_machine` | Looks up an existing machine by UUID or hostname, including role, mesh, public IP, and status fields. |
| `durantic_image` | Looks up a single image by UUID, name, or Docker image URL. |
| `durantic_images` | Lists all images available to the account (own and official). |

## Examples

The [`examples/`](examples/) directory contains ready-to-use configurations:

| Path | Description |
|------|-------------|
| [`examples/provider/provider.tf`](examples/provider/provider.tf) | Provider configuration |
| [`examples/data-sources/durantic_machine/data-source.tf`](examples/data-sources/durantic_machine/data-source.tf) | Looking up a machine and its public/mesh IPs |
| [`examples/data-sources/durantic_image/data-source.tf`](examples/data-sources/durantic_image/data-source.tf) | Looking up a single image by Docker image URL or name |
| [`examples/data-sources/durantic_images/data-source.tf`](examples/data-sources/durantic_images/data-source.tf) | Listing images and looking up by name |
| [`examples/resources/durantic_machine_config/resource.tf`](examples/resources/durantic_machine_config/resource.tf) | Assigning Terraform-created roles and a mesh network to an existing machine |
| [`examples/resources/durantic_machine_config/import.sh`](examples/resources/durantic_machine_config/import.sh) | Importing existing machine config by machine UUID |
| [`examples/resources/durantic_machine_deployment/resource.tf`](examples/resources/durantic_machine_deployment/resource.tf) | Provisioning an existing machine — assigning roles and a mesh network and triggering an OS install |
| [`examples/resources/durantic_machine_deployment/import.sh`](examples/resources/durantic_machine_deployment/import.sh) | Importing an existing machine deployment by machine UUID |
| [`examples/resources/durantic_machine_role/resource.tf`](examples/resources/durantic_machine_role/resource.tf) | Minimal and full machine role resource examples |
| [`examples/resources/durantic_machine_role/import.sh`](examples/resources/durantic_machine_role/import.sh) | Importing an existing machine role by UUID |
| [`examples/resources/durantic_mesh_network/resource.tf`](examples/resources/durantic_mesh_network/resource.tf) | Minimal and full mesh network resource examples |
| [`examples/resources/durantic_mesh_network/import.sh`](examples/resources/durantic_mesh_network/import.sh) | Importing an existing mesh network by UUID |
| [`examples/resources/durantic_route/resource.tf`](examples/resources/durantic_route/resource.tf) | Minimal and full route resource examples |
| [`examples/resources/durantic_route/import.sh`](examples/resources/durantic_route/import.sh) | Importing an existing route by UUID |
| [`examples/resources/durantic_vip/resource.tf`](examples/resources/durantic_vip/resource.tf) | Minimal and full VIP resource examples |
| [`examples/resources/durantic_vip/import.sh`](examples/resources/durantic_vip/import.sh) | Importing an existing VIP by UUID |
| [`examples/resources/durantic_registry_credential/resource.tf`](examples/resources/durantic_registry_credential/resource.tf) | Minimal and full registry credential examples |
| [`examples/resources/durantic_registry_credential/import.sh`](examples/resources/durantic_registry_credential/import.sh) | Importing an existing registry credential by UUID |
| [`examples/resources/durantic_secrets_backend/resource.tf`](examples/resources/durantic_secrets_backend/resource.tf) | HTTP and Vault secrets backend examples |
| [`examples/resources/durantic_secrets_backend/import.sh`](examples/resources/durantic_secrets_backend/import.sh) | Importing an existing secrets backend by UUID |
| [`examples/resources/durantic_secret/resource.tf`](examples/resources/durantic_secret/resource.tf) | Minimal and full secret resource examples |
| [`examples/resources/durantic_secret/import.sh`](examples/resources/durantic_secret/import.sh) | Importing an existing secret by UUID (value must be set manually after import) |
| [`examples/resources/durantic_variable/resource.tf`](examples/resources/durantic_variable/resource.tf) | Minimal and full variable resource examples |
| [`examples/resources/durantic_variable/import.sh`](examples/resources/durantic_variable/import.sh) | Importing an existing variable by UUID |
| [`examples/resources/durantic_route_policy_set/resource.tf`](examples/resources/durantic_route_policy_set/resource.tf) | Policy set with and without nested rules |
| [`examples/resources/durantic_route_policy_set/import.sh`](examples/resources/durantic_route_policy_set/import.sh) | Importing an existing route policy set by UUID |

> These example files are also used to generate the documentation under `docs/`.

### Usage Example

```hcl
terraform {
  required_providers {
    durantic = {
      source  = "durantic/durantic"
      version = "~> 1.0"
    }
  }
}

provider "durantic" {
  # api_token read from DURANTIC_API_TOKEN
}

# Look up an image by name
data "durantic_images" "all" {}

locals {
  ubuntu_image = one([
    for img in data.durantic_images.all.images : img
    if strcontains(img.name, "ubuntu-25")
  ])
}

resource "durantic_machine_role" "example" {
  name           = "web-server"
  description    = "Base web server configuration"
  image_uuid     = local.ubuntu_image.uuid
  merge_priority = 100
  requires_mesh  = false

  template_data = <<-EOT
    #cloud-config
    packages:
      - nginx
    runcmd:
      - systemctl enable --now nginx
  EOT
}

output "machine_role_uuid" {
  value = durantic_machine_role.example.uuid
}
```

## Developing the Provider

See [README_dev.md](README_dev.md) for full development instructions.

Set credentials before running acceptance tests:

```shell
export DURANTIC_API_TOKEN="your-token"
export DURANTIC_ENDPOINT="https://api.demo.durantic.dev"  # optional
make testacc
```

## Local Development Setup

### Build & install the provider locally

```bash
make install
# equivalent: go install -v ./...
# Installs binary to $GOPATH/bin (e.g. ~/go/bin/terraform-provider-durantic)
```

### Configure `.terraformrc` for dev overrides

Terraform reads `~/.terraformrc` to override provider resolution. Create or edit this file to point Terraform at your locally built binary. Replace USERNAME with your home directory:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/durantic/durantic" = "/home/USERNAME/go/bin"
  }

  # For all other providers, install them directly from their origin
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

The path must point to the directory containing the `terraform-provider-durantic` binary (i.e. `$GOPATH/bin`). With dev overrides active, `terraform init` is not required for the overridden provider — just run `terraform plan`/`apply` directly.

### End-to-end workflow

```bash
# 1. Build and install
make install

# 2. Set credentials and API endpoint
export DURANTIC_API_TOKEN="your-token"
export DURANTIC_ENDPOINT="https://api.dev.durantic.io"

# 3. Point to a local .tf file and run
cd /path/to/your/tf/config
terraform plan
```

## Adding New Resources

1. Create `internal/provider/<resource_name>_resource.go`.
2. Define a resource struct with a `*durantic.APIClient` field and a Terraform model struct.
3. Implement `Metadata`, `Schema`, `Configure`, `Create`, `Read`, `Update`, `Delete`, and `ImportState`.
4. Register the resource in `provider.go` `Resources()`.
5. Add acceptance tests in `internal/provider/<resource_name>_resource_test.go`.
6. Run `TF_ACC=1 go test -v ./internal/provider/` to verify.

See `internal/provider/machine_role_resource.go` for a complete reference implementation.

## API compatibility

The provider is generated from the Durantic platform's OpenAPI contract, so its resource and data-source coverage tracks the API directly.

The Durantic API is **stable and backward compatible**: there is no path-based versioning (no `/api/v1/`), and published provider releases preserve compatibility with the platform. Pin the provider with a `~>` version constraint as usual; any behavioral changes are called out in the [release notes](https://github.com/durantic/terraform-provider-durantic/releases).
