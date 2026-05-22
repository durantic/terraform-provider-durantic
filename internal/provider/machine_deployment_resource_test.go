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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccMachineDeploymentResource(t *testing.T) {
	machineUUID := os.Getenv("DURANTIC_TEST_MACHINE_DEPLOYMENT_UUID")
	meshNetworkUUID := os.Getenv("DURANTIC_TEST_MACHINE_DEPLOYMENT_MESH_NETWORK_UUID")
	meshNetworkUUID2 := os.Getenv("DURANTIC_TEST_MACHINE_DEPLOYMENT_MESH_NETWORK_UUID2")
	roleNamesEnv := os.Getenv("DURANTIC_TEST_MACHINE_DEPLOYMENT_ROLE_NAMES")
	if machineUUID == "" || meshNetworkUUID == "" || roleNamesEnv == "" {
		t.Skip("DURANTIC_TEST_MACHINE_DEPLOYMENT_UUID, DURANTIC_TEST_MACHINE_DEPLOYMENT_MESH_NETWORK_UUID, and DURANTIC_TEST_MACHINE_DEPLOYMENT_ROLE_NAMES must be set for this acceptance test")
	}

	roleNames := splitTestRoleNames(roleNamesEnv)
	roleNameChecks := make([]knownvalue.Check, len(roleNames))
	for i, roleName := range roleNames {
		roleNameChecks[i] = knownvalue.StringExact(roleName)
	}

	steps := []resource.TestStep{
		// Step 1: create — apply config and provision, wait for completed
		{
			Config: testAccMachineDeploymentConfig(machineUUID, meshNetworkUUID, roleNames, "v1"),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"durantic_machine_deployment.test",
					tfjsonpath.New("machine_uuid"),
					knownvalue.StringExact(machineUUID),
				),
				statecheck.ExpectKnownValue(
					"durantic_machine_deployment.test",
					tfjsonpath.New("mesh_network_uuid"),
					knownvalue.StringExact(meshNetworkUUID),
				),
				statecheck.ExpectKnownValue(
					"durantic_machine_deployment.test",
					tfjsonpath.New("role_names"),
					knownvalue.ListExact(roleNameChecks),
				),
				statecheck.ExpectKnownValue(
					"durantic_machine_deployment.test",
					tfjsonpath.New("provision_status"),
					knownvalue.StringExact("completed"),
				),
				statecheck.ExpectKnownValue(
					"durantic_machine_deployment.test",
					tfjsonpath.New("provision_uuid"),
					knownvalue.NotNull(),
				),
			},
		},
	}

	// Step 2: update mesh_network_uuid — no re-provision (provision_uuid must not change)
	if meshNetworkUUID2 != "" {
		steps = append(steps, resource.TestStep{
			Config: testAccMachineDeploymentConfig(machineUUID, meshNetworkUUID2, roleNames, "v1"),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(
					"durantic_machine_deployment.test",
					tfjsonpath.New("mesh_network_uuid"),
					knownvalue.StringExact(meshNetworkUUID2),
				),
				statecheck.ExpectKnownValue(
					"durantic_machine_deployment.test",
					tfjsonpath.New("provision_status"),
					knownvalue.StringExact("completed"),
				),
			},
		})
	}

	// Step 3: bump force_provision — triggers re-provision
	steps = append(steps, resource.TestStep{
		Config: testAccMachineDeploymentConfig(machineUUID, meshNetworkUUID, roleNames, "v2"),
		ConfigStateChecks: []statecheck.StateCheck{
			statecheck.ExpectKnownValue(
				"durantic_machine_deployment.test",
				tfjsonpath.New("force_provision"),
				knownvalue.StringExact("v2"),
			),
			statecheck.ExpectKnownValue(
				"durantic_machine_deployment.test",
				tfjsonpath.New("provision_status"),
				knownvalue.StringExact("completed"),
			),
			statecheck.ExpectKnownValue(
				"durantic_machine_deployment.test",
				tfjsonpath.New("provision_uuid"),
				knownvalue.NotNull(),
			),
		},
	})

	// Step 4: import by machine_uuid
	steps = append(steps, resource.TestStep{
		ResourceName:                         "durantic_machine_deployment.test",
		ImportState:                          true,
		ImportStateVerify:                    true,
		ImportStateVerifyIdentifierAttribute: "machine_uuid",
		ImportStateVerifyIgnore:              []string{"force_provision", "provision_uuid", "provision_status"},
		ImportStateIdFunc: func(s *terraform.State) (string, error) {
			rs, ok := s.RootModule().Resources["durantic_machine_deployment.test"]
			if !ok {
				return "", fmt.Errorf("resource durantic_machine_deployment.test not found in state")
			}
			return rs.Primary.Attributes["machine_uuid"], nil
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps:                    steps,
	})
}

func testAccMachineDeploymentConfig(machineUUID, meshNetworkUUID string, roleNames []string, forceProvision string) string {
	quotedRoleNames := make([]string, len(roleNames))
	for i, roleName := range roleNames {
		quotedRoleNames[i] = fmt.Sprintf("%q", roleName)
	}

	return fmt.Sprintf(`
resource "durantic_machine_deployment" "test" {
  machine_uuid      = %[1]q
  mesh_network_uuid = %[2]q
  role_names        = [%[3]s]
  force_provision   = %[4]q
}
`, machineUUID, meshNetworkUUID, strings.Join(quotedRoleNames, ", "), forceProvision)
}
