package provider

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	durantic "github.com/durantic/controlplane-client-go/durantic"
)

// TestMain runs a cleanup sweep before the acceptance suite so no leftover test
// resource from a previous (or crashed) run can collide with this run — e.g. a stray
// mesh network still holding the globally-unique network_cidr a test wants to create.
//
// The sweep only runs for the live-account acceptance tier (TF_ACC=1 with an API
// token); the unit tier (plain `go test`, no TF_ACC) skips it and makes no network
// calls. Each `go test` invocation — including each Terraform-version matrix leg —
// sweeps before its own tests, so a leg cannot inherit another leg's leftovers.
func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") != "" && os.Getenv(apiTokenEnvName) != "" {
		testAccSweepLeftovers()
	}
	os.Exit(m.Run())
}

// testAccResourceNamePrefix is the prefix every acceptance-test resource name uses.
// The sweep deletes only resources whose name starts with it, leaving any baseline
// resources in the (test-only) account untouched.
const testAccResourceNamePrefix = "test-"

func isTestResourceName(name string) bool {
	return strings.HasPrefix(name, testAccResourceNamePrefix)
}

// testAccAPIClientFromEnv builds a controlplane client from the same environment
// variables the provider reads (DURANTIC_ENDPOINT / DURANTIC_API_TOKEN /
// DURANTIC_INSECURE_SKIP_VERIFY).
func testAccAPIClientFromEnv() *durantic.APIClient {
	endpoint := stringCoalesce(os.Getenv(endpointEnvName), defaultAPIEndpoint)
	parsed, _ := url.Parse(endpoint)

	cfg := durantic.NewConfiguration()
	cfg.Scheme = parsed.Scheme
	cfg.Host = parsed.Host
	cfg.DefaultHeader["Authorization"] = "Bearer " + os.Getenv(apiTokenEnvName)

	if v, err := strconv.ParseBool(os.Getenv(insecureSkipVerifyEnvName)); err == nil && v {
		cfg.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test-only opt-in, mirrors provider config
			},
		}
	}

	return durantic.NewAPIClient(cfg)
}

// testNamedResource is satisfied by every list-item schema the sweep handles.
type testNamedResource interface {
	GetUuid() string
	GetName() string
}

// testAccSweep lists resources of one type and deletes those left over from a prior
// test run (name starting with the test prefix). List and delete failures are logged
// and skipped — the sweep is best-effort and must not mask the actual test results.
func testAccSweep[T any, PT interface {
	*T
	testNamedResource
}](
	label string,
	list func() ([]T, *http.Response, error),
	del func(uuid string) (*http.Response, error),
) {
	items, _, err := list()
	if err != nil {
		log.Printf("acc sweep %s: list failed, skipping: %v", label, err)
		return
	}

	deleted := 0
	for i := range items {
		item := PT(&items[i])
		if !isTestResourceName(item.GetName()) {
			continue
		}
		if _, err := del(item.GetUuid()); err != nil {
			log.Printf("acc sweep %s: deleting %q (%s) failed: %v", label, item.GetName(), item.GetUuid(), err)
			continue
		}
		deleted++
	}
	if deleted > 0 {
		log.Printf("acc sweep %s: removed %d leftover resource(s)", label, deleted)
	}
}

// testAccSweepLeftovers removes leftover acceptance-test resources from the account
// before the suite runs, across every type the resource tests create.
func testAccSweepLeftovers() {
	client := testAccAPIClientFromEnv()
	ctx := context.Background()

	testAccSweep[durantic.MeshNetworkSchema]("mesh_network",
		func() ([]durantic.MeshNetworkSchema, *http.Response, error) {
			return client.MeshNetworksAPI.ControlplaneApiListMeshNetworks(ctx).Execute()
		},
		func(uuid string) (*http.Response, error) {
			return client.MeshNetworksAPI.ControlplaneApiDeleteMeshNetwork(ctx, uuid).Execute()
		},
	)

	testAccSweep[durantic.MachineRoleSchema]("machine_role",
		func() ([]durantic.MachineRoleSchema, *http.Response, error) {
			return client.MachineRolesAPI.ControlplaneApiListMachineRoles(ctx).Execute()
		},
		func(uuid string) (*http.Response, error) {
			return client.MachineRolesAPI.ControlplaneApiDeleteMachineRole(ctx, uuid).Execute()
		},
	)

	testAccSweep[durantic.AccountSecretSchema]("secret",
		func() ([]durantic.AccountSecretSchema, *http.Response, error) {
			return client.SecretsAPI.ControlplaneApiListAccountSecrets(ctx).Execute()
		},
		func(uuid string) (*http.Response, error) {
			return client.SecretsAPI.ControlplaneApiDeleteAccountSecret(ctx, uuid).Execute()
		},
	)

	testAccSweep[durantic.AccountVariableSchema]("variable",
		func() ([]durantic.AccountVariableSchema, *http.Response, error) {
			return client.VariablesAPI.ControlplaneApiListAccountVariables(ctx).Execute()
		},
		func(uuid string) (*http.Response, error) {
			return client.VariablesAPI.ControlplaneApiDeleteAccountVariable(ctx, uuid).Execute()
		},
	)

	testAccSweep[durantic.RegistryCredentialSchema]("registry_credential",
		func() ([]durantic.RegistryCredentialSchema, *http.Response, error) {
			return client.RegistryCredentialsAPI.ProvisioningApiListRegistryCredentials(ctx).Execute()
		},
		func(uuid string) (*http.Response, error) {
			return client.RegistryCredentialsAPI.ProvisioningApiDeleteRegistryCredential(ctx, uuid).Execute()
		},
	)

	testAccSweep[durantic.SecretsBackendSchema]("secrets_backend",
		func() ([]durantic.SecretsBackendSchema, *http.Response, error) {
			return client.SecretsBackendsAPI.ControlplaneApiListSecretsBackends(ctx).Execute()
		},
		func(uuid string) (*http.Response, error) {
			return client.SecretsBackendsAPI.ControlplaneApiDeleteSecretsBackend(ctx, uuid).Execute()
		},
	)

	testAccSweep[durantic.RoutePolicySetSchema]("route_policy_set",
		func() ([]durantic.RoutePolicySetSchema, *http.Response, error) {
			return client.RoutePolicySetsAPI.ControlplaneApiListRoutePolicySets(ctx).Execute()
		},
		func(uuid string) (*http.Response, error) {
			return client.RoutePolicySetsAPI.ControlplaneApiDeleteRoutePolicySet(ctx, uuid).Execute()
		},
	)
}
