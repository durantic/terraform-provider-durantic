// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func testExternalBGPPeerModel() *ExternalBGPPeerResourceModel {
	return &ExternalBGPPeerResourceModel{
		MachineUUID:      types.StringValue("11111111-1111-1111-1111-111111111111"),
		Name:             types.StringValue("test-peer"),
		PeerIP:           types.StringValue("192.0.2.1"),
		PeerASN:          types.Int64Value(64512),
		Enabled:          types.BoolValue(true),
		LocalIP:          types.StringNull(),
		Password:         types.StringNull(),
		ImportPolicyUUID: types.StringNull(),
		ExportPolicyUUID: types.StringNull(),
	}
}

func marshalToMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func TestBuildCreateExternalBGPPeerRequest(t *testing.T) {
	t.Run("required fields are set by name", func(t *testing.T) {
		m := marshalToMap(t, buildCreateExternalBGPPeerRequest(testExternalBGPPeerModel()))

		want := map[string]any{
			"machine_uuid": "11111111-1111-1111-1111-111111111111",
			"name":         "test-peer",
			"peer_ip":      "192.0.2.1",
			"peer_asn":     float64(64512),
			"enabled":      true,
		}
		for k, v := range want {
			if m[k] != v {
				t.Errorf("%s = %v, want %v", k, m[k], v)
			}
		}
		for _, k := range []string{"local_ip", "import_policy_uuid", "export_policy_uuid"} {
			if v, present := m[k]; present && v != nil {
				t.Errorf("%s = %v, want it unset", k, v)
			}
		}
		// The API spec defaults password to "" and the generated constructor
		// applies that, so an unconfigured password goes out as "" — what the
		// server assumes when it's omitted anyway.
		if v := m["password"]; v != nil && v != "" {
			t.Errorf("password = %v, want unset or the empty API default", v)
		}
	})

	t.Run("optional fields are sent when set", func(t *testing.T) {
		model := testExternalBGPPeerModel()
		model.LocalIP = types.StringValue("192.0.2.2")
		model.Password = types.StringValue("s3cret")
		model.ImportPolicyUUID = types.StringValue("22222222-2222-2222-2222-222222222222")
		model.ExportPolicyUUID = types.StringValue("33333333-3333-3333-3333-333333333333")
		model.Enabled = types.BoolValue(false)

		m := marshalToMap(t, buildCreateExternalBGPPeerRequest(model))

		want := map[string]any{
			"local_ip":           "192.0.2.2",
			"password":           "s3cret",
			"import_policy_uuid": "22222222-2222-2222-2222-222222222222",
			"export_policy_uuid": "33333333-3333-3333-3333-333333333333",
			"enabled":            false,
		}
		for k, v := range want {
			if m[k] != v {
				t.Errorf("%s = %v, want %v", k, m[k], v)
			}
		}
	})
}

// Removing an optional field from the configuration has to clear it on the
// server, which needs an explicit JSON null — an omitted field means "leave
// unchanged".
func TestBuildUpdateExternalBGPPeerRequest_ClearsUnsetFieldsWithNull(t *testing.T) {
	m := marshalToMap(t, buildUpdateExternalBGPPeerRequest(testExternalBGPPeerModel()))

	for _, k := range []string{"local_ip", "password", "import_policy_uuid", "export_policy_uuid"} {
		v, present := m[k]
		if !present {
			t.Errorf("%s is omitted, want an explicit null so the server clears it", k)
			continue
		}
		if v != nil {
			t.Errorf("%s = %v, want null", k, v)
		}
	}
}

