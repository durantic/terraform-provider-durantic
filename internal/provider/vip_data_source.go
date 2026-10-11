// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VIPDataSource{}

func NewVIPDataSource() datasource.DataSource {
	return &VIPDataSource{}
}

type VIPDataSource struct {
	client *durantic.APIClient
}

func (d *VIPDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vip"
}

func (d *VIPDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Durantic VIP (virtual IP) by UUID or name.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the VIP. Set either `uuid` or `name`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the VIP. Set either `uuid` or `name`.",
				Optional:            true,
				Computed:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the VIP is enabled.",
				Computed:            true,
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "IP address of the VIP.",
				Computed:            true,
			},
			"health_check_type": schema.StringAttribute{
				MarkdownDescription: "Health check type (`\"\"` when disabled, otherwise e.g. `tcp`, `http`, `https`, `grpc`, `grpcs`).",
				Computed:            true,
			},
			"health_check_target": schema.StringAttribute{
				MarkdownDescription: "Health check target (path or command depending on type).",
				Computed:            true,
			},
			"health_check_interval_seconds": schema.Int64Attribute{
				MarkdownDescription: "Interval in seconds between health checks.",
				Computed:            true,
			},
			"health_check_timeout_seconds": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds for a single health check.",
				Computed:            true,
			},
			"health_check_healthy_threshold": schema.Int64Attribute{
				MarkdownDescription: "Consecutive successful checks before a machine is considered healthy.",
				Computed:            true,
			},
			"health_check_unhealthy_threshold": schema.Int64Attribute{
				MarkdownDescription: "Consecutive failed checks before a machine is considered unhealthy.",
				Computed:            true,
			},
			"health_check_holdoff_seconds": schema.Int64Attribute{
				MarkdownDescription: "Seconds to wait after startup before health checks begin counting.",
				Computed:            true,
			},
			"machine_uuids": schema.ListAttribute{
				MarkdownDescription: "UUIDs of the machines the VIP is assigned to.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"machine_count": schema.Int64Attribute{
				MarkdownDescription: "Number of machines the VIP is assigned to.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the VIP was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the VIP was last updated.",
				Computed:            true,
			},
		},
	}
}

func (d *VIPDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VIPDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VIPResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !requireUUIDOrName("VIP", data.UUID, data.Name, &resp.Diagnostics) {
		return
	}

	var vip *durantic.VIPSchema

	if isKnownString(data.UUID) {
		got, httpResp, err := d.client.VIPsAPI.
			ControlplaneApiGetVip(ctx, data.UUID.ValueString()).
			Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading VIP",
				fmt.Sprintf("Could not read VIP %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
			)
			return
		}
		vip = got
	} else {
		vips, httpResp, err := d.client.VIPsAPI.ControlplaneApiListVips(ctx).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Listing VIPs",
				fmt.Sprintf("Could not list VIPs: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		match, ok := findUniqueByName("VIP", data.Name.ValueString(), vips, func(v durantic.VIPSchema) string { return v.GetName() }, &resp.Diagnostics)
		if !ok {
			return
		}
		vip = &match
	}

	resp.Diagnostics.Append(mapVIPToDataSourceModel(ctx, vip, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mapVIPToDataSourceModel maps a VIP onto the shared VIP model. Unlike the
// resource, a data source is read-only, so machine_uuids is filled from the
// machines the API reports rather than preserved from configuration.
func mapVIPToDataSourceModel(ctx context.Context, vip *durantic.VIPSchema, model *VIPResourceModel) diag.Diagnostics {
	mapVIPToModel(vip, model)

	// mapVIPToModel leaves address untouched when the API has none.
	if addr, ok := vip.GetAddressOk(); !ok || addr == nil {
		model.Address = types.StringNull()
	}

	machines := vip.GetMachines()
	uuids := make([]string, 0, len(machines))
	for _, m := range machines {
		uuids = append(uuids, m.GetUuid())
	}

	list, diags := types.ListValueFrom(ctx, types.StringType, uuids)
	model.MachineUUIDs = list
	return diags
}
