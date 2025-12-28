// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &MachineDataSource{}

func NewMachineDataSource() datasource.DataSource {
	return &MachineDataSource{}
}

// MachineDataSource defines the data source implementation.
type MachineDataSource struct {
	client *durantic.APIClient
}

// MachineDataSourceModel describes the data source data model.
type MachineDataSourceModel struct {
	UUID                  types.String `tfsdk:"uuid"`
	Hostname              types.String `tfsdk:"hostname"`
	SystemUuid            types.String `tfsdk:"system_uuid"`
	NeedsProvisioning     types.Bool   `tfsdk:"needs_provisioning"`
	MeshNetworkUuid       types.String `tfsdk:"mesh_network_uuid"`
	MeshNetworkName       types.String `tfsdk:"mesh_network_name"`
	MeshNetworkCidr       types.String `tfsdk:"mesh_network_cidr"`
	WgIpAddress           types.String `tfsdk:"wg_ip_address"`
	WgPubkey              types.String `tfsdk:"wg_pubkey"`
	AdvertisedRoutes      types.List   `tfsdk:"advertised_routes"`
	DiscoveredIpAddresses types.List   `tfsdk:"discovered_ip_addresses"`
	NetplanConfig         types.String `tfsdk:"netplan_config"`
	DockerRegistryAuth    types.String `tfsdk:"docker_registry_auth"`
	TargetDisk            types.String `tfsdk:"target_disk"`
	RoleNames             types.List   `tfsdk:"role_names"`
	IsOnline              types.Bool   `tfsdk:"is_online"`
	IsInInitrd            types.Bool   `tfsdk:"is_in_initrd"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
	HardwareSummary       types.String `tfsdk:"hardware_summary"`
}

func (d *MachineDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine"
}

func (d *MachineDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get detailed information about a specific machine by UUID.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the machine",
				Required:            true,
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
			"mesh_network_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the assigned mesh network",
				Computed:            true,
			},
			"mesh_network_name": schema.StringAttribute{
				MarkdownDescription: "Name of the assigned mesh network",
				Computed:            true,
			},
			"mesh_network_cidr": schema.StringAttribute{
				MarkdownDescription: "CIDR of the assigned mesh network",
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
			"advertised_routes": schema.ListAttribute{
				MarkdownDescription: "Routes advertised by the machine",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"discovered_ip_addresses": schema.ListAttribute{
				MarkdownDescription: "Discovered IP addresses",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"netplan_config": schema.StringAttribute{
				MarkdownDescription: "Netplan configuration",
				Computed:            true,
			},
			"docker_registry_auth": schema.StringAttribute{
				MarkdownDescription: "Docker registry authentication configuration (JSON)",
				Computed:            true,
				Sensitive:           true,
			},
			"target_disk": schema.StringAttribute{
				MarkdownDescription: "Target disk for provisioning",
				Computed:            true,
			},
			"role_names": schema.ListAttribute{
				MarkdownDescription: "List of role names assigned to the machine",
				Computed:            true,
				ElementType:         types.StringType,
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
			"hardware_summary": schema.StringAttribute{
				MarkdownDescription: "Hardware information (JSON)",
				Computed:            true,
			},
		},
	}
}

func (d *MachineDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to get machine details
	machine, httpResp, err := d.client.MachinesAPI.
		ProvisioningApiGetMachine(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Machine",
			fmt.Sprintf("Could not read machine %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Map basic fields
	data.Hostname = types.StringValue(machine.GetHostname())
	data.NeedsProvisioning = types.BoolValue(machine.GetNeedsProvisioning())
	data.CreatedAt = types.StringValue(machine.GetCreatedAt())
	data.UpdatedAt = types.StringValue(machine.GetUpdatedAt())

	// Handle nullable system_uuid
	if machine.SystemUuid.IsSet() {
		data.SystemUuid = types.StringValue(*machine.SystemUuid.Get())
	} else {
		data.SystemUuid = types.StringNull()
	}

	// Handle mesh network - flatten nested object
	if machine.MeshNetwork.IsSet() {
		network := machine.GetMeshNetwork()
		data.MeshNetworkUuid = types.StringValue(network.GetUuid())
		data.MeshNetworkName = types.StringValue(network.GetName())
		data.MeshNetworkCidr = types.StringValue(network.GetNetworkCidr())
	} else {
		data.MeshNetworkUuid = types.StringNull()
		data.MeshNetworkName = types.StringNull()
		data.MeshNetworkCidr = types.StringNull()
	}

	// Handle nullable wg_ip_address
	if machine.WgIpAddress.IsSet() {
		data.WgIpAddress = types.StringValue(*machine.WgIpAddress.Get())
	} else {
		data.WgIpAddress = types.StringNull()
	}

	// Handle nullable wg_pubkey
	if machine.WgPubkey.IsSet() {
		data.WgPubkey = types.StringValue(*machine.WgPubkey.Get())
	} else {
		data.WgPubkey = types.StringNull()
	}

	// Handle advertised_routes list
	if len(machine.GetAdvertisedRoutes()) > 0 {
		routes, diags := types.ListValueFrom(ctx, types.StringType, machine.GetAdvertisedRoutes())
		resp.Diagnostics.Append(diags...)
		data.AdvertisedRoutes = routes
	} else {
		data.AdvertisedRoutes = types.ListNull(types.StringType)
	}

	// Handle discovered_ip_addresses list
	if len(machine.GetDiscoveredIpAddresses()) > 0 {
		ips, diags := types.ListValueFrom(ctx, types.StringType, machine.GetDiscoveredIpAddresses())
		resp.Diagnostics.Append(diags...)
		data.DiscoveredIpAddresses = ips
	} else {
		data.DiscoveredIpAddresses = types.ListNull(types.StringType)
	}

	// Handle nullable netplan_config
	if machine.NetplanConfig.IsSet() {
		data.NetplanConfig = types.StringValue(*machine.NetplanConfig.Get())
	} else {
		data.NetplanConfig = types.StringNull()
	}

	// Handle docker_registry_auth - convert map to JSON string
	if len(machine.GetDockerRegistryAuth()) > 0 {
		authJSON, err := json.Marshal(machine.GetDockerRegistryAuth())
		if err == nil {
			data.DockerRegistryAuth = types.StringValue(string(authJSON))
		} else {
			data.DockerRegistryAuth = types.StringNull()
		}
	} else {
		data.DockerRegistryAuth = types.StringNull()
	}

	// Handle nullable target_disk
	if machine.TargetDisk.IsSet() {
		data.TargetDisk = types.StringValue(*machine.TargetDisk.Get())
	} else {
		data.TargetDisk = types.StringNull()
	}

	// Handle role_names list
	if len(machine.GetRoleNames()) > 0 {
		roles, diags := types.ListValueFrom(ctx, types.StringType, machine.GetRoleNames())
		resp.Diagnostics.Append(diags...)
		data.RoleNames = roles
	} else {
		data.RoleNames = types.ListNull(types.StringType)
	}

	// Handle optional bool pointers
	if machine.IsOnline != nil {
		data.IsOnline = types.BoolValue(*machine.IsOnline)
	} else {
		data.IsOnline = types.BoolValue(false)
	}

	if machine.IsInInitrd != nil {
		data.IsInInitrd = types.BoolValue(*machine.IsInInitrd)
	} else {
		data.IsInInitrd = types.BoolValue(false)
	}

	// Handle hardware - convert to JSON string
	if machine.Hardware.IsSet() {
		hardwareJSON, err := json.Marshal(machine.GetHardware())
		if err == nil {
			data.HardwareSummary = types.StringValue(string(hardwareJSON))
		} else {
			data.HardwareSummary = types.StringNull()
		}
	} else {
		data.HardwareSummary = types.StringNull()
	}

	// Write logs using the tflog package
	tflog.Trace(ctx, "read machine data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