func TestBuildUpdateExternalBGPPeerRequest_SendsSetFields(t *testing.T) {
	model := testExternalBGPPeerModel()
	model.PeerASN = types.Int64Value(64513)
	model.Enabled = types.BoolValue(false)
	model.LocalIP = types.StringValue("192.0.2.2")
	model.Password = types.StringValue("s3cret")
	model.ImportPolicyUUID = types.StringValue("22222222-2222-2222-2222-222222222222")

	m := marshalToMap(t, buildUpdateExternalBGPPeerRequest(model))

	want := map[string]any{
		"name":               "test-peer",
		"peer_ip":            "192.0.2.1",
		"peer_asn":           float64(64513),
		"enabled":            false,
		"local_ip":           "192.0.2.2",
		"password":           "s3cret",
		"import_policy_uuid": "22222222-2222-2222-2222-222222222222",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %v", k, m[k], v)
		}
	}
	if v, present := m["export_policy_uuid"]; !present || v != nil {
		t.Errorf("export_policy_uuid = %v (present=%v), want explicit null", v, present)
	}
	// machine_uuid is not updatable (changing it replaces the resource).
	if _, present := m["machine_uuid"]; present {
		t.Error("machine_uuid must not be sent on update")
	}
}

func testExternalBGPPeerAPIJSON() []byte {
	return []byte(`{
		"uuid": "44444444-4444-4444-4444-444444444444",
		"machine_uuid": "11111111-1111-1111-1111-111111111111",
		"machine_hostname": "gw-01",
		"name": "test-peer",
		"peer_ip": "192.0.2.1",
		"peer_asn": 64512,
		"local_ip": "192.0.2.2",
		"enabled": true,
		"password": "masked",
		"import_policy": {"uuid": "22222222-2222-2222-2222-222222222222", "name": "in", "advanced_mode": false},
		"export_policy": null,
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-02T00:00:00Z",
		"some_future_field": 1
	}`)
}

func assertMappedExternalBGPPeer(t *testing.T, model *ExternalBGPPeerResourceModel) {
	t.Helper()

	checks := map[string][2]string{
		"uuid":               {model.UUID.ValueString(), "44444444-4444-4444-4444-444444444444"},
		"machine_uuid":       {model.MachineUUID.ValueString(), "11111111-1111-1111-1111-111111111111"},
		"machine_hostname":   {model.MachineHostname.ValueString(), "gw-01"},
		"name":               {model.Name.ValueString(), "test-peer"},
		"peer_ip":            {model.PeerIP.ValueString(), "192.0.2.1"},
		"local_ip":           {model.LocalIP.ValueString(), "192.0.2.2"},
		"import_policy_uuid": {model.ImportPolicyUUID.ValueString(), "22222222-2222-2222-2222-222222222222"},
		"import_policy_name": {model.ImportPolicyName.ValueString(), "in"},
		"created_at":         {model.CreatedAt.ValueString(), "2026-01-01T00:00:00Z"},
		"updated_at":         {model.UpdatedAt.ValueString(), "2026-01-02T00:00:00Z"},
	}
	for field, c := range checks {
		if c[0] != c[1] {
			t.Errorf("%s = %q, want %q", field, c[0], c[1])
		}
	}
	if model.PeerASN.ValueInt64() != 64512 {
		t.Errorf("peer_asn = %d, want 64512", model.PeerASN.ValueInt64())
	}
	if !model.Enabled.ValueBool() {
		t.Error("enabled = false, want true")
	}
	if !model.ExportPolicyUUID.IsNull() || !model.ExportPolicyName.IsNull() {
		t.Errorf("export policy = (%v, %v), want null", model.ExportPolicyUUID, model.ExportPolicyName)
	}
}

