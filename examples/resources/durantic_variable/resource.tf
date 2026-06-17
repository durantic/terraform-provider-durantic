# Minimal example — only required fields
resource "durantic_variable" "minimal" {
  name  = "my-variable"
  value = "my-value"
}

# Full example with description
resource "durantic_variable" "example" {
  name        = "api-endpoint"
  value       = "https://api.example.com"
  description = "External API endpoint used by workloads"
}
