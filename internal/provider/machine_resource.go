// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MachineResource{}
var _ resource.ResourceWithImportState = &MachineResource{}

func NewMachineResource() resource.Resource {
	return &MachineResource{}
}

// MachineResource defines the resource implementation.
type MachineResource struct {
	client *durantic.APIClient
}

// MachineResourceModel describes the resource data model.
type MachineResourceModel struct {
	// Identifier (Computed on import)
	UUID types.String `tfsdk:"uuid"`

	// Configurable fields (Optional - can be updated)
	MeshNetworkUuid    types.String `tfsdk:"mesh_network_uuid"`
	AdvertisedRoutes   types.List   `tfsdk:"advertised_routes"`
	DockerRegistryAuth types.String `tfsdk:"docker_registry_auth"`
	RoleNames          types.List   `tfsdk:"role_names"`
	TargetDisk         types.String `tfsdk:"target_disk"`

	// Read-only fields (Computed)
	Hostname          types.String `tfsdk:"hostname"`
	SystemUuid        types.String `tfsdk:"system_uuid"`
	NeedsProvisioning types.Bool   `tfsdk:"needs_provisioning"`
	WgIpAddress       types.String `tfsdk:"wg_ip_address"`
	WgPubkey          types.String `tfsdk:"wg_pubkey"`
	IsOnline          types.Bool   `tfsdk:"is_online"`
	IsInInitrd        types.Bool   `tfsdk:"is_in_initrd"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (r *MachineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (r *MachineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a machine's configuration. Machines are auto-discovered and cannot be created or deleted via Terraform. Use `terraform import` to manage existing machines.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the machine",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mesh_network_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the mesh network to assign the machine to",
				Optional:            true,
			},
			"advertised_routes": schema.ListAttribute{
				MarkdownDescription: "Routes advertised by the machine",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"docker_registry_auth": schema.StringAttribute{
				MarkdownDescription: "Docker registry authentication configuration (JSON string)",
				Optional:            true,
				Sensitive:           true,
			},
			"role_names": schema.ListAttribute{
				MarkdownDescription: "List of role names to assign to the machine",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"target_disk": schema.StringAttribute{
				MarkdownDescription: "Target disk for provisioning (e.g., /dev/sda)",
				Optional:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Machine hostname",
				Computed:            true,
			},
			"system_uuid": schema.StringAttribute{
				MarkdownDescription: "Hardware system UUID",
				Computed:            true,
			},
			"needs_provisioning": schema.BoolAttribute{
				MarkdownDescription: "Whether the machine needs provisioning",
				Computed:            true,
			},
			"wg_ip_address": schema.StringAttribute{
				MarkdownDescription: "WireGuard IP address",
				Computed:            true,
			},
			"wg_pubkey": schema.StringAttribute{
				MarkdownDescription: "WireGuard public key",
				Computed:            true,
			},
			"is_online": schema.BoolAttribute{
				MarkdownDescription: "Whether the machine is currently online",
				Computed:            true,
			},
			"is_in_initrd": schema.BoolAttribute{
				MarkdownDescription: "Whether the machine is in installer initrd",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the machine was created",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the machine was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *MachineResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *MachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Machines cannot be created - they are auto-discovered
	resp.Diagnostics.AddError(
		"Cannot Create Machine",
		"Machines are auto-discovered and cannot be created via Terraform. "+
			"Use 'terraform import durantic_machine.<name> <machine_uuid>' to manage existing machines.",
	)
}

func (r *MachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MachineResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to get machine
	machine, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiGetMachine(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		// Handle 404 - resource deleted outside Terraform
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Machine",
			fmt.Sprintf("Could not read machine %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Map response to state
	mapMachineToResourceModel(ctx, machine, &data, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MachineResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request
	updateReq := durantic.NewMachineUpdateSchema()

	// Set mesh_network_uuid (nullable)
	if !data.MeshNetworkUuid.IsNull() {
		updateReq.SetMeshNetworkUuid(data.MeshNetworkUuid.ValueString())
	}

	// Set advertised_routes
	if !data.AdvertisedRoutes.IsNull() {
		var routes []string
		diags := data.AdvertisedRoutes.ElementsAs(ctx, &routes, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			updateReq.SetAdvertisedRoutes(routes)
		}
	}

	// Set docker_registry_auth (nullable string)
	if !data.DockerRegistryAuth.IsNull() {
		updateReq.SetDockerRegistryAuth(data.DockerRegistryAuth.ValueString())
	}

	// Set role_names
	if !data.RoleNames.IsNull() {
		var roles []string
		diags := data.RoleNames.ElementsAs(ctx, &roles, false)
		resp.Diagnostics.Append(diags...)
		if !resp.Diagnostics.HasError() {
			updateReq.SetRoleNames(roles)
		}
	}

	// Set target_disk (nullable)
	if !data.TargetDisk.IsNull() {
		updateReq.SetTargetDisk(data.TargetDisk.ValueString())
	}

	// Call API to update machine
	machine, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiUpdateMachine(ctx, data.UUID.ValueString()).
		MachineUpdateSchema(*updateReq).
		Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Machine",
			fmt.Sprintf("Could not update machine %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Map response to state
	mapMachineToResourceModel(ctx, machine, &data, &resp.Diagnostics)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MachineResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Machines cannot be deleted - just remove from state
	// The API does not provide a delete endpoint for machines
	tflog.Trace(ctx, fmt.Sprintf("machine %s removed from Terraform state (machines cannot be deleted via API)", data.UUID.ValueString()))
}

func (r *MachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// mapMachineToResourceModel maps API MachineResponseSchema to Terraform model.
func mapMachineToResourceModel(ctx context.Context, machine *durantic.MachineResponseSchema, model *MachineResourceModel, diagnostics *diag.Diagnostics) {
	// Set computed fields from API response
	model.Hostname = types.StringValue(machine.GetHostname())
	model.NeedsProvisioning = types.BoolValue(machine.GetNeedsProvisioning())
	model.CreatedAt = types.StringValue(machine.GetCreatedAt())
	model.UpdatedAt = types.StringValue(machine.GetUpdatedAt())
	model.SystemUuid = stringFromNullable(machine.SystemUuid)
	model.WgIpAddress = stringFromNullable(machine.WgIpAddress)
	model.WgPubkey = stringFromNullable(machine.WgPubkey)
	model.IsOnline = boolFromPointer(machine.IsOnline)
	model.IsInInitrd = boolFromPointer(machine.IsInInitrd)

	// Only update configurable fields if they are not set in the plan.
	// This preserves user-provided values from the configuration.

	// Handle mesh_network_uuid from API if not set in model
	if model.MeshNetworkUuid.IsNull() {
		if machine.MeshNetwork.IsSet() {
			network := machine.GetMeshNetwork()
			model.MeshNetworkUuid = types.StringValue(network.GetUuid())
		} else {
			model.MeshNetworkUuid = types.StringNull()
		}
	}

	if model.AdvertisedRoutes.IsNull() {
		model.AdvertisedRoutes = listFromSlice(ctx, diagnostics, machine.GetAdvertisedRoutes())
	}

	if model.DockerRegistryAuth.IsNull() {
		if len(machine.GetDockerRegistryAuth()) > 0 {
			authJSON, err := json.Marshal(machine.GetDockerRegistryAuth())
			if err != nil {
				diagnostics.AddError("Failed to serialize docker_registry_auth",
					fmt.Sprintf("Could not marshal docker registry auth to JSON: %s", err))
				model.DockerRegistryAuth = types.StringNull()
			} else {
				model.DockerRegistryAuth = types.StringValue(string(authJSON))
			}
		} else {
			model.DockerRegistryAuth = types.StringNull()
		}
	}

	if model.RoleNames.IsNull() {
		model.RoleNames = listFromSlice(ctx, diagnostics, machine.GetRoleNames())
	}

	if model.TargetDisk.IsNull() {
		model.TargetDisk = stringFromNullable(machine.TargetDisk)
	}
}
