// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &MachineDataSource{}

func NewMachineDataSource() datasource.DataSource {
	return &MachineDataSource{}
}

type MachineDataSource struct {
	client *durantic.APIClient
}

type MachineCommonModel struct {
	UUID                  types.String `tfsdk:"uuid"`
	Hostname              types.String `tfsdk:"hostname"`
	RoleNames             types.List   `tfsdk:"role_names"`
	MeshNetworkUUID       types.String `tfsdk:"mesh_network_uuid"`
	WgIPAddress           types.String `tfsdk:"wg_ip_address"`
	DiscoveredIPAddresses types.List   `tfsdk:"discovered_ip_addresses"`
	PublicIPAddresses     types.List   `tfsdk:"public_ip_addresses"`
	PrivateIPAddresses    types.List   `tfsdk:"private_ip_addresses"`
	IsOnline              types.Bool   `tfsdk:"is_online"`
	NeedsProvisioning     types.Bool   `tfsdk:"needs_provisioning"`
	PendingConfigPush     types.Bool   `tfsdk:"pending_config_push"`
	TunnelType            types.String `tfsdk:"tunnel_type"`
	StunEnabled           types.Bool   `tfsdk:"stun_enabled"`
	AutoUpdate            types.Bool   `tfsdk:"auto_update"`
	InjectAgent           types.Bool   `tfsdk:"inject_agent"`
	TargetDisk            types.String `tfsdk:"target_disk"`
	KexecInstaller        types.Bool   `tfsdk:"kexec_installer"`
}

type MachineDataSourceModel struct {
	MachineCommonModel
	NotFoundOk types.Bool `tfsdk:"not_found_ok"`
}

func (d *MachineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (d *MachineDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Durantic machine by UUID or hostname.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the machine. Set either `uuid` or `hostname`.",
				Optional:            true,
				Computed:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Machine hostname. Set either `uuid` or `hostname`.",
				Optional:            true,
				Computed:            true,
			},
			"role_names": schema.ListAttribute{
				MarkdownDescription: "Machine role names currently assigned to this machine.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"mesh_network_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the mesh network assigned to this machine.",
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
			"tunnel_type": schema.StringAttribute{
				MarkdownDescription: "Configured tunnel type.",
				Computed:            true,
			},
			"stun_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether STUN is enabled for this machine.",
				Computed:            true,
			},
			"auto_update": schema.BoolAttribute{
				MarkdownDescription: "Whether automatic updates are enabled for this machine.",
				Computed:            true,
			},
			"inject_agent": schema.BoolAttribute{
				MarkdownDescription: "Whether agent injection is enabled for this machine.",
				Computed:            true,
			},
			"target_disk": schema.StringAttribute{
				MarkdownDescription: "Target disk configured for this machine.",
				Computed:            true,
			},
			"kexec_installer": schema.BoolAttribute{
				MarkdownDescription: "Whether kexec installer is enabled for this machine.",
				Computed:            true,
			},
			"not_found_ok": schema.BoolAttribute{
				MarkdownDescription: "If `true` (default), the data source returns null values instead of erroring when the machine does not exist. Set to `false` to fail hard when the machine is missing.",
				Optional:            true,
			},
		},
	}
}

func (d *MachineDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*durantic.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *durantic.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *MachineDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MachineDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	selectorCount := countSetStrings(data.UUID, data.Hostname)
	if selectorCount != 1 {
		resp.Diagnostics.AddError(
			"Invalid Machine Lookup",
			"Exactly one of uuid or hostname must be set.",
		)
		return
	}

	if isKnownString(data.UUID) {
		machine, httpResp, err := d.client.MachinesAPI.
			ProvisioningApiGetMachine(ctx, data.UUID.ValueString()).
			Execute()
		if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
			if httpResp != nil && httpResp.StatusCode == 404 && notFoundOk(data.NotFoundOk) {
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
			resp.Diagnostics.AddError(
				"Error Reading Machine",
				fmt.Sprintf("Could not read machine %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
			)
			return
		}
		resp.Diagnostics.Append(mapMachineResponseToCommonModel(machine, &data.MachineCommonModel)...)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	machines, httpResp, err := d.client.MachinesAPI.ProvisioningApiListMachines(ctx).Execute()
	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		resp.Diagnostics.AddError(
			"Error Listing Machines",
			fmt.Sprintf("Could not list machines: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	matches := make([]durantic.MachineSchema, 0, 1)
	for _, machine := range machines {
		if machine.GetHostname() == data.Hostname.ValueString() {
			matches = append(matches, machine)
		}
	}

	if len(matches) == 0 {
		if notFoundOk(data.NotFoundOk) {
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"No Machine Found",
			fmt.Sprintf("Could not find a machine with hostname %q.", data.Hostname.ValueString()),
		)
		return
	}

	if len(matches) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Machines Found",
			fmt.Sprintf("Found %d machines with hostname %q. Use uuid for an unambiguous lookup.", len(matches), data.Hostname.ValueString()),
		)
		return
	}

	machine, httpResp, err := d.client.MachinesAPI.
		ProvisioningApiGetMachine(ctx, matches[0].GetUuid()).
		Execute()
	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		resp.Diagnostics.AddError(
			"Error Reading Machine",
			fmt.Sprintf("Could not read machine %s: %s", matches[0].GetUuid(), extractAPIError(httpResp, err)),
		)
		return
	}

	resp.Diagnostics.Append(mapMachineResponseToCommonModel(machine, &data.MachineCommonModel)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// notFoundOk returns true when the attribute is unset (null), making lenient behavior the default.
func notFoundOk(v types.Bool) bool {
	return v.IsNull() || v.ValueBool()
}

func countSetStrings(values ...types.String) int {
	count := 0
	for _, value := range values {
		if isKnownString(value) {
			count++
		}
	}
	return count
}

func isKnownString(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueString() != ""
}
