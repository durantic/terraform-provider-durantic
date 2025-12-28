// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	NetworkCIDR      types.String `tfsdk:"network_cidr"`
	IsDefault        types.Bool   `tfsdk:"is_default"`
	AvailableIPCount types.Int64  `tfsdk:"available_ip_count"`
	MachineCount     types.Int64  `tfsdk:"machine_count"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
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
				MarkdownDescription: "Network CIDR range (e.g., 10.0.0.0/16). Cannot be changed after creation.",
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
			"available_ip_count": schema.Int64Attribute{
				MarkdownDescription: "Number of available IP addresses in the network",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"machine_count": schema.Int64Attribute{
				MarkdownDescription: "Number of machines in the network",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the network was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the network was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *MeshNetworkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MeshNetworkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MeshNetworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build create request
	createReq := durantic.NewCreateMeshNetworkSchema(
		data.Name.ValueString(),
		data.NetworkCIDR.ValueString(),
	)

	if !data.IsDefault.IsNull() {
		isDefault := data.IsDefault.ValueBool()
		createReq.SetIsDefault(isDefault)
	}

	// Call API to create mesh network
	meshNetwork, httpResp, err := r.client.MeshNetworksAPI.
		ControlplaneApiCreateMeshNetwork(ctx).
		CreateMeshNetworkSchema(*createReq).
		Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Mesh Network",
			fmt.Sprintf("Could not create mesh network, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	// Map response to state
	mapMeshNetworkToModel(meshNetwork, &data)

	// Write logs using the tflog package
	tflog.Trace(ctx, "created mesh network")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MeshNetworkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MeshNetworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to get mesh network
	meshNetwork, httpResp, err := r.client.MeshNetworksAPI.
		ControlplaneApiGetMeshNetwork(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		// Handle 404 - resource deleted outside Terraform
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Mesh Network",
			fmt.Sprintf("Could not read mesh network %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Map response to state
	mapMeshNetworkToModel(meshNetwork, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MeshNetworkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MeshNetworkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build update request (only name and is_default can be updated)
	updateReq := durantic.NewUpdateMeshNetworkSchema()

	if !data.Name.IsNull() {
		updateReq.SetName(data.Name.ValueString())
	}

	if !data.IsDefault.IsNull() {
		updateReq.SetIsDefault(data.IsDefault.ValueBool())
	}

	// Call API to update mesh network
	meshNetwork, httpResp, err := r.client.MeshNetworksAPI.
		ControlplaneApiUpdateMeshNetwork(ctx, data.UUID.ValueString()).
		UpdateMeshNetworkSchema(*updateReq).
		Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Mesh Network",
			fmt.Sprintf("Could not update mesh network %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Map response to state
	mapMeshNetworkToModel(meshNetwork, &data)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MeshNetworkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MeshNetworkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to delete mesh network
	httpResp, err := r.client.MeshNetworksAPI.
		ControlplaneApiDeleteMeshNetwork(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		// Ignore 404 errors - resource already deleted
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

// Helper function to map API schema to Terraform model
func mapMeshNetworkToModel(meshNetwork *durantic.MeshNetworkSchema, model *MeshNetworkResourceModel) {
	model.UUID = types.StringValue(meshNetwork.GetUuid())
	model.Name = types.StringValue(meshNetwork.GetName())
	model.NetworkCIDR = types.StringValue(meshNetwork.GetNetworkCidr())
	model.IsDefault = types.BoolValue(meshNetwork.GetIsDefault())
	model.AvailableIPCount = types.Int64Value(int64(meshNetwork.GetAvailableIpCount()))
	model.MachineCount = types.Int64Value(int64(meshNetwork.GetMachineCount()))
	model.CreatedAt = types.StringValue(meshNetwork.GetCreatedAt())
	model.UpdatedAt = types.StringValue(meshNetwork.GetUpdatedAt())
}

// Helper function to extract error details from API responses
func extractAPIError(httpResp *http.Response, err error) string {
	if err != nil {
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			return fmt.Sprintf("%s: %s", httpResp.Status, apiErr.Error())
		}
		return err.Error()
	}
	return "unknown error"
}
