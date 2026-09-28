## 1.2.0 (September 28, 2026)

FEATURES:

* data-source/durantic_machine, resource/durantic_machine_config, resource/durantic_machine_deployment: new `mesh_ip_address` attribute. The control plane renamed its `wg_ip_address` API field to `mesh_ip_address`, and providers up to 1.1.1 read the old name, so `wg_ip_address` has been `null` for every machine against current control planes even when a mesh network and IP are assigned.

DEPRECATIONS:

* data-source/durantic_machine, resource/durantic_machine_config, resource/durantic_machine_deployment: `wg_ip_address` is deprecated in favour of `mesh_ip_address`. It is populated from the same value, so existing configurations resolve again without changes; it will be removed in a future major version.

BUG FIXES:

* resource/durantic_registry_credential, resource/durantic_vip: create requests now set their fields by name rather than by position. The regenerated API client orders constructor parameters differently, and positional calls would have sent the password as the username and swapped a VIP's name and address. Released versions were not affected.
* provider: the generated API client is updated to the current API surface and still ignores response fields it does not know about, so this release stays compatible with newer control planes, as 1.1.1 did.

## 1.1.1 (September 2, 2026)

BUG FIXES:

* provider: every resource and data source failed with `json: unknown field "..."` against control planes newer than the API the provider was generated from — for 1.1.0, any control plane at or after `2026.08.18` (the machine resources: `unknown field "environment_slug"`). The generated API client rejected response fields it did not know about, so an additive API change broke already-published providers. The client is regenerated to ignore unknown response fields (`controlplane-client-go` `durantic/v0.0.4`, same API surface as `v0.0.3`); this release changes nothing else. Providers 1.0.0 and 1.1.0 are **not** forward-compatible with newer control planes; upgrade.

## 1.1.0 (August 5, 2026)

BREAKING CHANGES:

* provider: the default API `endpoint` changed from `https://api.demo.durantic.dev` to `https://api.app.durantic.dev` — the new Durantic production environment. Configurations that omit `endpoint` now target production. If your account lives on the demo environment, set `endpoint = "https://api.demo.durantic.dev"` (or `DURANTIC_ENDPOINT`) explicitly, otherwise existing demo API tokens will fail authentication against the new default.

## 1.0.0 (June 17, 2026)

Initial release of the Durantic Terraform provider, for managing Durantic platform infrastructure through the Durantic API. Configure the provider with a Durantic API token (`DURANTIC_API_TOKEN`) and, optionally, a custom API `endpoint` (`DURANTIC_ENDPOINT`).

FEATURES:

* **New Resource:** `durantic_machine_config`
* **New Resource:** `durantic_machine_deployment`
* **New Resource:** `durantic_machine_role`
* **New Resource:** `durantic_mesh_network`
* **New Resource:** `durantic_registry_credential`
* **New Resource:** `durantic_route`
* **New Resource:** `durantic_route_policy_set`
* **New Resource:** `durantic_secret`
* **New Resource:** `durantic_secrets_backend`
* **New Resource:** `durantic_variable`
* **New Resource:** `durantic_vip`
* **New Data Source:** `durantic_image`
* **New Data Source:** `durantic_images`
* **New Data Source:** `durantic_machine`
