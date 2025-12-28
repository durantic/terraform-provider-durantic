// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccMeshNetworkResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMeshNetworkResourceConfig("test-network", "10.0.0.0/16", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-network"),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("network_cidr"),
						knownvalue.StringExact("10.0.0.0/16"),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("is_default"),
						knownvalue.Bool(false),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:      "durantic_mesh_network.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing
			{
				Config: testAccMeshNetworkResourceConfig("test-network-updated", "10.0.0.0/16", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-network-updated"),
					),
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("is_default"),
						knownvalue.Bool(true),
					),
				},
			},
			// Delete testing automatically occurs in resource.Test
		},
	})
}

func TestAccMeshNetworkResource_DefaultValue(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without is_default and verify it defaults to false
			{
				Config: testAccMeshNetworkResourceConfigWithoutDefault("test-network-default", "10.1.0.0/16"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_mesh_network.test",
						tfjsonpath.New("is_default"),
						knownvalue.Bool(false),
					),
				},
			},
		},
	})
}

func testAccMeshNetworkResourceConfig(name, cidr string, isDefault bool) string {
	return fmt.Sprintf(`
resource "durantic_mesh_network" "test" {
  name         = %[1]q
  network_cidr = %[2]q
  is_default   = %[3]t
}
`, name, cidr, isDefault)
}

func testAccMeshNetworkResourceConfigWithoutDefault(name, cidr string) string {
	return fmt.Sprintf(`
resource "durantic_mesh_network" "test" {
  name         = %[1]q
  network_cidr = %[2]q
}
`, name, cidr)
}
