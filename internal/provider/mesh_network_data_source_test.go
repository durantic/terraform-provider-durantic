// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const (
	testMeshNetworkUUID     = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	testMeshNetworkListPath = "/api/connectivity/mesh-networks/"
)

func testMeshNetworkJSON(uuid, name, cidr string) string {
	return fmt.Sprintf(`{
		"uuid": %q,
		"name": %q,
		"network_cidr": %q,
		"is_default": true,
		"route_reflector_mode": false,
		"available_ip_count": 250,
		"machine_count": 4,
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-02T00:00:00Z",
		"some_future_field": "ignored"
	}`, uuid, name, cidr)
}

func TestMeshNetworkDataSourceRead(t *testing.T) {
	client := newFakeControlPlane(t, map[string]string{
		testMeshNetworkListPath: "[" +
			testMeshNetworkJSON(testMeshNetworkUUID, "prod-mesh", "10.0.0.0/24") + "," +
			testMeshNetworkJSON("dddddddd-dddd-dddd-dddd-dddddddddddd", "dup-mesh", "10.1.0.0/24") + "," +
			testMeshNetworkJSON("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", "dup-mesh", "10.2.0.0/24") + "]",
		testMeshNetworkListPath + testMeshNetworkUUID: testMeshNetworkJSON(testMeshNetworkUUID, "prod-mesh", "10.0.0.0/24"),
	})

	assertProdMesh := func(t *testing.T, config map[string]string) {
		t.Helper()
		resp := readTestDataSource(t, NewMeshNetworkDataSource(), client, config)
		assertNoDiagnostics(t, resp.Diagnostics)

		if got := stateString(t, resp, "uuid"); got != testMeshNetworkUUID {
			t.Errorf("uuid = %q, want %q", got, testMeshNetworkUUID)
		}
		if got := stateString(t, resp, "name"); got != "prod-mesh" {
			t.Errorf("name = %q", got)
		}
		if got := stateString(t, resp, "network_cidr"); got != "10.0.0.0/24" {
			t.Errorf("network_cidr = %q", got)
		}
		if !stateBool(t, resp, "is_default") {
			t.Error("is_default = false, want true")
		}
		if stateBool(t, resp, "route_reflector_mode") {
			t.Error("route_reflector_mode = true, want false")
		}
		if got := stateInt64(t, resp, "available_ip_count"); got != 250 {
			t.Errorf("available_ip_count = %d", got)
		}
		if got := stateInt64(t, resp, "machine_count"); got != 4 {
			t.Errorf("machine_count = %d", got)
		}
	}

	t.Run("by uuid", func(t *testing.T) {
		assertProdMesh(t, map[string]string{"uuid": testMeshNetworkUUID})
	})

	t.Run("by name", func(t *testing.T) {
		assertProdMesh(t, map[string]string{"name": "prod-mesh"})
	})

	t.Run("unknown name", func(t *testing.T) {
		resp := readTestDataSource(t, NewMeshNetworkDataSource(), client, map[string]string{"name": "missing"})
		assertErrorDiagnostic(t, resp.Diagnostics, "No Mesh Network Found")
	})

	t.Run("ambiguous name", func(t *testing.T) {
		resp := readTestDataSource(t, NewMeshNetworkDataSource(), client, map[string]string{"name": "dup-mesh"})
		assertErrorDiagnostic(t, resp.Diagnostics, "Multiple Mesh Networks Found")
	})

	t.Run("unknown uuid", func(t *testing.T) {
		resp := readTestDataSource(t, NewMeshNetworkDataSource(), client, map[string]string{"uuid": "does-not-exist"})
		assertErrorDiagnostic(t, resp.Diagnostics, "Error Reading Mesh Network")
	})

	t.Run("no selector", func(t *testing.T) {
		resp := readTestDataSource(t, NewMeshNetworkDataSource(), client, map[string]string{})
		assertErrorDiagnostic(t, resp.Diagnostics, "Invalid Mesh Network Lookup")
	})
}

func TestAccMeshNetworkDataSource_NotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "durantic_mesh_network" "missing" {
  name = "test-mesh-that-does-not-exist"
}
`,
				ExpectError: regexp.MustCompile("No Mesh Network Found"),
			},
		},
	})
}

// The mesh network is created in the same configuration, so no pre-existing
// infrastructure is needed. network_cidr is globally unique, so this test uses
// a range no other test does.
func TestAccMeshNetworkDataSource_ByNameAndUUID(t *testing.T) {
	const config = `
resource "durantic_mesh_network" "test" {
  name         = "test-mesh-lookup"
  network_cidr = "10.9.0.0/24"
}

data "durantic_mesh_network" "by_name" {
  name = durantic_mesh_network.test.name
}

data "durantic_mesh_network" "by_uuid" {
  uuid = durantic_mesh_network.test.uuid
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs("data.durantic_mesh_network.by_name", tfjsonpath.New("uuid"), "durantic_mesh_network.test", tfjsonpath.New("uuid"), compare.ValuesSame()),
					statecheck.CompareValuePairs("data.durantic_mesh_network.by_uuid", tfjsonpath.New("name"), "durantic_mesh_network.test", tfjsonpath.New("name"), compare.ValuesSame()),
					statecheck.ExpectKnownValue("data.durantic_mesh_network.by_name", tfjsonpath.New("network_cidr"), knownvalue.StringExact("10.9.0.0/24")),
				},
			},
		},
	})
}
