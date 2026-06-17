package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"durantic": providerserver.NewProtocol6WithError(New("test")()),
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
