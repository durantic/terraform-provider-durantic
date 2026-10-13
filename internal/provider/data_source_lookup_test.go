// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// newFakeControlPlane starts an HTTP server that answers GETs for the given
// paths with the given JSON bodies (404 for anything else) and returns a client
// pointed at it. This lets data source Read be exercised without a live API.
func newFakeControlPlane(t *testing.T, routes map[string]string) *durantic.APIClient {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, ok := routes[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"detail":"not found"}`)
			return
		}
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parsing fake control plane URL: %v", err)
	}

	cfg := durantic.NewConfiguration()
	cfg.Scheme = u.Scheme
	cfg.Host = u.Host
	return durantic.NewAPIClient(cfg)
}

// readTestDataSource configures ds with client and runs Read with the given
// string attributes set in the configuration (everything else null). The
// response carries the resulting state and any diagnostics.
func readTestDataSource(t *testing.T, ds datasource.DataSource, client *durantic.APIClient, config map[string]string) *datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()

	var schemaResp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", schemaResp.Diagnostics)
	}
	if diags := schemaResp.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("invalid schema: %v", diags)
	}

	configurable, ok := ds.(datasource.DataSourceWithConfigure)
	if !ok {
		t.Fatalf("%T does not implement DataSourceWithConfigure", ds)
	}
	var cfgResp datasource.ConfigureResponse
	configurable.Configure(ctx, datasource.ConfigureRequest{ProviderData: client}, &cfgResp)
	if cfgResp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %v", cfgResp.Diagnostics)
	}

	objType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not an object")
	}
	values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	for name, v := range config {
		values[name] = tftypes.NewValue(tftypes.String, v)
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)},
	}
	ds.Read(ctx, datasource.ReadRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)},
	}, resp)
	return resp
}

func assertNoDiagnostics(t *testing.T, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %v", diags)
	}
}

func assertErrorDiagnostic(t *testing.T, diags diag.Diagnostics, summary string) {
	t.Helper()
	for _, d := range diags.Errors() {
		if strings.Contains(d.Summary(), summary) {
			return
		}
	}
	t.Fatalf("expected an error with summary containing %q, got: %v", summary, diags)
}

func stateString(t *testing.T, resp *datasource.ReadResponse, attr string) string {
	t.Helper()
	var v types.String
	assertNoDiagnostics(t, resp.State.GetAttribute(context.Background(), path.Root(attr), &v))
	return v.ValueString()
}

func stateInt64(t *testing.T, resp *datasource.ReadResponse, attr string) int64 {
	t.Helper()
	var v types.Int64
	assertNoDiagnostics(t, resp.State.GetAttribute(context.Background(), path.Root(attr), &v))
	return v.ValueInt64()
}

func stateBool(t *testing.T, resp *datasource.ReadResponse, attr string) bool {
	t.Helper()
	var v types.Bool
	assertNoDiagnostics(t, resp.State.GetAttribute(context.Background(), path.Root(attr), &v))
	return v.ValueBool()
}

func TestRequireUUIDOrName(t *testing.T) {
	cases := []struct {
		name     string
		uuid, nm types.String
		wantOK   bool
	}{
		{"uuid only", types.StringValue("u"), types.StringNull(), true},
		{"name only", types.StringNull(), types.StringValue("n"), true},
		{"both", types.StringValue("u"), types.StringValue("n"), false},
		{"neither", types.StringNull(), types.StringNull(), false},
		{"empty strings count as unset", types.StringValue(""), types.StringValue(""), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := requireUUIDOrName("VIP", c.uuid, c.nm, &diags)
			if got != c.wantOK {
				t.Errorf("requireUUIDOrName = %v, want %v", got, c.wantOK)
			}
			if got == diags.HasError() {
				t.Errorf("diagnostics error = %v, want %v", diags.HasError(), !got)
			}
			if !got {
				assertErrorDiagnostic(t, diags, "Invalid VIP Lookup")
			}
		})
	}
}

func TestFindUniqueByName(t *testing.T) {
	type item struct{ name, id string }
	nameOf := func(i item) string { return i.name }

	t.Run("single match", func(t *testing.T) {
		var diags diag.Diagnostics
		got, ok := findUniqueByName("Thing", "b", []item{{"a", "1"}, {"b", "2"}}, nameOf, &diags)
		if !ok || got.id != "2" {
			t.Errorf("got (%v, %v), want id 2", got, ok)
		}
		assertNoDiagnostics(t, diags)
	})

	t.Run("no match", func(t *testing.T) {
		var diags diag.Diagnostics
		if _, ok := findUniqueByName("Thing", "z", []item{{"a", "1"}}, nameOf, &diags); ok {
			t.Error("ok = true, want false")
		}
		assertErrorDiagnostic(t, diags, "No Thing Found")
	})

	t.Run("ambiguous match", func(t *testing.T) {
		var diags diag.Diagnostics
		if _, ok := findUniqueByName("Thing", "a", []item{{"a", "1"}, {"a", "2"}}, nameOf, &diags); ok {
			t.Error("ok = true, want false")
		}
		assertErrorDiagnostic(t, diags, "Multiple Things Found")
	})

	t.Run("names are matched exactly", func(t *testing.T) {
		var diags diag.Diagnostics
		if _, ok := findUniqueByName("Thing", "a", []item{{"ab", "1"}, {"A", "2"}}, nameOf, &diags); ok {
			t.Error("ok = true, want false")
		}
	})
}
