# Developing the Durantic Terraform Provider

Contributor guide for `terraform-provider-durantic`. For provider *usage*, see [README.md](README.md).

## Prerequisites

- [Go](https://go.dev/doc/install) >= 1.26
- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- Access to the private `github.com/durantic/controlplane-client-go` module. The provider's API client is generated there, so building from source requires read access to that repo (set `GOPRIVATE=github.com/durantic/*` and configure git credentials). External source builds are not currently supported — use the published Registry binaries instead.

## Build & install

```bash
make install      # builds and installs terraform-provider-durantic to $GOPATH/bin
```

## Run against a local build (dev overrides)

Point Terraform at your locally built binary so `terraform plan`/`apply` use it without `terraform init`. See **Local Development Setup** in [README.md](README.md#local-development-setup) for the `~/.terraformrc` `dev_overrides` block.

## Tests

```bash
make test         # unit tests
make testacc      # acceptance tests — create real resources against a live API
```

Acceptance tests require credentials:

```bash
export DURANTIC_API_TOKEN="your-token"
export DURANTIC_ENDPOINT="https://api.demo.durantic.dev"   # optional; this is the default
make testacc
```

> Acceptance tests create real resources and may incur cost. Run them against a non-production account.

## Docs & examples

Docs under `docs/` are generated from the schema and the files in `examples/` via [`terraform-plugin-docs`](https://github.com/hashicorp/terraform-plugin-docs):

```bash
make generate
```

Run `make generate` after changing any resource/data-source schema or its example, and commit the regenerated `docs/`.

## Adding a resource

See **Adding New Resources** in [README.md](README.md#adding-new-resources).
