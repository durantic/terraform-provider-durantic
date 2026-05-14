// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MeshNetworkResource{}
var _ resource.ResourceWithImportState = &MeshNetworkResource{}

func NewMeshNetworkResource() resource.Resource {
	return &MeshNetworkResource{}
}

// MeshNetworkResource defines the resource implementation.
type MeshNetworkResource struct {
	client *durantic.APIClient
}

// MeshNetworkResourceModel describes the resource data model.
type MeshNetworkResourceModel struct {
	UUID               types.String `tfsdk:"uuid"`
	Name               types.String `tfsdk:"name"`
	NetworkCidr        types.String `tfsdk:"network_cidr"`
	IsDefault          types.Bool   `tfsdk:"is_default"`
	RouteReflectorMode types.Bool   `tfsdk:"route_reflector_mode"`
	AvailableIpCount   types.Int64  `tfsdk:"available_ip_count"`
	MachineCount       types.Int64  `tfsdk:"machine_count"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (r *MeshNetworkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mesh_network"
}

func (r *MeshNetworkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Mesh network resource for Durantic infrastructure",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the mesh network",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the mesh network",
				Required:            true,
			},
			"network_cidr": schema.StringAttribute{
				MarkdownDescription: "CIDR block for the mesh network. Changing this forces a new resource to be created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether this is the default mesh network",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"route_reflector_mode": schema.BoolAttribute{
				MarkdownDescription: "Whether route reflector mode is enabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"available_ip_count": schema.Int64Attribute{
				MarkdownDescription: "Number of available IP addresses in the network",
				Computed:            true,
			},
			"machine_count": schema.Int64Attribute{
				MarkdownDescription: "Number of machines connected to the network",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the mesh network was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the mesh network was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *MeshNetworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MeshNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MeshNetworkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := durantic.NewCreateMeshNetworkSchema(data.Name.ValueString(), data.NetworkCidr.ValueString())
	createReq.SetIsDefault(data.IsDefault.ValueBool())
	createReq.SetRouteReflectorMode(data.RouteReflectorMode.ValueBool())

	meshNetwork, httpResp, err := r.client.MeshNetworksAPI.
		ControlplaneApiCreateMeshNetwork(ctx).
		CreateMeshNetworkSchema(*createReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 300 {
			resp.Diagnostics.AddError(
				"Error Creating Mesh Network",
				fmt.Sprintf("Could not create mesh network, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			var raw meshNetworkRaw
			if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating Mesh Network",
					fmt.Sprintf("Could not parse mesh network response: %s", jsonErr),
				)
				return
			}
			mapRawToMeshNetworkModel(&raw, &data)
			tflog.Trace(ctx, "created mesh network")
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating Mesh Network",
			fmt.Sprintf("Could not create mesh network, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	mapMeshNetworkToModel(meshNetwork, &data)

	tflog.Trace(ctx, "created mesh network")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MeshNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MeshNetworkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	meshNetwork, httpResp, err := r.client.MeshNetworksAPI.
		ControlplaneApiGetMeshNetwork(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw meshNetworkRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToMeshNetworkModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading Mesh Network",
			fmt.Sprintf("Could not read mesh network %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapMeshNetworkToModel(meshNetwork, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MeshNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MeshNetworkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := durantic.NewUpdateMeshNetworkSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetIsDefault(data.IsDefault.ValueBool())
	updateReq.SetRouteReflectorMode(data.RouteReflectorMode.ValueBool())

	meshNetwork, httpResp, err := r.client.MeshNetworksAPI.
		ControlplaneApiUpdateMeshNetwork(ctx, data.UUID.ValueString()).
		UpdateMeshNetworkSchema(*updateReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw meshNetworkRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToMeshNetworkModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Updating Mesh Network",
			fmt.Sprintf("Could not update mesh network %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapMeshNetworkToModel(meshNetwork, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MeshNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MeshNetworkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.MeshNetworksAPI.
		ControlplaneApiDeleteMeshNetwork(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "mesh network already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Mesh Network",
			fmt.Sprintf("Could not delete mesh network %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted mesh network")
}

func (r *MeshNetworkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

type meshNetworkRaw struct {
	UUID               string `json:"uuid"`
	Name               string `json:"name"`
	NetworkCidr        string `json:"network_cidr"`
	IsDefault          bool   `json:"is_default"`
	RouteReflectorMode bool   `json:"route_reflector_mode"`
	AvailableIpCount   int32  `json:"available_ip_count"`
	MachineCount       int32  `json:"machine_count"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

func mapRawToMeshNetworkModel(raw *meshNetworkRaw, model *MeshNetworkResourceModel) {
	model.UUID = types.StringValue(raw.UUID)
	model.Name = types.StringValue(raw.Name)
	model.NetworkCidr = types.StringValue(raw.NetworkCidr)
	model.IsDefault = types.BoolValue(raw.IsDefault)
	model.RouteReflectorMode = types.BoolValue(raw.RouteReflectorMode)
	model.AvailableIpCount = types.Int64Value(int64(raw.AvailableIpCount))
	model.MachineCount = types.Int64Value(int64(raw.MachineCount))
	model.CreatedAt = types.StringValue(raw.CreatedAt)
	model.UpdatedAt = types.StringValue(raw.UpdatedAt)
}

// Helper function to map API schema to Terraform model.
func mapMeshNetworkToModel(n *durantic.MeshNetworkSchema, model *MeshNetworkResourceModel) {
	model.UUID = types.StringValue(n.GetUuid())
	model.Name = types.StringValue(n.GetName())
	model.NetworkCidr = types.StringValue(n.GetNetworkCidr())
	model.IsDefault = types.BoolValue(n.GetIsDefault())
	model.RouteReflectorMode = types.BoolValue(n.GetRouteReflectorMode())
	model.AvailableIpCount = types.Int64Value(int64(n.GetAvailableIpCount()))
	model.MachineCount = types.Int64Value(int64(n.GetMachineCount()))
	model.CreatedAt = types.StringValue(n.GetCreatedAt())
	model.UpdatedAt = types.StringValue(n.GetUpdatedAt())
}
