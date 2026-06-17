package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccRoutePolicySetResource(t *testing.T) {
	rName := acctest.RandString(6)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read — minimal (no rules)
			{
				Config: testAccRoutePolicySetResourceConfigMinimal("test-policy-" + rName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_route_policy_set.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-policy-"+rName),
					),
					statecheck.ExpectKnownValue(
						"durantic_route_policy_set.test",
						tfjsonpath.New("default_action"),
						knownvalue.StringExact("accept"),
					),
					statecheck.ExpectKnownValue(
						"durantic_route_policy_set.test",
						tfjsonpath.New("advanced_mode"),
						knownvalue.Bool(false),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:                         "durantic_route_policy_set.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["durantic_route_policy_set.test"]
					if !ok {
						return "", fmt.Errorf("resource durantic_route_policy_set.test not found in state")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
			// Update — add rules
			{
				Config: testAccRoutePolicySetResourceConfigWithRules("test-policy-" + rName + "-upd"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_route_policy_set.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-policy-"+rName+"-upd"),
					),
					statecheck.ExpectKnownValue(
						"durantic_route_policy_set.test",
						tfjsonpath.New("rules").AtSliceIndex(0).AtMapKey("sequence"),
						knownvalue.Int64Exact(10),
					),
					statecheck.ExpectKnownValue(
						"durantic_route_policy_set.test",
						tfjsonpath.New("rules").AtSliceIndex(0).AtMapKey("action"),
						knownvalue.StringExact("accept"),
					),
				},
			},
		},
	})
}

func testAccRoutePolicySetResourceConfigMinimal(name string) string {
	return fmt.Sprintf(`
resource "durantic_route_policy_set" "test" {
  name = %[1]q
}
`, name)
}

func testAccRoutePolicySetResourceConfigWithRules(name string) string {
	return fmt.Sprintf(`
resource "durantic_route_policy_set" "test" {
  name           = %[1]q
  default_action = "reject"

  rules {
    sequence       = 10
    action         = "accept"
    match_type     = "prefix_list"
    match_prefixes = ["10.0.0.0/8"]
  }
}
`, name)
}
