// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var _ datasource.DataSource = &MeshNetworkDataSource{}

func NewMeshNetworkDataSource() datasource.DataSource {
	return &MeshNetworkDataSource{}
}

type MeshNetworkDataSource struct {
	client *durantic.APIClient
}

func (d *MeshNetworkDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mesh_network"
}

func (d *MeshNetworkDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Durantic mesh network by UUID or name.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the mesh network. Set either `uuid` or `name`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the mesh network. Set either `uuid` or `name`.",
				Optional:            true,
				Computed:            true,
			},
			"network_cidr": schema.StringAttribute{
				MarkdownDescription: "CIDR block of the mesh network.",
				Computed:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether this is the account's default mesh network.",
				Computed:            true,
			},
			"route_reflector_mode": schema.BoolAttribute{
				MarkdownDescription: "Whether the mesh network uses route reflector mode.",
				Computed:            true,
			},
			"available_ip_count": schema.Int64Attribute{
				MarkdownDescription: "Number of IP addresses still available in the network.",
				Computed:            true,
			},
			"machine_count": schema.Int64Attribute{
				MarkdownDescription: "Number of machines in the mesh network.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the mesh network was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the mesh network was last updated.",
				Computed:            true,
			},
		},
	}
}

func (d *MeshNetworkDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MeshNetworkDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MeshNetworkResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !requireUUIDOrName("Mesh Network", data.UUID, data.Name, &resp.Diagnostics) {
		return
	}

	var network *durantic.MeshNetworkSchema

	if isKnownString(data.UUID) {
		got, httpResp, err := d.client.MeshNetworksAPI.
			ControlplaneApiGetMeshNetwork(ctx, data.UUID.ValueString()).
			Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Mesh Network",
				fmt.Sprintf("Could not read mesh network %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
			)
			return
		}
		network = got
	} else {
		networks, httpResp, err := d.client.MeshNetworksAPI.ControlplaneApiListMeshNetworks(ctx).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Listing Mesh Networks",
				fmt.Sprintf("Could not list mesh networks: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		match, ok := findUniqueByName("Mesh Network", data.Name.ValueString(), networks, func(n durantic.MeshNetworkSchema) string { return n.GetName() }, &resp.Diagnostics)
		if !ok {
			return
		}
		network = &match
	}

	mapMeshNetworkToModel(network, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
