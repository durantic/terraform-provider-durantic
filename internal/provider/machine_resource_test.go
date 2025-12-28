// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMachineResource(t *testing.T) {
	// Replace with actual machine UUID from your test environment
	testMachineUUID := "test-machine-uuid"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ImportState testing
			{
				Config:            testAccMachineResourceConfig(testMachineUUID, ""),
				ResourceName:      "durantic_machine.test",
				ImportState:       true,
				ImportStateId:     testMachineUUID,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccMachineResourceConfig(testMachineUUID, `
  role_names        = ["test-role"]
  advertised_routes = ["10.0.0.0/24"]
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("durantic_machine.test", "uuid", testMachineUUID),
					resource.TestCheckResourceAttr("durantic_machine.test", "role_names.#", "1"),
					resource.TestCheckResourceAttr("durantic_machine.test", "role_names.0", "test-role"),
					resource.TestCheckResourceAttr("durantic_machine.test", "advertised_routes.#", "1"),
					resource.TestCheckResourceAttr("durantic_machine.test", "advertised_routes.0", "10.0.0.0/24"),
				),
			},
		},
	})
}

func testAccMachineResourceConfig(uuid, extraConfig string) string {
	return fmt.Sprintf(`
resource "durantic_machine" "test" {
  # UUID is set via import
%s
}

# Import with: terraform import durantic_machine.test %s
`, extraConfig, uuid)
}
