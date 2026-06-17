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

func TestAccSecretResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccSecretResourceConfig("test-secret", "initial-secret-value", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_secret.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-secret"),
					),
					// value is sensitive — not checked directly
				},
			},
			// ImportState testing
			// Note: value cannot be imported as the API never returns it.
			{
				ResourceName:                         "durantic_secret.test",
				ImportState:                          true,
				ImportStateVerify:                    false, // value is never returned by API
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["durantic_secret.test"]
					if !ok {
						return "", fmt.Errorf("resource durantic_secret.test not found in state")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
			// Update testing
			{
				Config: testAccSecretResourceConfig("test-secret", "updated-secret-value", "a description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_secret.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("a description"),
					),
				},
			},
		},
	})
}

func testAccSecretResourceConfig(name, value, description string) string {
	return fmt.Sprintf(`
resource "durantic_secret" "test" {
  name        = %[1]q
  value       = %[2]q
  description = %[3]q
}
`, name, value, description)
}
