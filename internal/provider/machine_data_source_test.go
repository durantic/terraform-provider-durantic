// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccMachineDataSource_NotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "durantic_machine" "missing" {
  hostname     = "terraform-provider-missing-machine"
  not_found_ok = false
}
`,
				ExpectError: regexp.MustCompile("No Machine Found"),
			},
		},
	})
}

func TestAccMachineDataSource_NotFoundOk(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "durantic_machine" "missing" {
  hostname = "terraform-provider-missing-machine"
}
`,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.durantic_machine.missing",
						tfjsonpath.New("uuid"),
						knownvalue.Null(),
					),
				},
			},
		},
	})
}

func TestAccMachineDataSource_ByUUID(t *testing.T) {
	machineUUID := os.Getenv("DURANTIC_TEST_MACHINE_UUID")
	if machineUUID == "" {
		t.Skip("DURANTIC_TEST_MACHINE_UUID must be set for this acceptance test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "durantic_machine" "test" {
  uuid = %[1]q
}
`, machineUUID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.durantic_machine.test",
						tfjsonpath.New("uuid"),
						knownvalue.StringExact(machineUUID),
					),
					statecheck.ExpectKnownValue(
						"data.durantic_machine.test",
						tfjsonpath.New("hostname"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccMachineDataSource_ByHostname(t *testing.T) {
	machineHostname := os.Getenv("DURANTIC_TEST_MACHINE_HOSTNAME")
	if machineHostname == "" {
		t.Skip("DURANTIC_TEST_MACHINE_HOSTNAME must be set for this acceptance test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "durantic_machine" "test" {
  hostname = %[1]q
}
`, machineHostname),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.durantic_machine.test",
						tfjsonpath.New("hostname"),
						knownvalue.StringExact(machineHostname),
					),
					statecheck.ExpectKnownValue(
						"data.durantic_machine.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}
