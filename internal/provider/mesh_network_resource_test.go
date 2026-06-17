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

func TestAccMeshNetworkResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMeshNetworkResourceConfig("test-mesh-network", "10.0.0.0/24", false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-mesh-network"),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("network_cidr"),
						knownvalue.StringExact("10.0.0.0/24"),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("is_default"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("route_reflector_mode"),
						knownvalue.Bool(false),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:                         "durantic_mesh_network.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["durantic_mesh_network.test"]
					if !ok {
						return "", fmt.Errorf("resource durantic_mesh_network.test not found in state")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
			// Update testing (name and flags only — network_cidr is immutable)
			{
				Config: testAccMeshNetworkResourceConfig("test-mesh-network-updated", "10.0.0.0/24", true, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-mesh-network-updated"),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("is_default"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("route_reflector_mode"),
						knownvalue.Bool(true),
					),
				},
			},
			// Delete testing automatically occurs in resource.Test
		},
	})
}

func TestAccMeshNetworkResource_Defaults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required fields and verify defaults
			{
				Config: testAccMeshNetworkResourceConfigMinimal("test-mesh-minimal", "10.1.0.0/24"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-mesh-minimal"),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("network_cidr"),
						knownvalue.StringExact("10.1.0.0/24"),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("is_default"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("route_reflector_mode"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func TestAccMeshNetworkResource_RequiresReplace(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with initial CIDR
			{
				Config: testAccMeshNetworkResourceConfigMinimal("test-mesh-replace", "10.2.0.0/24"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("network_cidr"),
						knownvalue.StringExact("10.2.0.0/24"),
					),
				},
			},
			// Change CIDR — should trigger resource replacement
			{
				Config: testAccMeshNetworkResourceConfigMinimal("test-mesh-replace", "10.3.0.0/24"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("network_cidr"),
						knownvalue.StringExact("10.3.0.0/24"),
					),
				},
			},
		},
	})
}

func testAccMeshNetworkResourceConfig(name, networkCidr string, isDefault, routeReflectorMode bool) string {
	return fmt.Sprintf(`
resource "durantic_mesh_network" "test" {
  name                 = %[1]q
  network_cidr         = %[2]q
  is_default           = %[3]t
  route_reflector_mode = %[4]t
}
`, name, networkCidr, isDefault, routeReflectorMode)
}

func testAccMeshNetworkResourceConfigMinimal(name, networkCidr string) string {
	return fmt.Sprintf(`
resource "durantic_mesh_network" "test" {
  name         = %[1]q
  network_cidr = %[2]q
}
`, name, networkCidr)
}
