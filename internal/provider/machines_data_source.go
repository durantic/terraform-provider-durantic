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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &MachinesDataSource{}

func NewMachinesDataSource() datasource.DataSource {
	return &MachinesDataSource{}
}

// MachinesDataSource defines the data source implementation.
type MachinesDataSource struct {
	client *durantic.APIClient
}

// MachinesDataSourceModel describes the data source data model.
type MachinesDataSourceModel struct {
	Machines []MachineDataModel `tfsdk:"machines"`
	ID       types.String       `tfsdk:"id"`
}

// MachineDataModel describes a single machine in the list.
type MachineDataModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Hostname    types.String `tfsdk:"hostname"`
	IsOnline    types.Bool   `tfsdk:"is_online"`
	IsInInitrd  types.Bool   `tfsdk:"is_in_initrd"`
	RoleNames   types.List   `tfsdk:"role_names"`
	WgIpAddress types.String `tfsdk:"wg_ip_address"`
}

func (d *MachinesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machines"
}

func (d *MachinesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List all machines for the authenticated account. Provides a lightweight view of machine inventory.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Static identifier for the data source",
				Computed:            true,
			},
			"machines": schema.ListNestedAttribute{
				MarkdownDescription: "List of machines",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the machine",
							Computed:            true,
						},
						"hostname": schema.StringAttribute{
							MarkdownDescription: "Machine hostname",
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
						"role_names": schema.ListAttribute{
							MarkdownDescription: "List of role names assigned to the machine",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"wg_ip_address": schema.StringAttribute{
							MarkdownDescription: "WireGuard IP address assigned to the machine",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *MachinesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *MachinesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MachinesDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to list all machines
	machines, httpResp, err := d.client.MachinesAPI.
		ProvisioningApiListMachines(ctx).
		Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Machines",
			fmt.Sprintf("Could not list machines, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	// Map API response to Terraform model
	data.ID = types.StringValue("machines")
	data.Machines = make([]MachineDataModel, 0, len(machines))

	for _, machine := range machines {
		machineModel := MachineDataModel{
			UUID:     types.StringValue(machine.GetUuid()),
			Hostname: types.StringValue(machine.GetHostname()),
		}

		// Handle optional bool pointers
		if machine.IsOnline != nil {
			machineModel.IsOnline = types.BoolValue(*machine.IsOnline)
		} else {
			machineModel.IsOnline = types.BoolValue(false)
		}

		if machine.IsInInitrd != nil {
			machineModel.IsInInitrd = types.BoolValue(*machine.IsInInitrd)
		} else {
			machineModel.IsInInitrd = types.BoolValue(false)
		}

		// Handle role_names list
		if len(machine.GetRoleNames()) > 0 {
			roleNames, diags := types.ListValueFrom(ctx, types.StringType, machine.GetRoleNames())
			resp.Diagnostics.Append(diags...)
			machineModel.RoleNames = roleNames
		} else {
			machineModel.RoleNames = types.ListNull(types.StringType)
		}

		// Handle nullable WgIpAddress
		if machine.WgIpAddress.IsSet() {
			machineModel.WgIpAddress = types.StringValue(*machine.WgIpAddress.Get())
		} else {
			machineModel.WgIpAddress = types.StringNull()
		}

		data.Machines = append(data.Machines, machineModel)
	}

	// Write logs using the tflog package
	tflog.Trace(ctx, fmt.Sprintf("read %d machines", len(data.Machines)))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
