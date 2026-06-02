// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccMachineConfigResource(t *testing.T) {
	machineUUID := os.Getenv("DURANTIC_TEST_MACHINE_CONFIG_UUID")
	meshNetworkUUID := os.Getenv("DURANTIC_TEST_MACHINE_CONFIG_MESH_NETWORK_UUID")
	roleNamesEnv := os.Getenv("DURANTIC_TEST_MACHINE_CONFIG_ROLE_NAMES")
	if machineUUID == "" || meshNetworkUUID == "" || roleNamesEnv == "" {
		t.Skip("DURANTIC_TEST_MACHINE_CONFIG_UUID, DURANTIC_TEST_MACHINE_CONFIG_MESH_NETWORK_UUID, and DURANTIC_TEST_MACHINE_CONFIG_ROLE_NAMES must be set for this acceptance test")
	}

	roleNames := splitTestRoleNames(roleNamesEnv)
	roleNameChecks := make([]knownvalue.Check, len(roleNames))
	for i, roleName := range roleNames {
		roleNameChecks[i] = knownvalue.StringExact(roleName)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccMachineConfigResourceConfig(machineUUID, meshNetworkUUID, roleNames),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"durantic_machine_config.test",
						tfjsonpath.New("machine_uuid"),
						knownvalue.StringExact(machineUUID),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_config.test",
						tfjsonpath.New("mesh_network_uuid"),
						knownvalue.StringExact(meshNetworkUUID),
					),
					statecheck.ExpectKnownValue(
						"durantic_machine_config.test",
						tfjsonpath.New("role_names"),
						knownvalue.ListExact(roleNameChecks),
					),
				},
			},
		},
	})
}

func testAccMachineConfigResourceConfig(machineUUID, meshNetworkUUID string, roleNames []string) string {
	quotedRoleNames := make([]string, len(roleNames))
	for i, roleName := range roleNames {
		quotedRoleNames[i] = fmt.Sprintf("%q", roleName)
	}

	return fmt.Sprintf(`
resource "durantic_machine_config" "test" {
  machine_uuid      = %[1]q
  mesh_network_uuid = %[2]q
  role_names        = [%[3]s]
}
`, machineUUID, meshNetworkUUID, strings.Join(quotedRoleNames, ", "))
}

func splitTestRoleNames(value string) []string {
	parts := strings.Split(value, ",")
	roleNames := make([]string, 0, len(parts))
	for _, part := range parts {
		roleName := strings.TrimSpace(part)
		if roleName != "" {
			roleNames = append(roleNames, roleName)
		}
	}
	return roleNames
}
