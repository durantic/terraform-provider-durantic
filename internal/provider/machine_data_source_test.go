// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMachineDataSource(t *testing.T) {
	// Replace with actual machine UUID from your test environment
	testMachineUUID := "test-machine-uuid"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccMachineDataSourceConfig(testMachineUUID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.durantic_machine.test", "uuid", testMachineUUID),
					resource.TestCheckResourceAttrSet("data.durantic_machine.test", "hostname"),
					resource.TestCheckResourceAttrSet("data.durantic_machine.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.durantic_machine.test", "updated_at"),
				),
			},
		},
	})
}

func testAccMachineDataSourceConfig(uuid string) string {
	return fmt.Sprintf(`
data "durantic_machine" "test" {
  uuid = %[1]q
}
`, uuid)
}
