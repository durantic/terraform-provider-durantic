package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccSecretsBackendResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSecretsBackendResourceConfig("test-backend", "http", "https://secrets.example.com", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_secrets_backend.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-backend"),
					),
					statecheck.ExpectKnownValue(
						"durantic_secrets_backend.test",
						tfjsonpath.New("backend_type"),
						knownvalue.StringExact("http"),
					),
					statecheck.ExpectKnownValue(
						"durantic_secrets_backend.test",
						tfjsonpath.New("url"),
						knownvalue.StringExact("https://secrets.example.com"),
					),
					statecheck.ExpectKnownValue(
						"durantic_secrets_backend.test",
						tfjsonpath.New("enabled"),
						knownvalue.Bool(true),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:                         "durantic_secrets_backend.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["durantic_secrets_backend.test"]
					if !ok {
						return "", fmt.Errorf("resource durantic_secrets_backend.test not found in state")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
			// Update testing
			{
				Config: testAccSecretsBackendResourceConfig("test-backend-updated", "http", "https://secrets2.example.com", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_secrets_backend.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-backend-updated"),
					),
					statecheck.ExpectKnownValue(
						"durantic_secrets_backend.test",
						tfjsonpath.New("url"),
						knownvalue.StringExact("https://secrets2.example.com"),
					),
					statecheck.ExpectKnownValue(
						"durantic_secrets_backend.test",
						tfjsonpath.New("enabled"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func testAccSecretsBackendResourceConfig(name, backendType, url string, enabled bool) string {
	return fmt.Sprintf(`
resource "durantic_secrets_backend" "test" {
  name         = %[1]q
  backend_type = %[2]q
  url          = %[3]q
  enabled      = %[4]t
}
`, name, backendType, url, enabled)
}
