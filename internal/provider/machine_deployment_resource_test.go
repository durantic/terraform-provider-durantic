// # Test tiers for durantic_machine_deployment
//
// Unit tests (TestPollProvision_*):
//   - No real infrastructure required.
//   - Run automatically in CI on every push via `make test`.
//   - Test the polling logic in isolation using injected mock fetchers.
//
// Acceptance tests (TestAccMachineDeploymentResource):
//   - Require a live Durantic environment (real machines, mesh network, roles).
//   - Will physically boot a machine into the installer and wait ~15 minutes.
//   - Run manually or in a dedicated test environment via TF_ACC=1 make testacc.
//   - Required env vars: DURANTIC_TEST_MACHINE_DEPLOYMENT_UUID,
//     DURANTIC_TEST_MACHINE_DEPLOYMENT_MESH_NETWORK_UUID,
//     DURANTIC_TEST_MACHINE_DEPLOYMENT_ROLE_NAMES.
//   - Optional: DURANTIC_TEST_MACHINE_DEPLOYMENT_MESH_NETWORK_UUID2 (for update test).
//
// This two-tier approach mirrors the e2e/ and agent/ test suites in this repo.

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// --- Unit tests for pollProvision (no infra required, run in CI) ---

func newTestDeploymentResource() *MachineDeploymentResource {
	return &MachineDeploymentResource{pollInterval: time.Millisecond}
}

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

func makeDetail(status string, isTerminal bool, errMsg string) *durantic.ProvisionDetailSchema {
	d := durantic.NewProvisionDetailSchemaWithDefaults()
	d.Status = status
	d.IsTerminal = boolPtr(isTerminal)
	d.IsActive = boolPtr(!isTerminal)
	d.CurrentStep = status
	d.CurrentMessage = status
	if errMsg != "" {
		d.ErrorMessage = strPtr(errMsg)
	}
	return d
}

func TestPollProvision_Completed(t *testing.T) {
	r := newTestDeploymentResource()
	calls := 0
	fetch := func(_ context.Context, _, _ string) (*durantic.ProvisionDetailSchema, *http.Response, error) {
		calls++
		if calls < 3 {
			return makeDetail("downloading", false, ""), nil, nil
		}
		return makeDetail("completed", true, ""), nil, nil
	}

	status, diags := r.pollProvision(context.Background(), "machine-1", "provision-1", fetch)
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if status != "completed" {
		t.Errorf("expected completed, got %q", status)
	}
	if calls != 3 {
		t.Errorf("expected 3 poll calls, got %d", calls)
	}
}

func TestPollProvision_ErrorStatus(t *testing.T) {
	r := newTestDeploymentResource()
	fetch := func(_ context.Context, _, _ string) (*durantic.ProvisionDetailSchema, *http.Response, error) {
		return makeDetail("error", true, "disk write failed"), nil, nil
	}

	status, diags := r.pollProvision(context.Background(), "machine-1", "provision-1", fetch)
	if !diags.HasError() {
		t.Fatal("expected error diagnostic")
	}
	if status != "error" {
		t.Errorf("expected error status, got %q", status)
	}
	if diags[0].Detail() != "Provision provision-1 for machine machine-1 failed: disk write failed" {
		t.Errorf("unexpected error detail: %q", diags[0].Detail())
	}
}

func TestPollProvision_RenderFailed(t *testing.T) {
	r := newTestDeploymentResource()
	fetch := func(_ context.Context, _, _ string) (*durantic.ProvisionDetailSchema, *http.Response, error) {
		return makeDetail("render_failed", true, "template syntax error"), nil, nil
	}

	_, diags := r.pollProvision(context.Background(), "machine-1", "provision-1", fetch)
	if !diags.HasError() {
		t.Fatal("expected error diagnostic")
	}
	detail := diags[0].Detail()
	if !strings.Contains(detail, "template syntax error") {
		t.Errorf("error detail should contain error message, got: %q", detail)
	}
}

func TestPollProvision_ContextCanceled(t *testing.T) {
	r := newTestDeploymentResource()
	fetch := func(_ context.Context, _, _ string) (*durantic.ProvisionDetailSchema, *http.Response, error) {
		return makeDetail("downloading", false, ""), nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, diags := r.pollProvision(ctx, "machine-1", "provision-1", fetch)
	if !diags.HasError() {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(diags[0].Summary(), "Timed Out") {
		t.Errorf("expected timeout summary, got: %q", diags[0].Summary())
	}
}

func TestPollProvision_APIError(t *testing.T) {
	r := newTestDeploymentResource()
	fetch := func(_ context.Context, _, _ string) (*durantic.ProvisionDetailSchema, *http.Response, error) {
		return nil, nil, fmt.Errorf("connection refused")
	}

	_, diags := r.pollProvision(context.Background(), "machine-1", "provision-1", fetch)
	if !diags.HasError() {
		t.Fatal("expected error diagnostic")
	}
	if !strings.Contains(diags[0].Summary(), "Error Polling") {
		t.Errorf("unexpected summary: %q", diags[0].Summary())
	}
}

func TestPollProvision_ErrorStatusNoMessage(t *testing.T) {
	r := newTestDeploymentResource()
	fetch := func(_ context.Context, _, _ string) (*durantic.ProvisionDetailSchema, *http.Response, error) {
		return makeDetail("timeout", true, ""), nil, nil
	}

	_, diags := r.pollProvision(context.Background(), "machine-1", "provision-1", fetch)
	if !diags.HasError() {
		t.Fatal("expected error diagnostic")
	}
	// Falls back to generic message when ErrorMessage is empty
	if !strings.Contains(diags[0].Detail(), `ended with status "timeout"`) {
		t.Errorf("unexpected error detail: %q", diags[0].Detail())
	}
}

// --- Acceptance tests (require real infrastructure) ---

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
