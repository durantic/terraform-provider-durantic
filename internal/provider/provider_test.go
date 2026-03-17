// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"durantic": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the durantic provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"durantic": providerserver.NewProtocol6WithError(New("test")()),
	"echo":     echoprovider.NewProviderServer(),
}

func testAccPreCheck(t *testing.T) {
	// Check that DURANTIC_API_TOKEN is set for acceptance tests
	if v := os.Getenv("DURANTIC_API_TOKEN"); v == "" {
		t.Fatal("DURANTIC_API_TOKEN must be set for acceptance tests")
	}

	// DURANTIC_ENDPOINT is optional, log if not set
	if v := os.Getenv("DURANTIC_ENDPOINT"); v == "" {
		t.Log("DURANTIC_ENDPOINT not set, using default")
	}
}
