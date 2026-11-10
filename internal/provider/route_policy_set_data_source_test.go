// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const (
	testPolicySetUUID     = "12121212-1212-1212-1212-121212121212"
	testPolicySetListPath = "/api/connectivity/route-policy-sets/"
)

const testPolicyRuleJSON = `{
	"uuid": "34343434-3434-3434-3434-343434343434",
	"sequence": 10,
	"enabled": true,
	"description": "accept the mesh range",
	"match_type": "prefix",
	"match_prefixes": ["10.0.0.0/8"],
	"match_prefix_len_min": 16,
	"match_prefix_len_max": null,
	"match_communities": [],
	"match_as_path_regex": "",
	"action": "accept",
	"set_local_pref": 200,
	"set_med": null,
	"set_communities": [],
	"add_communities": ["65000:100", "65000:200"],
	"remove_communities": [],
	"prepend_as_count": 0
}`

func testPolicySetJSON(uuid, name, rules string) string {
	return fmt.Sprintf(`{
		"uuid": %q,
		"name": %q,
		"description": "edge policy",
		"default_action": "reject",
		"advanced_mode": true,
		"local_pref": 150,
		"rules": [%s],
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-02T00:00:00Z",
		"some_future_field": "ignored"
	}`, uuid, name, rules)
}

func TestRoutePolicySetDataSourceRead(t *testing.T) {
	client := newFakeControlPlane(t, map[string]string{
		testPolicySetListPath: "[" +
			testPolicySetJSON(testPolicySetUUID, "edge-in", testPolicyRuleJSON) + "," +
			testPolicySetJSON("56565656-5656-5656-5656-565656565656", "dup-set", "") + "," +
			testPolicySetJSON("78787878-7878-7878-7878-787878787878", "dup-set", "") + "]",
		testPolicySetListPath + testPolicySetUUID:                      testPolicySetJSON(testPolicySetUUID, "edge-in", testPolicyRuleJSON),
		testPolicySetListPath + "90909090-9090-9090-9090-909090909090": testPolicySetJSON("90909090-9090-9090-9090-909090909090", "empty-set", ""),
	})

	assertEdgeIn := func(t *testing.T, config map[string]string) {
		t.Helper()
		resp := readTestDataSource(t, NewRoutePolicySetDataSource(), client, config)
		assertNoDiagnostics(t, resp.Diagnostics)

		if got := stateString(t, resp, "uuid"); got != testPolicySetUUID {
			t.Errorf("uuid = %q, want %q", got, testPolicySetUUID)
		}
		if got := stateString(t, resp, "name"); got != "edge-in" {
			t.Errorf("name = %q", got)
		}
		if got := stateString(t, resp, "default_action"); got != "reject" {
			t.Errorf("default_action = %q", got)
		}
		if got := stateInt64(t, resp, "local_pref"); got != 150 {
			t.Errorf("local_pref = %d", got)
		}
		if !stateBool(t, resp, "advanced_mode") {
			t.Error("advanced_mode = false, want true")
		}

		var rules []RoutePolicyRuleModel
		assertNoDiagnostics(t, resp.State.GetAttribute(context.Background(), path.Root("rules"), &rules))
		if len(rules) != 1 {
			t.Fatalf("got %d rules, want 1", len(rules))
		}
		rule := rules[0]
		if rule.Sequence.ValueInt64() != 10 || rule.Action.ValueString() != "accept" || rule.MatchType.ValueString() != "prefix" {
			t.Errorf("rule = sequence %v, action %v, match_type %v", rule.Sequence, rule.Action, rule.MatchType)
		}
		if rule.MatchPrefixLenMin.ValueInt64() != 16 || !rule.MatchPrefixLenMax.IsNull() {
			t.Errorf("prefix length = (%v, %v), want (16, null)", rule.MatchPrefixLenMin, rule.MatchPrefixLenMax)
		}
		if rule.SetLocalPref.ValueInt64() != 200 || !rule.SetMed.IsNull() {
			t.Errorf("set_local_pref/set_med = (%v, %v), want (200, null)", rule.SetLocalPref, rule.SetMed)
		}

		var prefixes, added []string
		assertNoDiagnostics(t, rule.MatchPrefixes.ElementsAs(context.Background(), &prefixes, false))
		assertNoDiagnostics(t, rule.AddCommunities.ElementsAs(context.Background(), &added, false))
		if len(prefixes) != 1 || prefixes[0] != "10.0.0.0/8" {
			t.Errorf("match_prefixes = %v", prefixes)
		}
		if len(added) != 2 || added[0] != "65000:100" || added[1] != "65000:200" {
			t.Errorf("add_communities = %v", added)
		}
	}

	t.Run("by uuid", func(t *testing.T) {
		assertEdgeIn(t, map[string]string{"uuid": testPolicySetUUID})
	})

	t.Run("by name", func(t *testing.T) {
		assertEdgeIn(t, map[string]string{"name": "edge-in"})
	})

	t.Run("a policy set without rules has an empty rule list", func(t *testing.T) {
		resp := readTestDataSource(t, NewRoutePolicySetDataSource(), client, map[string]string{"uuid": "90909090-9090-9090-9090-909090909090"})
		assertNoDiagnostics(t, resp.Diagnostics)

		var rules []RoutePolicyRuleModel
		assertNoDiagnostics(t, resp.State.GetAttribute(context.Background(), path.Root("rules"), &rules))
		if len(rules) != 0 {
			t.Errorf("got %d rules, want none", len(rules))
		}
	})

	t.Run("unknown name", func(t *testing.T) {
		resp := readTestDataSource(t, NewRoutePolicySetDataSource(), client, map[string]string{"name": "missing"})
		assertErrorDiagnostic(t, resp.Diagnostics, "No Route Policy Set Found")
	})

	t.Run("ambiguous name", func(t *testing.T) {
		resp := readTestDataSource(t, NewRoutePolicySetDataSource(), client, map[string]string{"name": "dup-set"})
		assertErrorDiagnostic(t, resp.Diagnostics, "Multiple Route Policy Sets Found")
	})

	t.Run("unknown uuid", func(t *testing.T) {
		resp := readTestDataSource(t, NewRoutePolicySetDataSource(), client, map[string]string{"uuid": "does-not-exist"})
		assertErrorDiagnostic(t, resp.Diagnostics, "Error Reading Route Policy Set")
	})

	t.Run("no selector", func(t *testing.T) {
		resp := readTestDataSource(t, NewRoutePolicySetDataSource(), client, map[string]string{})
		assertErrorDiagnostic(t, resp.Diagnostics, "Invalid Route Policy Set Lookup")
	})
}

