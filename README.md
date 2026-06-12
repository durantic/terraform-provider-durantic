# Terraform Provider for Durantic

[![Tests](https://github.com/durantic/terraform-provider/actions/workflows/test.yml/badge.svg)](https://github.com/durantic/terraform-provider/actions/workflows/test.yml)

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

## Resources

| Resource               | Description |
|------------------------|-------------|
| `durantic_machine_config` | Manages desired configuration for an existing Durantic machine — mesh network assignment, role names, tunnel settings, and provisioning-related config flags. |
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

## TODO: Terraform provider test strategy

The repository will be made public so the provider can be published on the Terraform Registry. We do not expect outside contributors immediately, so CI can remain optimized for Durantic-maintained branches and trusted release workflows. The important split is between tests that are safe to run without secrets and tests that mutate a live controlplane account.

The tiers below are **coverage levels gated by the credentials and infrastructure available, not three separate test suites.** The same `go test` binary does more as more is provided: with no credentials it runs static checks and unit tests; adding a live API token activates live-account acceptance; the `e2e/` harness on a KVM runner adds real provisioning. Tier 1 needs no credentials and runs on every push and pull request; Tiers 2–3 mutate real infrastructure and run on trusted branches, scheduled runs, and release candidates.

How the tiers are selected to run:

- Pure-Go unit tests run under plain `go test ./...` — the plugin-testing framework auto-skips every `resource.Test` when `TF_ACC` is unset, so the no-credential job needs no token.
- Live-account acceptance tests use `resource.Test`, which only executes when `TF_ACC=1`, and needs the Terraform CLI installed plus a real `DURANTIC_API_TOKEN`.
- In practice this is three runs:
  - `go test ./...` → static/unit (Tier 1)
  - `TF_ACC=1 go test ./internal/provider/` with a **dev01 autotest token** → live-account acceptance (Tier 2)
  - `pytest -m terraform` in `e2e/` → provisioning e2e (Tier 3)

### Tier 1: Static and unit CI

Run on every push and pull request without Durantic credentials:

- [ ] Build the provider (`go build ./...`)
- [ ] Run Go unit tests without `TF_ACC` (`go test ./...`)
- [ ] Run formatting and lint checks (`gofmt`, `golangci-lint`)
- [ ] Run docs/code generation (`make generate`) and fail on an unexpected git diff
- [ ] Keep `TestPollProvision_*` in this tier; these tests mock provision polling and require no live infrastructure

Before the repo is public, make sure dependencies needed by this tier are public or otherwise available without private GitHub credentials. In particular, the OpenAPI client module (`github.com/durantic/controlplane-client-go/durantic`) must not prevent a clean public build. This is a hard gate: it blocks both this tier and Registry publishing, since CI currently fetches that module with a private GitHub App token (`GOPRIVATE`).

### Tier 2: Live-account acceptance CI

Run on trusted Durantic branches, scheduled runs, and release candidates with a dedicated `terraform-provider-autotest` account. Selected by providing `DURANTIC_API_TOKEN`/`DURANTIC_ENDPOINT` so the `TestAcc*` tests activate under `TF_ACC=1`:

- [ ] Continue running Terraform Plugin Testing with `TF_ACC=1 go test -v -cover ./internal/provider/`
- [ ] Use `DURANTIC_API_TOKEN` and `DURANTIC_ENDPOINT` for the target environment (the dedicated dev01 autotest account, not shared stage)
- [ ] Use stable public base-image URLs for image data source tests instead of static image UUID secrets. For example, `ghcr.io/durantic/linux-ubuntu-25.10:latest` should remain available across environments.
- [ ] Refactor image data source acceptance tests so UUID/name lookups are derived from the image found by `docker_image_url`, rather than configured through `DURANTIC_TEST_IMAGE_UUID` or `DURANTIC_TEST_IMAGE_NAME`.
- [ ] Do not require static machine fixture secrets for normal provider acceptance CI. A machine record is created by agent/QEMU registration, not ordinary API CRUD, so dynamically registered machines belong in the e2e tier.
- [ ] Keep only `durantic_machine` not-found behavior in this tier; cover successful machine lookup in the Terraform e2e suite. Do not boot QEMU machines from provider acceptance CI — that duplicates the e2e tier's job.
- [ ] Add acceptance coverage for resources currently registered by the provider but missing tests, especially `durantic_route` and `durantic_vip`
- [ ] Keep Terraform CLI version matrix coverage for scheduled/release workflows; use a single current Terraform version for ordinary trusted branch CI

### Tier 3: Provisioning e2e CI

These tests live entirely in the `e2e/` repo, as Python `pytest` cases under the `terraform` marker — they need QEMU/KVM and a real provisioning flow. The terraform-provider repo contributes only the provider binary (built from the ref under test), the `.tf` config exercised (reuse `examples/`), and a CI job that calls the reusable e2e workflow. The QGA verification step (reading state inside the booted VM) is a Python capability the Go `resource.Test` framework cannot replicate, so this is not a Go test; the detailed Go `durantic_machine_deployment` acceptance test remains the optional short-term bridge below, not the primary path.

- [ ] Add `pytest.mark.terraform` tests in `e2e/` for provider-driven flows
- [ ] Build the provider from the terraform-provider ref under test and expose it to Terraform via a CLI `dev_overrides` config (drive `plan`/`apply` directly; `terraform init` is skipped for a dev-overridden provider)
- [ ] Prefer dynamically registered QEMU machines from the e2e fixture over long-lived static machine UUIDs
- [ ] Cover successful `durantic_machine` data source lookups against the dynamically registered QEMU machine
- [ ] Exercise at least one real `durantic_machine_deployment` apply that provisions a VM, then verify the result through controlplane and qemu-guest-agent. Allow the apply at least 15 minutes (matches the provider's provision poll timeout and the e2e `PROVISION_TIMEOUT`).
- [ ] Keep these tests skippable for local development without QEMU or credentials, as the existing `t.Skip()` guards already allow
- [ ] Call the reusable e2e workflow from terraform-provider CI with `suite: terraform` on trusted branches and release candidates

If a short-term bridge is needed before dynamic QEMU registration is wired into the provider e2e suite, configure these temporary secrets for the existing `durantic_machine_deployment` acceptance test:

- [ ] `DURANTIC_TEST_MACHINE_DEPLOYMENT_UUID` — UUID of a dedicated disposable test machine
- [ ] `DURANTIC_TEST_MACHINE_DEPLOYMENT_MESH_NETWORK_UUID` — UUID of a mesh network to assign
- [ ] `DURANTIC_TEST_MACHINE_DEPLOYMENT_MESH_NETWORK_UUID2` — second mesh network UUID for the update-without-reprovision test step
- [ ] `DURANTIC_TEST_MACHINE_DEPLOYMENT_ROLE_NAMES` — comma-separated role names that exist in the test environment

## Durantic OpenAPI contract versioning

The provider imports the tagged generated Go client module (`github.com/durantic/controlplane-client-go/durantic`) and pins that module in `go.mod`. The client is generated from a checked-in OpenAPI artifact, not from an arbitrary deployed `/api/openapi.json` endpoint.

The canonical controlplane schema is exported to `controlplane/schema/openapi.json`. The live Django Ninja schema sets `info.version` from `CONTROLPLANE_VERSION`, which deployed environments receive from the controlplane image tag; local exports leave it unset so the committed artifact remains stable at `dev`.

The client repository keeps its source contract in `controlplane-client-go/openapi.json`, regenerates the generated `durantic/` package from that file, and tags releases as `durantic/v*` for Go module consumption.

The `/api/` path prefix is the compatibility boundary for published provider versions. Do not introduce `/api/v1/` paths unless the provider and generated client are released together with a migration plan.

## TODO: Publishing to the Terraform Registry

The following steps are required before the provider can be published to registry.terraform.io:

- [ ] Rename the GitHub repo from `terraform-provider` to `terraform-provider-durantic` (Registry requirement for provider repos)
- [ ] Make the repo public (registry.terraform.io only supports public repositories)
- [ ] Generate a GPG key pair (`gpg --full-generate-key`), register the public key on registry.terraform.io, and add the private key + passphrase as `GPG_PRIVATE_KEY` and `PASSPHRASE` secrets in the GitHub repo
- [ ] Sign in to registry.terraform.io with the `durantic` GitHub org, publish the provider via the UI — this sets up the webhook that makes the Registry watch for new releases
- [ ] Push a `v*` tag to trigger the release workflow and publish the first version
