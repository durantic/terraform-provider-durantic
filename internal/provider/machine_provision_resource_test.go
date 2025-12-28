// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMachineProvisionResource(t *testing.T) {
	// Replace with actual machine UUID from your test environment
	testMachineUUID := "test-machine-uuid"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccMachineProvisionResourceConfig(testMachineUUID, "discover", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("durantic_machine_provision.test", "machine_uuid", testMachineUUID),
					resource.TestCheckResourceAttr("durantic_machine_provision.test", "mode", "discover"),
					resource.TestCheckResourceAttrSet("durantic_machine_provision.test", "id"),
					resource.TestCheckResourceAttrSet("durantic_machine_provision.test", "last_provisioned"),
					resource.TestCheckResourceAttrSet("durantic_machine_provision.test", "status"),
				),
			},
			// Update (force replacement) with different mode
			{
				Config: testAccMachineProvisionResourceConfig(testMachineUUID, "clear", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("durantic_machine_provision.test", "mode", "clear"),
				),
			},
			// Update with triggers (force replacement)
			{
				Config: testAccMachineProvisionResourceConfig(testMachineUUID, "rebuild", `
  triggers = {
    config_version = "v2"
  }
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("durantic_machine_provision.test", "mode", "rebuild"),
					resource.TestCheckResourceAttr("durantic_machine_provision.test", "triggers.config_version", "v2"),
				),
			},
		},
	})
}

func testAccMachineProvisionResourceConfig(machineUUID, mode, extraConfig string) string {
	return fmt.Sprintf(`
resource "durantic_machine_provision" "test" {
  machine_uuid = %[1]q
  mode         = %[2]q
%[3]s
}
`, machineUUID, mode, extraConfig)
}