func TestAccRoutePolicySetDataSource_NotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "durantic_route_policy_set" "missing" {
  name = "test-policy-that-does-not-exist"
}
`,
				ExpectError: regexp.MustCompile("No Route Policy Set Found"),
			},
		},
	})
}

// The policy set is created in the same configuration, so no pre-existing
// infrastructure is needed.
func TestAccRoutePolicySetDataSource_ByNameAndUUID(t *testing.T) {
	const config = `
resource "durantic_route_policy_set" "test" {
  name = "test-policy-lookup"
}

data "durantic_route_policy_set" "by_name" {
  name = durantic_route_policy_set.test.name
}

data "durantic_route_policy_set" "by_uuid" {
  uuid = durantic_route_policy_set.test.uuid
}
`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.CompareValuePairs("data.durantic_route_policy_set.by_name", tfjsonpath.New("uuid"), "durantic_route_policy_set.test", tfjsonpath.New("uuid"), compare.ValuesSame()),
					statecheck.CompareValuePairs("data.durantic_route_policy_set.by_uuid", tfjsonpath.New("name"), "durantic_route_policy_set.test", tfjsonpath.New("name"), compare.ValuesSame()),
					statecheck.ExpectKnownValue("data.durantic_route_policy_set.by_name", tfjsonpath.New("rules"), knownvalue.ListSizeExact(0)),
				},
			},
		},
	})
}