// The password is never read back from the API: whatever the control plane
// returns (the real value, a mask or an empty string), state keeps the
// configured value, or null when none was configured.
func TestMapExternalBGPPeerToModel(t *testing.T) {
	var peer durantic.ExternalBGPPeerSchema
	if err := json.Unmarshal(testExternalBGPPeerAPIJSON(), &peer); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}

	t.Run("maps fields and preserves the configured password", func(t *testing.T) {
		model := &ExternalBGPPeerResourceModel{Password: types.StringValue("configured")}
		mapExternalBGPPeerToModel(&peer, model)

		assertMappedExternalBGPPeer(t, model)
		if model.Password.ValueString() != "configured" {
			t.Errorf("password = %q, want the configured value kept", model.Password.ValueString())
		}
	})

	t.Run("keeps a null password null", func(t *testing.T) {
		model := &ExternalBGPPeerResourceModel{Password: types.StringNull()}
		mapExternalBGPPeerToModel(&peer, model)

		if !model.Password.IsNull() {
			t.Errorf("password = %v, want null", model.Password)
		}
	})

	t.Run("empty local_ip maps to null", func(t *testing.T) {
		var p durantic.ExternalBGPPeerSchema
		if err := json.Unmarshal([]byte(`{"uuid":"u","machine_uuid":"m","machine_hostname":"h","name":"n","peer_ip":"192.0.2.1","peer_asn":1,"enabled":true,"password":"","local_ip":"","created_at":"c","updated_at":"u"}`), &p); err != nil {
			t.Fatalf("decoding fixture: %v", err)
		}
		model := &ExternalBGPPeerResourceModel{}
		mapExternalBGPPeerToModel(&p, model)

		if !model.LocalIP.IsNull() {
			t.Errorf("local_ip = %v, want null", model.LocalIP)
		}
	})
}

func TestMapRawToExternalBGPPeerModel(t *testing.T) {
	var raw externalBGPPeerRaw
	if err := json.Unmarshal(testExternalBGPPeerAPIJSON(), &raw); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}

	model := &ExternalBGPPeerResourceModel{Password: types.StringValue("configured")}
	mapRawToExternalBGPPeerModel(&raw, model)

	assertMappedExternalBGPPeer(t, model)
	if model.Password.ValueString() != "configured" {
		t.Errorf("password = %q, want the configured value kept", model.Password.ValueString())
	}
}

// TestAccExternalBGPPeerResource needs a machine to terminate the session:
//
//	DURANTIC_TEST_BGP_PEER_MACHINE_UUID  UUID of an existing machine
func TestAccExternalBGPPeerResource(t *testing.T) {
	machineUUID := os.Getenv("DURANTIC_TEST_BGP_PEER_MACHINE_UUID")
	if machineUUID == "" {
		t.Skip("DURANTIC_TEST_BGP_PEER_MACHINE_UUID must be set for this acceptance test")
	}

	const addr = "durantic_external_bgp_peer.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with an import policy attached
			{
				Config: testAccExternalBGPPeerResourceConfig(machineUUID, "test-bgp-peer", 64512, true, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("name"), knownvalue.StringExact("test-bgp-peer")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("peer_asn"), knownvalue.Int64Exact(64512)),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("enabled"), knownvalue.Bool(true)),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("import_policy_name"), knownvalue.StringExact("test-bgp-peer-policy")),
				},
			},
			// Import — the password is never returned by the API
			{
				ResourceName:            addr,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			// Update ASN, disable the session and detach the policy
			{
				Config: testAccExternalBGPPeerResourceConfig(machineUUID, "test-bgp-peer", 64513, false, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("peer_asn"), knownvalue.Int64Exact(64513)),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("enabled"), knownvalue.Bool(false)),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("import_policy_uuid"), knownvalue.Null()),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("import_policy_name"), knownvalue.Null()),
				},
			},
		},
	})
}

func testAccExternalBGPPeerResourceConfig(machineUUID, name string, asn int, enabled, withPolicy bool) string {
	importPolicy := ""
	if withPolicy {
		importPolicy = "import_policy_uuid = durantic_route_policy_set.test.uuid"
	}

	return fmt.Sprintf(`
resource "durantic_route_policy_set" "test" {
  name = "%[2]s-policy"
}

resource "durantic_external_bgp_peer" "test" {
  machine_uuid = %[1]q
  name         = %[2]q
  peer_ip      = "192.0.2.1"
  peer_asn     = %[3]d
  enabled      = %[4]t
  password     = "test-bgp-md5"
  %[5]s
}
`, machineUUID, name, asn, enabled, importPolicy)
}
