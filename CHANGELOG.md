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
