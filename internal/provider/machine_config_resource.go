// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
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

var _ resource.Resource = &MachineConfigResource{}
var _ resource.ResourceWithImportState = &MachineConfigResource{}

func NewMachineConfigResource() resource.Resource {
	return &MachineConfigResource{}
}

type MachineConfigResource struct {
	client *durantic.APIClient
}

type MachineConfigResourceModel struct {
	MachineUUID types.String `tfsdk:"machine_uuid"`
	MachineCommonModel
}

func (r *MachineConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine_config"
}

func (r *MachineConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages desired configuration for an existing Durantic machine. Destroying this resource does not delete the machine.",

		Attributes: map[string]schema.Attribute{
			"machine_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the existing machine to configure.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
				MarkdownDescription: "Complete list of role names assigned to this machine.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
			"mesh_network_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the mesh network assigned to this machine.",
				Optional:            true,
				Computed:            true,
			},
			"tunnel_type": schema.StringAttribute{
				MarkdownDescription: "Tunnel type configured for this machine.",
				Optional:            true,
				Computed:            true,
			},
			"stun_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether STUN is enabled for this machine.",
				Optional:            true,
				Computed:            true,
			},
			"auto_update": schema.BoolAttribute{
				MarkdownDescription: "Whether automatic updates are enabled for this machine.",
				Optional:            true,
				Computed:            true,
			},
			"inject_agent": schema.BoolAttribute{
				MarkdownDescription: "Whether agent injection is enabled for this machine.",
				Optional:            true,
				Computed:            true,
			},
			"target_disk": schema.StringAttribute{
				MarkdownDescription: "Target disk configured for this machine.",
				Optional:            true,
				Computed:            true,
			},
			"kexec_installer": schema.BoolAttribute{
				MarkdownDescription: "Whether kexec installer is enabled for this machine.",
				Optional:            true,
				Computed:            true,
			},
			"wg_ip_address": schema.StringAttribute{
				MarkdownDescription: "Mesh IP address assigned to this machine.",
				Computed:            true,
			},
			"discovered_ip_addresses": schema.ListAttribute{
				MarkdownDescription: "IP addresses discovered for this machine.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"public_ip_addresses": schema.ListAttribute{
				MarkdownDescription: "Public/selectable IP addresses for this machine. Mirrors discovered IP addresses used by the Cluster Wizard.",
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
				MarkdownDescription: "Whether this machine needs provisioning.",
				Computed:            true,
			},
			"pending_config_push": schema.BoolAttribute{
				MarkdownDescription: "Whether this machine has a pending config push.",
				Computed:            true,
			},
		},
	}
}

func (r *MachineConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MachineConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MachineConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiUpdateMachine(ctx, data.MachineUUID.ValueString()).
		MachineUpdateSchema(buildMachineUpdateRequest(ctx, &data, &resp.Diagnostics)).
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
	data.MachineUUID = data.UUID

	tflog.Trace(ctx, "updated machine config")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MachineConfigResourceModel

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
			"Error Reading Machine Config",
			fmt.Sprintf("Could not read machine %s: %s", data.MachineUUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	resp.Diagnostics.Append(mapMachineResponseToCommonModel(machine, &data.MachineCommonModel)...)
	data.MachineUUID = data.UUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MachineConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiUpdateMachine(ctx, data.MachineUUID.ValueString()).
		MachineUpdateSchema(buildMachineUpdateRequest(ctx, &data, &resp.Diagnostics)).
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
	data.MachineUUID = data.UUID
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "removed machine config from Terraform state without deleting or reprovisioning the machine")
}

func (r *MachineConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("machine_uuid"), req, resp)
}

func buildMachineUpdateRequest(ctx context.Context, data *MachineConfigResourceModel, diags *diag.Diagnostics) durantic.MachineUpdateSchema {
	updateReq := durantic.NewMachineUpdateSchema()

	if isKnownString(data.MeshNetworkUUID) {
		updateReq.SetMeshNetworkUuid(data.MeshNetworkUUID.ValueString())
	}

	if isKnownString(data.TunnelType) {
		updateReq.SetTunnelType(durantic.TunnelType(data.TunnelType.ValueString()))
	}

	if !data.StunEnabled.IsNull() && !data.StunEnabled.IsUnknown() {
		updateReq.SetStunEnabled(data.StunEnabled.ValueBool())
	}

	if !data.AutoUpdate.IsNull() && !data.AutoUpdate.IsUnknown() {
		updateReq.SetAutoUpdate(data.AutoUpdate.ValueBool())
	}

	if !data.InjectAgent.IsNull() && !data.InjectAgent.IsUnknown() {
		updateReq.SetInjectAgent(data.InjectAgent.ValueBool())
	}

	if isKnownString(data.TargetDisk) {
		updateReq.SetTargetDisk(data.TargetDisk.ValueString())
	}

	if !data.KexecInstaller.IsNull() && !data.KexecInstaller.IsUnknown() {
		updateReq.SetKexecInstaller(data.KexecInstaller.ValueBool())
	}

	if !data.RoleNames.IsNull() && !data.RoleNames.IsUnknown() {
		var roleNames []string
		diags.Append(data.RoleNames.ElementsAs(ctx, &roleNames, false)...)
		updateReq.SetRoleNames(roleNames)
	}

	return *updateReq
}
