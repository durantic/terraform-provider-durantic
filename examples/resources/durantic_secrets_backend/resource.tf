# Minimal example — HTTP secrets backend
resource "durantic_secrets_backend" "minimal" {
  name         = "my-secrets-backend"
  backend_type = "http"
  url          = "https://secrets.example.com"
}

# Full example — HashiCorp Vault backend with configuration
resource "durantic_secrets_backend" "vault" {
  name         = "production-vault"
  backend_type = "vault"
  url          = "https://vault.example.com"
  enabled      = true
  ca_cert      = file("vault-ca.pem")

  config = {
    auth_path  = "auth/approle"
    mount_path = "secret"
    role_id    = var.vault_role_id
  }
}
