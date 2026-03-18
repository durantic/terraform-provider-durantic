# Terraform Provider for Durantic

A Terraform provider for managing [Durantic](https://durantic.io) infrastructure resources.

**Registry address:** `registry.durantic.io/durantic/durantic` [WIP]

## Requirements

| Tool      | Version  |
|-----------|----------|
| Terraform | >= 1.0   |
| Go        | >= 1.26 *(contributors only)* |

## Provider Configuration

### Schema

| Attribute             | Type   | Required | Description |
|-----------------------|--------|----------|-------------|
| `endpoint`            | string | No       | Durantic API endpoint URL. Defaults to `https://api.stage.durantic.dev`. |
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
      source  = "registry.durantic.io/durantic/durantic"
      version = "~> 1.0"
    }
  }
}

# Recommended: supply credentials via environment variables
provider "durantic" {}

# Or set explicitly (avoid committing api_token to source control)
provider "durantic" {
  endpoint  = "https://api.durantic.io"
  api_token = var.durantic_api_token
}
```

## API Client Dependency (controlplane-client-go)

The provider uses an OpenAPI-generated Go client:

- **Module:** `github.com/durantic/controlplane-client-go/durantic`
- **Authentication:** Bearer token via `Authorization` header


### Contributor setup

The module is consumed via a `replace` directive in `go.mod` that points to a local clone of `controlplane-client-go`. Contributors must clone that repository alongside this one and ensure the replace path is correct before building:

```go
replace github.com/durantic/controlplane-client-go/durantic => <path-to-your-local-clone>/durantic
```

## Resources

| Resource               | Description |
|------------------------|-------------|
| `durantic_machine_role` | Manages a Durantic machine role — a named configuration template (cloud-init data, merge priority, mesh requirement) applied to machines. |

## Data Sources

| Data Source       | Description |
|-------------------|-------------|
| `durantic_images` | Lists all images available to the account (own and official). |

## Examples

The [`examples/`](examples/) directory contains ready-to-use configurations:

| Path | Description |
|------|-------------|
| [`examples/provider/provider.tf`](examples/provider/provider.tf) | Provider configuration |
| [`examples/data-sources/durantic_images/data-source.tf`](examples/data-sources/durantic_images/data-source.tf) | Listing images and looking up by name |
| [`examples/resources/durantic_machine_role/resource.tf`](examples/resources/durantic_machine_role/resource.tf) | Minimal and full machine role resource examples |
| [`examples/resources/durantic_machine_role/import.sh`](examples/resources/durantic_machine_role/import.sh) | Importing an existing resource by UUID |

> These example files are also used to generate the documentation under `docs/`.

### Usage Example

```hcl
terraform {
  required_providers {
    durantic = {
      source  = "registry.durantic.io/durantic/durantic"
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
export DURANTIC_ENDPOINT="https://api.stage.durantic.dev"  # optional
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
    "registry.durantic.io/durantic/durantic" = "/home/USERNAME/go/bin"
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

# 2. Set credentials
export DURANTIC_API_TOKEN="your-token"

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
