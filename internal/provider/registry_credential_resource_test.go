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

func TestAccRegistryCredentialResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRegistryCredentialResourceConfig("test-registry", "registry.example.com", "testuser", "testpassword", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_registry_credential.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-registry"),
					),
					statecheck.ExpectKnownValue(
						"durantic_registry_credential.test",
						tfjsonpath.New("registry_url"),
						knownvalue.StringExact("registry.example.com"),
					),
					statecheck.ExpectKnownValue(
						"durantic_registry_credential.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("testuser"),
					),
				},
			},
			// ImportState testing
			// Note: password cannot be imported as the API never returns it.
			{
				ResourceName:                         "durantic_registry_credential.test",
				ImportState:                          true,
				ImportStateVerify:                    false, // password is never returned by API
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["durantic_registry_credential.test"]
					if !ok {
						return "", fmt.Errorf("resource durantic_registry_credential.test not found in state")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
			// Update testing
			{
				Config: testAccRegistryCredentialResourceConfig("test-registry-updated", "registry.example.com", "newuser", "newpassword", "a description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_registry_credential.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-registry-updated"),
					),
					statecheck.ExpectKnownValue(
						"durantic_registry_credential.test",
						tfjsonpath.New("username"),
						knownvalue.StringExact("newuser"),
					),
					statecheck.ExpectKnownValue(
						"durantic_registry_credential.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("a description"),
					),
				},
			},
		},
	})
}

func testAccRegistryCredentialResourceConfig(name, registryURL, username, password, description string) string {
	return fmt.Sprintf(`
resource "durantic_registry_credential" "test" {
  name         = %[1]q
  registry_url = %[2]q
  username     = %[3]q
  password     = %[4]q
  description  = %[5]q
}
`, name, registryURL, username, password, description)
}
