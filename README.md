# Terraform Provider for Durantic

A Terraform provider for managing [Durantic](https://durantic.io) infrastructure resources.

**Registry address:** `registry.durantic.io/durantic/durantic`

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

## Usage Example

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

resource "durantic_machine_role" "example" {
  name           = "web-server"
  description    = "Base web server configuration"
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

## Adding New Resources

1. Create `internal/provider/<resource_name>_resource.go`.
2. Define a resource struct with a `*durantic.APIClient` field and a Terraform model struct.
3. Implement `Metadata`, `Schema`, `Configure`, `Create`, `Read`, `Update`, `Delete`, and `ImportState`.
4. Register the resource in `provider.go` `Resources()`.
5. Add acceptance tests in `internal/provider/<resource_name>_resource_test.go`.
6. Run `TF_ACC=1 go test -v ./internal/provider/` to verify.

See `internal/provider/machine_role_resource.go` for a complete reference implementation.
