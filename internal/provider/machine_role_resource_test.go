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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccMachineRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMachineRoleResourceConfig("test-role", 100, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-role"),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("merge_priority"),
						knownvalue.Int64Exact(100),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("requires_mesh"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("template_data"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(""),
					),
				},
			},
			// ImportState testing
			{
				ResourceName:                         "durantic_machine_role.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["durantic_machine_role.test"]
					if !ok {
						return "", fmt.Errorf("resource durantic_machine_role.test not found in state")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
			},
			// Update testing
			{
				Config: testAccMachineRoleResourceConfig("test-role-updated", 50, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-role-updated"),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("merge_priority"),
						knownvalue.Int64Exact(50),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("requires_mesh"),
						knownvalue.Bool(true),
					),
				},
			},
			// Delete testing automatically occurs in resource.Test
		},
	})
}

func TestAccMachineRoleResource_WithDescription(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMachineRoleResourceConfigWithDescription("test-role-desc", "A test machine role", "some template data"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact("test-role-desc"),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("A test machine role"),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("template_data"),
						knownvalue.StringExact("some template data"),
					),
				},
			},
		},
	})
}

func TestAccMachineRoleResource_Defaults(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required fields and verify defaults
			{
				Config: testAccMachineRoleResourceConfigMinimal("test-role-minimal"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("merge_priority"),
						knownvalue.Int64Exact(100),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("requires_mesh"),
						knownvalue.Bool(false),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("template_data"),
						knownvalue.StringExact(""),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_role.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact(""),
					),
				},
			},
		},
	})
}

func testAccMachineRoleResourceConfig(name string, mergePriority int, requiresMesh bool) string {
	return fmt.Sprintf(`
resource "durantic_machine_role" "test" {
  name           = %[1]q
  merge_priority = %[2]d
  requires_mesh  = %[3]t
}
`, name, mergePriority, requiresMesh)
}

func testAccMachineRoleResourceConfigWithDescription(name, description, templateData string) string {
	return fmt.Sprintf(`
resource "durantic_machine_role" "test" {
  name          = %[1]q
  description   = %[2]q
  template_data = %[3]q
}
`, name, description, templateData)
}

func testAccMachineRoleResourceConfigMinimal(name string) string {
	return fmt.Sprintf(`
resource "durantic_machine_role" "test" {
  name = %[1]q
}
`, name)
}
