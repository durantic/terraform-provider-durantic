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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &VIPResource{}
var _ resource.ResourceWithImportState = &VIPResource{}

func NewVIPResource() resource.Resource {
	return &VIPResource{}
}

type VIPResource struct {
	client *durantic.APIClient
}

type VIPResourceModel struct {
	UUID                          types.String `tfsdk:"uuid"`
	Name                          types.String `tfsdk:"name"`
	Enabled                       types.Bool   `tfsdk:"enabled"`
	Address                       types.String `tfsdk:"address"`
	HealthCheckType               types.String `tfsdk:"health_check_type"`
	HealthCheckTarget             types.String `tfsdk:"health_check_target"`
	HealthCheckIntervalSeconds    types.Int64  `tfsdk:"health_check_interval_seconds"`
	HealthCheckTimeoutSeconds     types.Int64  `tfsdk:"health_check_timeout_seconds"`
	HealthCheckHealthyThreshold   types.Int64  `tfsdk:"health_check_healthy_threshold"`
	HealthCheckUnhealthyThreshold types.Int64  `tfsdk:"health_check_unhealthy_threshold"`
	HealthCheckHoldoffSeconds     types.Int64  `tfsdk:"health_check_holdoff_seconds"`
	MachineUUIDs                  types.List   `tfsdk:"machine_uuids"`
	MachineCount                  types.Int64  `tfsdk:"machine_count"`
	CreatedAt                     types.String `tfsdk:"created_at"`
	UpdatedAt                     types.String `tfsdk:"updated_at"`
}

func (r *VIPResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vip"
}

