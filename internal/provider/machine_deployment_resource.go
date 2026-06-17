package provider

import (
	"context"
	"fmt"
	"net/http"
	"time"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const provisionPollInterval = 10 * time.Second
const provisionTimeout = 15 * time.Minute

// provisionDetailFetcher abstracts the API call inside pollProvision so it can be replaced in unit tests.
type provisionDetailFetcher func(ctx context.Context, machineUUID, provisionUUID string) (*durantic.ProvisionDetailSchema, *http.Response, error)

var _ resource.Resource = &MachineDeploymentResource{}
var _ resource.ResourceWithImportState = &MachineDeploymentResource{}

func NewMachineDeploymentResource() resource.Resource {
	return &MachineDeploymentResource{pollInterval: provisionPollInterval}
}

type MachineDeploymentResource struct {
	client       *durantic.APIClient
	pollInterval time.Duration
}

type MachineDeploymentResourceModel struct {
	MachineUUID     types.String `tfsdk:"machine_uuid"`
	ForceProvision  types.String `tfsdk:"force_provision"`
	ProvisionUUID   types.String `tfsdk:"provision_uuid"`
	ProvisionStatus types.String `tfsdk:"provision_status"`
	MachineCommonModel
}

func (r *MachineDeploymentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine_deployment"
}

func (r *MachineDeploymentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages configuration and OS provisioning for an existing Durantic machine. " +
			"On create or replace, applies the desired configuration and triggers a full OS install (`rebuild` mode): " +
			"the machine kexecs into the installer, downloads the OCI image specified by `role_names`, " +
			"writes it to disk, runs cloud-init, and reboots into the installed system. " +
			"The resource blocks until provisioning reaches a terminal state (`completed`, `error`, `timeout`, `canceled`, or `render_failed`). " +
			"Destroying this resource removes it from Terraform state only — the machine is not deleted or re-provisioned.\n\n" +
			"**Provision triggers:**\n" +
			"- `role_names` or `force_provision` change → resource is replaced → new provision run\n" +
			"- `mesh_network_uuid` and other config attributes change → in-place update only, no provision\n" +
			"- `terraform apply -replace` on a specific instance → new provision run for that machine",

		Attributes: map[string]schema.Attribute{
			"machine_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the existing machine to deploy.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"force_provision": schema.StringAttribute{
				MarkdownDescription: "Arbitrary string. Change this value (e.g. `\"v1\"` → `\"v2\"`) to force re-provision all machines in the group without changing their config. For a single machine, use `terraform apply -replace` instead.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provision_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the provision run triggered by this resource. Populated after a successful provision.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"provision_status": schema.StringAttribute{
				MarkdownDescription: "Terminal status of the last provision run: `completed`, `error`, `timeout`, `canceled`, or `render_failed`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the machine.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Machine hostname.",
				Computed:            true,
			},
			"role_names": schema.ListAttribute{
				MarkdownDescription: "List of role names to assign to this machine. Roles define the OS image and cloud-init templates applied during provisioning. **Changing this value triggers a new provision run.**",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"mesh_network_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the mesh network to assign to this machine. Changes are applied without re-provisioning.",
				Optional:            true,
				Computed:            true,
			},
			"tunnel_type": schema.StringAttribute{
				MarkdownDescription: "Tunnel type for this machine (e.g. `auto`, `wireguard`). Changes are applied without re-provisioning.",
				Optional:            true,
				Computed:            true,
			},
			"stun_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether STUN is enabled for this machine. Changes are applied without re-provisioning.",
				Optional:            true,
				Computed:            true,
			},
			"auto_update": schema.BoolAttribute{
				MarkdownDescription: "Whether automatic agent updates are enabled. Changes are applied without re-provisioning.",
				Optional:            true,
				Computed:            true,
			},
			"inject_agent": schema.BoolAttribute{
				MarkdownDescription: "Whether the Durantic agent is injected into the OS image during provisioning. Changes are applied without re-provisioning.",
				Optional:            true,
				Computed:            true,
			},
			"target_disk": schema.StringAttribute{
				MarkdownDescription: "Target disk for OS installation (e.g. `/dev/sda`). Takes effect on the next provision run.",
				Optional:            true,
				Computed:            true,
			},
			"kexec_installer": schema.BoolAttribute{
				MarkdownDescription: "Whether to use kexec for the installer boot (faster, skips BIOS). Disabled automatically for hardware that requires a full reboot (e.g. NVIDIA GPUs).",
				Optional:            true,
				Computed:            true,
			},
			"wg_ip_address": schema.StringAttribute{
				MarkdownDescription: "Mesh (WireGuard) IP address assigned to this machine.",
				Computed:            true,
			},
			"discovered_ip_addresses": schema.ListAttribute{
				MarkdownDescription: "IP addresses discovered for this machine.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"public_ip_addresses": schema.ListAttribute{
				MarkdownDescription: "Public/selectable IP addresses for this machine.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"private_ip_addresses": schema.ListAttribute{
				MarkdownDescription: "Private IP addresses for this machine. Contains the mesh IP when available.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"is_online": schema.BoolAttribute{
				MarkdownDescription: "Whether this machine is currently online.",
				Computed:            true,
			},
			"needs_provisioning": schema.BoolAttribute{
				MarkdownDescription: "Whether this machine has pending config changes that require a provision run to apply.",
				Computed:            true,
			},
			"pending_config_push": schema.BoolAttribute{
				MarkdownDescription: "Whether this machine has a pending config push.",
				Computed:            true,
			},
		},
	}
}

