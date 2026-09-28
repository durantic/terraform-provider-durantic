// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"testing"

	durantic "github.com/durantic/controlplane-client-go/durantic"
)

// A machine response from a control plane newer than the client this provider
// was generated from. Two published regressions live in this one payload:
//
//   - 1.1.0 failed to decode any response carrying a field it did not know
//     (`json: unknown field "environment_slug"`); `field_from_the_future` stands
//     in for whatever the control plane adds next.
//   - 1.1.1 read `wg_ip_address`, which the API stopped sending in
//     controlplane#203, so the attribute was silently null for every machine.
const machineResponseFromNewerControlPlane = `{
	"uuid": "8c8f1223-971e-4054-9eab-348912576210",
	"hostname": "u3-gate-a",
	"needs_provisioning": false,
	"created_at": "2026-09-28T12:40:00Z",
	"updated_at": "2026-09-28T12:41:00Z",
	"mesh_ip_address": "10.80.0.7",
	"mesh_network": {"uuid": "3f0c7a52-2b8e-4d6f-9a51-0f4a7c1e9b20", "name": "default", "network_cidr": "10.80.0.0/16"},
	"field_from_the_future": {"nested": true}
}`

func TestMapMachineResponse_NewerControlPlane(t *testing.T) {
	var machine durantic.MachineResponseSchema
	if err := json.Unmarshal([]byte(machineResponseFromNewerControlPlane), &machine); err != nil {
		t.Fatalf("decoding a response with an unknown field must not fail: %v", err)
	}

	var model MachineCommonModel
	if diags := mapMachineResponseToCommonModel(&machine, &model); diags.HasError() {
		t.Fatalf("mapping failed: %v", diags)
	}

	if got := model.MeshIPAddress.ValueString(); got != "10.80.0.7" {
		t.Errorf("mesh_ip_address = %q, want %q", got, "10.80.0.7")
	}
	if got := model.WgIPAddress.ValueString(); got != "10.80.0.7" {
		t.Errorf("deprecated wg_ip_address = %q, want it to alias mesh_ip_address", got)
	}
	if got := model.MeshNetworkUUID.ValueString(); got != "3f0c7a52-2b8e-4d6f-9a51-0f4a7c1e9b20" {
		t.Errorf("mesh_network_uuid = %q", got)
	}
}