func (r *VIPResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VIP (Virtual IP) resource for Durantic network load balancing",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the VIP",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the VIP",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the VIP is enabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"address": schema.StringAttribute{
				MarkdownDescription: "IP address for the VIP",
				Required:            true,
			},
			"health_check_type": schema.StringAttribute{
				MarkdownDescription: "Health check type. Valid values: `\"\"` (disabled), `tcp`, `http`, `https`, `grpc`, `grpcs`, `exec`",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"health_check_target": schema.StringAttribute{
				MarkdownDescription: "Health check target (path or command depending on type)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"health_check_interval_seconds": schema.Int64Attribute{
				MarkdownDescription: "Interval in seconds between health checks",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(5),
			},
			"health_check_timeout_seconds": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds for each health check",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3),
			},
			"health_check_healthy_threshold": schema.Int64Attribute{
				MarkdownDescription: "Number of consecutive successes required to mark as healthy",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2),
			},
			"health_check_unhealthy_threshold": schema.Int64Attribute{
				MarkdownDescription: "Number of consecutive failures required to mark as unhealthy",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3),
			},
			"health_check_holdoff_seconds": schema.Int64Attribute{
				MarkdownDescription: "Seconds to wait after creation before starting health checks",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"machine_uuids": schema.ListAttribute{
				MarkdownDescription: "UUIDs of machines to associate with this VIP",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"machine_count": schema.Int64Attribute{
				MarkdownDescription: "Number of machines associated with this VIP",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the VIP was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the VIP was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *VIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VIPResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Named setters, not positional args: the generated constructor takes its
	// required params in ALPHABETICAL order, so regenerating the client can
	// silently reorder them. Both of these are strings, so a swap compiles,
	// vets and lints clean -- it only shows up as a 400 from the API.
	createReq := durantic.NewCreateVIPSchemaWithDefaults()
	createReq.SetName(data.Name.ValueString())
	createReq.SetAddress(data.Address.ValueString())
	createReq.SetEnabled(data.Enabled.ValueBool())
	createReq.SetHealthCheckType(durantic.HealthCheckType(data.HealthCheckType.ValueString()))
	createReq.SetHealthCheckTarget(data.HealthCheckTarget.ValueString())
	createReq.SetHealthCheckIntervalSeconds(int32(data.HealthCheckIntervalSeconds.ValueInt64()))
	createReq.SetHealthCheckTimeoutSeconds(int32(data.HealthCheckTimeoutSeconds.ValueInt64()))
	createReq.SetHealthCheckHealthyThreshold(int32(data.HealthCheckHealthyThreshold.ValueInt64()))
	createReq.SetHealthCheckUnhealthyThreshold(int32(data.HealthCheckUnhealthyThreshold.ValueInt64()))
	createReq.SetHealthCheckHoldoffSeconds(int32(data.HealthCheckHoldoffSeconds.ValueInt64()))

	if !data.MachineUUIDs.IsNull() && !data.MachineUUIDs.IsUnknown() {
		var machineUUIDs []string
		resp.Diagnostics.Append(data.MachineUUIDs.ElementsAs(ctx, &machineUUIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.SetMachineUuids(machineUUIDs)
	}

	vip, httpResp, err := r.client.VIPsAPI.
		ControlplaneApiCreateVip(ctx).
		CreateVIPSchema(*createReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 300 {
			resp.Diagnostics.AddError(
				"Error Creating VIP",
				fmt.Sprintf("Could not create VIP, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			var raw vipRaw
			if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating VIP",
					fmt.Sprintf("Could not parse VIP response: %s", jsonErr),
				)
				return
			}
			mapRawToVIPModel(&raw, &data)
			tflog.Trace(ctx, "created VIP")
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating VIP",
			fmt.Sprintf("Could not create VIP, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	mapVIPToModel(vip, &data)

	tflog.Trace(ctx, "created VIP")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VIPResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vip, httpResp, err := r.client.VIPsAPI.
		ControlplaneApiGetVip(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw vipRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToVIPModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading VIP",
			fmt.Sprintf("Could not read VIP %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapVIPToModel(vip, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VIPResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := durantic.NewUpdateVIPSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetEnabled(data.Enabled.ValueBool())
	updateReq.SetAddress(data.Address.ValueString())
	updateReq.SetHealthCheckType(durantic.HealthCheckType(data.HealthCheckType.ValueString()))
	updateReq.SetHealthCheckTarget(data.HealthCheckTarget.ValueString())
	updateReq.SetHealthCheckIntervalSeconds(int32(data.HealthCheckIntervalSeconds.ValueInt64()))
	updateReq.SetHealthCheckTimeoutSeconds(int32(data.HealthCheckTimeoutSeconds.ValueInt64()))
	updateReq.SetHealthCheckHealthyThreshold(int32(data.HealthCheckHealthyThreshold.ValueInt64()))
	updateReq.SetHealthCheckUnhealthyThreshold(int32(data.HealthCheckUnhealthyThreshold.ValueInt64()))
	updateReq.SetHealthCheckHoldoffSeconds(int32(data.HealthCheckHoldoffSeconds.ValueInt64()))

	if !data.MachineUUIDs.IsNull() && !data.MachineUUIDs.IsUnknown() {
		var machineUUIDs []string
		resp.Diagnostics.Append(data.MachineUUIDs.ElementsAs(ctx, &machineUUIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.SetMachineUuids(machineUUIDs)
	} else {
		updateReq.SetMachineUuids([]string{})
	}

	vip, httpResp, err := r.client.VIPsAPI.
		ControlplaneApiUpdateVip(ctx, data.UUID.ValueString()).
		UpdateVIPSchema(*updateReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw vipRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToVIPModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Updating VIP",
			fmt.Sprintf("Could not update VIP %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapVIPToModel(vip, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VIPResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.VIPsAPI.
		ControlplaneApiDeleteVip(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "VIP already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting VIP",
			fmt.Sprintf("Could not delete VIP %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted VIP")
}

func (r *VIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

type vipRaw struct {
	UUID                          string  `json:"uuid"`
	Name                          string  `json:"name"`
	Enabled                       bool    `json:"enabled"`
	Address                       *string `json:"address,omitempty"`
	HealthCheckType               string  `json:"health_check_type"`
	HealthCheckTarget             string  `json:"health_check_target"`
	HealthCheckIntervalSeconds    int32   `json:"health_check_interval_seconds"`
	HealthCheckTimeoutSeconds     int32   `json:"health_check_timeout_seconds"`
	HealthCheckHealthyThreshold   int32   `json:"health_check_healthy_threshold"`
	HealthCheckUnhealthyThreshold int32   `json:"health_check_unhealthy_threshold"`
	HealthCheckHoldoffSeconds     int32   `json:"health_check_holdoff_seconds"`
	MachineCount                  int32   `json:"machine_count"`
	CreatedAt                     string  `json:"created_at"`
	UpdatedAt                     string  `json:"updated_at"`
}

func mapRawToVIPModel(raw *vipRaw, model *VIPResourceModel) {
	model.UUID = types.StringValue(raw.UUID)
	model.Name = types.StringValue(raw.Name)
	model.Enabled = types.BoolValue(raw.Enabled)
	model.MachineCount = types.Int64Value(int64(raw.MachineCount))
	model.CreatedAt = types.StringValue(raw.CreatedAt)
	model.UpdatedAt = types.StringValue(raw.UpdatedAt)

	if raw.Address != nil {
		model.Address = types.StringValue(*raw.Address)
	}

	model.HealthCheckType = types.StringValue(raw.HealthCheckType)
	model.HealthCheckTarget = types.StringValue(raw.HealthCheckTarget)
	model.HealthCheckIntervalSeconds = types.Int64Value(int64(raw.HealthCheckIntervalSeconds))
	model.HealthCheckTimeoutSeconds = types.Int64Value(int64(raw.HealthCheckTimeoutSeconds))
	model.HealthCheckHealthyThreshold = types.Int64Value(int64(raw.HealthCheckHealthyThreshold))
	model.HealthCheckUnhealthyThreshold = types.Int64Value(int64(raw.HealthCheckUnhealthyThreshold))
	model.HealthCheckHoldoffSeconds = types.Int64Value(int64(raw.HealthCheckHoldoffSeconds))
}

// mapVIPToModel maps an API VIPSchema to the Terraform model.
// machine_uuids is not updated here because the response returns machines as full objects
// rather than UUIDs; the write-only machine_uuids field retains its state value.
func mapVIPToModel(vip *durantic.VIPSchema, model *VIPResourceModel) {
	model.UUID = types.StringValue(vip.GetUuid())
	model.Name = types.StringValue(vip.GetName())
	model.Enabled = types.BoolValue(vip.GetEnabled())
	model.MachineCount = types.Int64Value(int64(vip.GetMachineCount()))
	model.CreatedAt = types.StringValue(vip.GetCreatedAt())
	model.UpdatedAt = types.StringValue(vip.GetUpdatedAt())

	if addr, ok := vip.GetAddressOk(); ok && addr != nil {
		model.Address = types.StringValue(*addr)
	}

	model.HealthCheckType = types.StringValue(string(vip.GetHealthCheckType()))
	model.HealthCheckTarget = types.StringValue(vip.GetHealthCheckTarget())
	model.HealthCheckIntervalSeconds = types.Int64Value(int64(vip.GetHealthCheckIntervalSeconds()))
	model.HealthCheckTimeoutSeconds = types.Int64Value(int64(vip.GetHealthCheckTimeoutSeconds()))
	model.HealthCheckHealthyThreshold = types.Int64Value(int64(vip.GetHealthCheckHealthyThreshold()))
	model.HealthCheckUnhealthyThreshold = types.Int64Value(int64(vip.GetHealthCheckUnhealthyThreshold()))
	model.HealthCheckHoldoffSeconds = types.Int64Value(int64(vip.GetHealthCheckHoldoffSeconds()))
}
