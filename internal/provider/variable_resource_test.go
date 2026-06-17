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

func TestAccVariableResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccVariableResourceConfig("test-variable", "initial-value", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_variable.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-variable"),
					),
					statecheck.ExpectKnownValue(
						"durantic_variable.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("initial-value"),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:                         "durantic_variable.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["durantic_variable.test"]
					if !ok {
						return "", fmt.Errorf("resource durantic_variable.test not found in state")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
			// Update testing
			{
				Config: testAccVariableResourceConfig("test-variable", "updated-value", "a description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_variable.test",
						tfjsonpath.New("value"),
						knownvalue.StringExact("updated-value"),
					),
					statecheck.ExpectKnownValue(
						"durantic_variable.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("a description"),
					),
				},
			},
		},
	})
}

func testAccVariableResourceConfig(name, value, description string) string {
	return fmt.Sprintf(`
resource "durantic_variable" "test" {
  name        = %[1]q
  value       = %[2]q
  description = %[3]q
}
`, name, value, description)
}