func (r *MachineDeploymentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*durantic.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *durantic.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *MachineDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MachineDeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: apply config
	updated, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiUpdateMachine(ctx, data.MachineUUID.ValueString()).
		MachineUpdateSchema(buildMachineUpdateRequest(ctx, &data.MachineCommonModel, &resp.Diagnostics)).
		Execute()
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		resp.Diagnostics.AddError(
			"Error Updating Machine Config",
			fmt.Sprintf("Could not update machine %s: %s", data.MachineUUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}
	resp.Diagnostics.Append(mapMachineResponseToCommonModel(updated, &data.MachineCommonModel)...)
	data.UUID = data.MachineUUID

	// Step 2: trigger provision (rebuild mode)
	provisionSchema := durantic.NewMachineProvisionSchema()
	provisionSchema.SetMode("rebuild")
	var provisionResp map[string]interface{}
	provisionResp, httpResp, err = r.client.MachinesAPI.
		ProvisioningApiProvisionMachine(ctx, data.MachineUUID.ValueString()).
		MachineProvisionSchema(*provisionSchema).
		Execute()
	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		resp.Diagnostics.AddError(
			"Error Triggering Provision",
			fmt.Sprintf("Could not trigger provision for machine %s: %s", data.MachineUUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}
	provisionUUID, _ := provisionResp["provision_uuid"].(string)

	data.ProvisionUUID = types.StringValue(provisionUUID)
	tflog.Info(ctx, "provision triggered", map[string]any{
		"machine_uuid":   data.MachineUUID.ValueString(),
		"provision_uuid": provisionUUID,
	})

	// Step 3: poll until terminal state
	pollCtx, cancel := context.WithTimeout(ctx, provisionTimeout)
	defer cancel()

	fetcher := func(ctx context.Context, machineUUID, provisionUUID string) (*durantic.ProvisionDetailSchema, *http.Response, error) {
		return r.client.MachinesAPI.ProvisioningApiGetMachineProvision(ctx, machineUUID, provisionUUID).Execute()
	}
	provisionStatus, diags := r.pollProvision(pollCtx, data.MachineUUID.ValueString(), provisionUUID, fetcher)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ProvisionStatus = types.StringValue(provisionStatus)

	// Step 4: refresh machine state after boot
	machine, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiGetMachine(ctx, data.MachineUUID.ValueString()).
		Execute()
	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		resp.Diagnostics.AddError(
			"Error Reading Machine After Provision",
			fmt.Sprintf("Could not read machine %s after provision: %s", data.MachineUUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}
	resp.Diagnostics.Append(mapMachineResponseToCommonModel(machine, &data.MachineCommonModel)...)
	data.UUID = data.MachineUUID

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MachineDeploymentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	machine, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiGetMachine(ctx, data.MachineUUID.ValueString()).
		Execute()
	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Machine Deployment",
			fmt.Sprintf("Could not read machine %s: %s", data.MachineUUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	resp.Diagnostics.Append(mapMachineResponseToCommonModel(machine, &data.MachineCommonModel)...)
	data.UUID = data.MachineUUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MachineDeploymentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only non-RequiresReplace attributes changed — update config, no provision triggered.
	updated, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiUpdateMachine(ctx, data.MachineUUID.ValueString()).
		MachineUpdateSchema(buildMachineUpdateRequest(ctx, &data.MachineCommonModel, &resp.Diagnostics)).
		Execute()
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		resp.Diagnostics.AddError(
			"Error Updating Machine Config",
			fmt.Sprintf("Could not update machine %s: %s", data.MachineUUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	resp.Diagnostics.Append(mapMachineResponseToCommonModel(updated, &data.MachineCommonModel)...)
	data.UUID = data.MachineUUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "removed machine deployment from Terraform state; machine is not deleted or re-provisioned")
}

func (r *MachineDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("machine_uuid"), req, resp)
}

// pollProvision polls the provision until it reaches a terminal state.
// Returns the terminal status string, or an error diagnostic if the provision failed.
func (r *MachineDeploymentResource) pollProvision(ctx context.Context, machineUUID, provisionUUID string, fetch provisionDetailFetcher) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	for {
		select {
		case <-ctx.Done():
			diags.AddError(
				"Provision Timed Out",
				fmt.Sprintf("Provision %s for machine %s did not reach a terminal state within %s.", provisionUUID, machineUUID, provisionTimeout),
			)
			return "", diags
		case <-time.After(r.pollInterval):
		}

		detail, httpResp, err := fetch(ctx, machineUUID, provisionUUID)
		if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
			diags.AddError(
				"Error Polling Provision Status",
				fmt.Sprintf("Could not get provision %s for machine %s: %s", provisionUUID, machineUUID, extractAPIError(httpResp, err)),
			)
			return "", diags
		}

		tflog.Info(ctx, "provision status", map[string]any{
			"machine_uuid":     machineUUID,
			"provision_uuid":   provisionUUID,
			"status":           detail.GetStatus(),
			"progress_percent": detail.GetProgressPercent(),
			"current_step":     detail.GetCurrentStep(),
			"current_message":  detail.GetCurrentMessage(),
		})

		if detail.GetIsTerminal() {
			status := detail.GetStatus()
			if status != "completed" {
				errMsg := detail.GetErrorMessage()
				if errMsg == "" {
					errMsg = fmt.Sprintf("provision ended with status %q", status)
				}
				diags.AddError(
					fmt.Sprintf("Provision Failed (%s)", status),
					fmt.Sprintf("Provision %s for machine %s failed: %s", provisionUUID, machineUUID, errMsg),
				)
				return status, diags
			}
			return status, diags
		}
	}
}
