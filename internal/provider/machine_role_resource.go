// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MachineRoleResource{}
var _ resource.ResourceWithImportState = &MachineRoleResource{}

func NewMachineRoleResource() resource.Resource {
	return &MachineRoleResource{}
}

// MachineRoleResource defines the resource implementation.
type MachineRoleResource struct {
	client *durantic.APIClient
}

// MachineRoleResourceModel describes the resource data model.
type MachineRoleResourceModel struct {
	UUID              types.String `tfsdk:"uuid"`
	Name              types.String `tfsdk:"name"`
	MergePriority     types.Int64  `tfsdk:"merge_priority"`
	TemplateData      types.String `tfsdk:"template_data"`
	Description       types.String `tfsdk:"description"`
	ImageUUID         types.String `tfsdk:"image_uuid"`
	RequiresMesh      types.Bool   `tfsdk:"requires_mesh"`
	IsOfficial        types.Bool   `tfsdk:"is_official"`
	ForkedFromUUID    types.String `tfsdk:"forked_from_uuid"`
	VipUUID           types.String `tfsdk:"vip_uuid"`
	RequiredImageName types.String `tfsdk:"required_image_name"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (r *MachineRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine_role"
}

func (r *MachineRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Machine role resource for Durantic infrastructure",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the machine role",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the machine role",
				Required:            true,
			},
			"merge_priority": schema.Int64Attribute{
				MarkdownDescription: "Merge priority for the machine role (default 100)",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(100),
			},
			"template_data": schema.StringAttribute{
				MarkdownDescription: "Template data for the machine role",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the machine role",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"image_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the image associated with this machine role",
				Optional:            true,
			},
			"vip_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the VIP associated with this machine role",
				Optional:            true,
			},
			"required_image_name": schema.StringAttribute{
				MarkdownDescription: "Required image name for this machine role (read-only, computed by API)",
				Computed:            true,
			},
			"requires_mesh": schema.BoolAttribute{
				MarkdownDescription: "Whether this machine role requires a mesh network",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"is_official": schema.BoolAttribute{
				MarkdownDescription: "Whether this is an official machine role",
				Computed:            true,
			},
			"forked_from_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the machine role this was forked from",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the machine role was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the machine role was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *MachineRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MachineRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MachineRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createReq := durantic.NewCreateMachineRoleSchema(data.Name.ValueString())
	createReq.SetMergePriority(int32(data.MergePriority.ValueInt64()))
	createReq.SetTemplateData(data.TemplateData.ValueString())
	createReq.SetDescription(data.Description.ValueString())
	createReq.SetRequiresMesh(data.RequiresMesh.ValueBool())

	if !data.ImageUUID.IsNull() {
		createReq.SetImageUuid(data.ImageUUID.ValueString())
	}

	if !data.VipUUID.IsNull() {
		createReq.SetVipUuid(data.VipUUID.ValueString())
	}

	machineRole, httpResp, err := r.client.MachineRolesAPI.
		ControlplaneApiCreateMachineRole(ctx).
		CreateMachineRoleSchema(*createReq).
		Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Machine Role",
			fmt.Sprintf("Could not create machine role, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	mapMachineRoleToModel(machineRole, &data)

	tflog.Trace(ctx, "created machine role")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MachineRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	machineRole, httpResp, err := r.client.MachineRolesAPI.
		ControlplaneApiGetMachineRole(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Machine Role",
			fmt.Sprintf("Could not read machine role %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapMachineRoleToModel(machineRole, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data MachineRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := durantic.NewUpdateMachineRoleSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetMergePriority(int32(data.MergePriority.ValueInt64()))
	updateReq.SetTemplateData(data.TemplateData.ValueString())
	updateReq.SetDescription(data.Description.ValueString())
	updateReq.SetRequiresMesh(data.RequiresMesh.ValueBool())

	if !data.ImageUUID.IsNull() {
		updateReq.SetImageUuid(data.ImageUUID.ValueString())
	}

	if !data.VipUUID.IsNull() {
		updateReq.SetVipUuid(data.VipUUID.ValueString())
	}

	machineRole, httpResp, err := r.client.MachineRolesAPI.
		ControlplaneApiUpdateMachineRole(ctx, data.UUID.ValueString()).
		UpdateMachineRoleSchema(*updateReq).
		Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Machine Role",
			fmt.Sprintf("Could not update machine role %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapMachineRoleToModel(machineRole, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MachineRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.MachineRolesAPI.
		ControlplaneApiDeleteMachineRole(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "machine role already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Machine Role",
			fmt.Sprintf("Could not delete machine role %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted machine role")
}

func (r *MachineRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// Helper function to map API schema to Terraform model.
func mapMachineRoleToModel(role *durantic.MachineRoleSchema, model *MachineRoleResourceModel) {
	model.UUID = types.StringValue(role.GetUuid())
	model.Name = types.StringValue(role.GetName())
	model.MergePriority = types.Int64Value(int64(role.GetMergePriority()))
	model.TemplateData = types.StringValue(role.GetTemplateData())
	model.Description = types.StringValue(role.GetDescription())

	imageRef, ok := role.GetImageOk()
	if ok && imageRef != nil {
		model.ImageUUID = types.StringValue(imageRef.GetUuid())
	} else {
		model.ImageUUID = types.StringNull()
	}

	model.RequiresMesh = types.BoolValue(role.GetRequiresMesh())
	model.IsOfficial = types.BoolValue(role.GetIsOfficial())

	forkedFrom, ok := role.GetForkedFromUuidOk()
	if ok && forkedFrom != nil {
		model.ForkedFromUUID = types.StringValue(*forkedFrom)
	} else {
		model.ForkedFromUUID = types.StringNull()
	}

	vipRef, ok := role.GetVipOk()
	if ok && vipRef != nil {
		model.VipUUID = types.StringValue(vipRef.GetUuid())
	} else {
		model.VipUUID = types.StringNull()
	}

	if role.HasRequiredImageName() {
		model.RequiredImageName = types.StringValue(role.GetRequiredImageName())
	} else {
		model.RequiredImageName = types.StringNull()
	}

	model.CreatedAt = types.StringValue(role.GetCreatedAt())
	model.UpdatedAt = types.StringValue(role.GetUpdatedAt())
}
