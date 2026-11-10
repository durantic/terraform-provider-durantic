// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const (
	testVIPUUID     = "77777777-7777-7777-7777-777777777777"
	testVIPOtherID  = "88888888-8888-8888-8888-888888888888"
	testVIPListPath = "/api/connectivity/vips/"
)

func testVIPJSON(uuid, name string, withAddress bool) string {
	address := `"address": null,`
	if withAddress {
		address = `"address": "203.0.113.10",`
	}
	return fmt.Sprintf(`{
		"uuid": %q,
		"name": %q,
		%s
		"enabled": true,
		"health_check_type": "http",
		"health_check_target": "/healthz",
		"health_check_interval_seconds": 10,
		"health_check_timeout_seconds": 5,
		"health_check_healthy_threshold": 2,
		"health_check_unhealthy_threshold": 3,
		"health_check_holdoff_seconds": 30,
		"machine_count": 2,
		"machines": [
			{"uuid": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "hostname": "gw-01"},
			{"uuid": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "hostname": "gw-02", "mesh_ip_address": "10.0.0.2"}
		],
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-02T00:00:00Z",
		"some_future_field": "ignored"
	}`, uuid, name, address)
}

func TestVIPDataSourceRead(t *testing.T) {
	client := newFakeControlPlane(t, map[string]string{
		testVIPListPath:                     "[" + testVIPJSON(testVIPUUID, "web-vip", true) + "," + testVIPJSON(testVIPOtherID, "other-vip", true) + "," + testVIPJSON("99999999-9999-9999-9999-999999999999", "other-vip", true) + "]",
		testVIPListPath + testVIPUUID:       testVIPJSON(testVIPUUID, "web-vip", true),
		testVIPListPath + "no-address-uuid": testVIPJSON("no-address-uuid", "bare-vip", false),
	})

	t.Run("by uuid", func(t *testing.T) {
		resp := readTestDataSource(t, NewVIPDataSource(), client, map[string]string{"uuid": testVIPUUID})
		assertNoDiagnostics(t, resp.Diagnostics)

		if got := stateString(t, resp, "name"); got != "web-vip" {
			t.Errorf("name = %q, want web-vip", got)
		}
		if got := stateString(t, resp, "address"); got != "203.0.113.10" {
			t.Errorf("address = %q", got)
		}
		if got := stateString(t, resp, "health_check_type"); got != "http" {
			t.Errorf("health_check_type = %q", got)
		}
		if got := stateInt64(t, resp, "health_check_holdoff_seconds"); got != 30 {
			t.Errorf("health_check_holdoff_seconds = %d", got)
		}
		if got := stateInt64(t, resp, "machine_count"); got != 2 {
			t.Errorf("machine_count = %d", got)
		}
		if !stateBool(t, resp, "enabled") {
			t.Error("enabled = false, want true")
		}

		var machineUUIDs types.List
		assertNoDiagnostics(t, resp.State.GetAttribute(context.Background(), path.Root("machine_uuids"), &machineUUIDs))
		var got []string
		assertNoDiagnostics(t, machineUUIDs.ElementsAs(context.Background(), &got, false))
		want := []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Errorf("machine_uuids = %v, want %v", got, want)
		}
	})

	t.Run("by name", func(t *testing.T) {
		resp := readTestDataSource(t, NewVIPDataSource(), client, map[string]string{"name": "web-vip"})
		assertNoDiagnostics(t, resp.Diagnostics)

		if got := stateString(t, resp, "uuid"); got != testVIPUUID {
			t.Errorf("uuid = %q, want %q", got, testVIPUUID)
		}
	})

	t.Run("a VIP without an address has a null address", func(t *testing.T) {
		resp := readTestDataSource(t, NewVIPDataSource(), client, map[string]string{"uuid": "no-address-uuid"})
		assertNoDiagnostics(t, resp.Diagnostics)

		var addr types.String
		assertNoDiagnostics(t, resp.State.GetAttribute(context.Background(), path.Root("address"), &addr))
		if !addr.IsNull() {
			t.Errorf("address = %v, want null", addr)
		}
	})

	t.Run("unknown name", func(t *testing.T) {
		resp := readTestDataSource(t, NewVIPDataSource(), client, map[string]string{"name": "missing"})
		assertErrorDiagnostic(t, resp.Diagnostics, "No VIP Found")
	})

	t.Run("ambiguous name", func(t *testing.T) {
		resp := readTestDataSource(t, NewVIPDataSource(), client, map[string]string{"name": "other-vip"})
		assertErrorDiagnostic(t, resp.Diagnostics, "Multiple VIPs Found")
	})

	t.Run("unknown uuid", func(t *testing.T) {
		resp := readTestDataSource(t, NewVIPDataSource(), client, map[string]string{"uuid": "does-not-exist"})
		assertErrorDiagnostic(t, resp.Diagnostics, "Error Reading VIP")
	})

	t.Run("no selector", func(t *testing.T) {
		resp := readTestDataSource(t, NewVIPDataSource(), client, map[string]string{})
		assertErrorDiagnostic(t, resp.Diagnostics, "Invalid VIP Lookup")
	})

	t.Run("both selectors", func(t *testing.T) {
		resp := readTestDataSource(t, NewVIPDataSource(), client, map[string]string{"uuid": testVIPUUID, "name": "web-vip"})
		assertErrorDiagnostic(t, resp.Diagnostics, "Invalid VIP Lookup")
	})
}

func TestAccVIPDataSource_NotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "durantic_vip" "missing" {
  name = "test-vip-that-does-not-exist"
}
`,
				ExpectError: regexp.MustCompile("No VIP Found"),
			},
		},
	})
}

// The VIP is created in the same configuration, so no pre-existing
// infrastructure is needed.
func TestAccVIPDataSource_ByNameAndUUID(t *testing.T) {
	const config = `
resource "durantic_vip" "test" {
  name    = "test-vip-lookup"
  address = "203.0.113.77"
}

data "durantic_vip" "by_name" {
  name = durantic_vip.test.name
}

data "durantic_vip" "by_uuid" {
  uuid = durantic_vip.test.uuid
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs("data.durantic_vip.by_name", tfjsonpath.New("uuid"), "durantic_vip.test", tfjsonpath.New("uuid"), compare.ValuesSame()),
					statecheck.CompareValuePairs("data.durantic_vip.by_uuid", tfjsonpath.New("name"), "durantic_vip.test", tfjsonpath.New("name"), compare.ValuesSame()),
					statecheck.ExpectKnownValue("data.durantic_vip.by_name", tfjsonpath.New("address"), knownvalue.StringExact("203.0.113.77")),
				},
			},
		},
	})
}
