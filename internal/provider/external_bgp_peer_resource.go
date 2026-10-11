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

var _ resource.Resource = &ExternalBGPPeerResource{}
var _ resource.ResourceWithImportState = &ExternalBGPPeerResource{}

func NewExternalBGPPeerResource() resource.Resource {
	return &ExternalBGPPeerResource{}
}

type ExternalBGPPeerResource struct {
	client *durantic.APIClient
}

type ExternalBGPPeerResourceModel struct {
	UUID             types.String `tfsdk:"uuid"`
	MachineUUID      types.String `tfsdk:"machine_uuid"`
	MachineHostname  types.String `tfsdk:"machine_hostname"`
	Name             types.String `tfsdk:"name"`
	PeerIP           types.String `tfsdk:"peer_ip"`
	PeerASN          types.Int64  `tfsdk:"peer_asn"`
	LocalIP          types.String `tfsdk:"local_ip"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Password         types.String `tfsdk:"password"`
	ImportPolicyUUID types.String `tfsdk:"import_policy_uuid"`
	ExportPolicyUUID types.String `tfsdk:"export_policy_uuid"`
	ImportPolicyName types.String `tfsdk:"import_policy_name"`
	ExportPolicyName types.String `tfsdk:"export_policy_name"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (r *ExternalBGPPeerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_bgp_peer"
}

func (r *ExternalBGPPeerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Durantic external BGP peer — a BGP session from a machine to a neighbor outside the mesh, optionally filtered by import and export route policy sets.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the external BGP peer.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"machine_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the machine that terminates the BGP session. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"machine_hostname": schema.StringAttribute{
				MarkdownDescription: "Hostname of the machine that terminates the BGP session.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the external BGP peer.",
				Required:            true,
			},
			"peer_ip": schema.StringAttribute{
				MarkdownDescription: "IP address of the BGP neighbor.",
				Required:            true,
			},
			"peer_asn": schema.Int64Attribute{
				MarkdownDescription: "AS number of the BGP neighbor.",
				Required:            true,
			},
			"local_ip": schema.StringAttribute{
				MarkdownDescription: "Local IP address to source the BGP session from. When unset, the machine picks one.",
				Optional:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the BGP session is enabled (default `true`).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "BGP MD5 authentication password. Sensitive — stored in state from your configuration and never read back from the API, so changes made outside Terraform are not detected and the value is not populated on import.",
				Optional:            true,
				Sensitive:           true,
			},
			"import_policy_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the route policy set applied to routes received from this peer.",
				Optional:            true,
			},
			"export_policy_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the route policy set applied to routes advertised to this peer.",
				Optional:            true,
			},
			"import_policy_name": schema.StringAttribute{
				MarkdownDescription: "Name of the import route policy set, if one is attached.",
				Computed:            true,
			},
			"export_policy_name": schema.StringAttribute{
				MarkdownDescription: "Name of the export route policy set, if one is attached.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the external BGP peer was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the external BGP peer was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *ExternalBGPPeerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ExternalBGPPeerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ExternalBGPPeerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peer, httpResp, err := r.client.ExternalBGPPeersAPI.
		ControlplaneApiCreateExternalBgpPeer(ctx).
		CreateExternalBGPPeerSchema(*buildCreateExternalBGPPeerRequest(&data)).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 300 {
			resp.Diagnostics.AddError(
				"Error Creating External BGP Peer",
				fmt.Sprintf("Could not create external BGP peer, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		// 2xx with deserialization error — parse raw body manually
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			var raw externalBGPPeerRaw
			if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating External BGP Peer",
					fmt.Sprintf("Could not parse external BGP peer response: %s", jsonErr),
				)
				return
			}
			mapRawToExternalBGPPeerModel(&raw, &data)
			tflog.Trace(ctx, "created external BGP peer")
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating External BGP Peer",
			fmt.Sprintf("Could not create external BGP peer, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	mapExternalBGPPeerToModel(peer, &data)

	tflog.Trace(ctx, "created external BGP peer")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalBGPPeerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ExternalBGPPeerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peer, httpResp, err := r.client.ExternalBGPPeersAPI.
		ControlplaneApiGetExternalBgpPeer(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw externalBGPPeerRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToExternalBGPPeerModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading External BGP Peer",
			fmt.Sprintf("Could not read external BGP peer %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapExternalBGPPeerToModel(peer, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalBGPPeerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ExternalBGPPeerResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peer, httpResp, err := r.client.ExternalBGPPeersAPI.
		ControlplaneApiUpdateExternalBgpPeer(ctx, data.UUID.ValueString()).
		UpdateExternalBGPPeerSchema(*buildUpdateExternalBGPPeerRequest(&data)).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw externalBGPPeerRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToExternalBGPPeerModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Updating External BGP Peer",
			fmt.Sprintf("Could not update external BGP peer %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapExternalBGPPeerToModel(peer, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ExternalBGPPeerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ExternalBGPPeerResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.ExternalBGPPeersAPI.
		ControlplaneApiDeleteExternalBgpPeer(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "external BGP peer already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting External BGP Peer",
			fmt.Sprintf("Could not delete external BGP peer %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted external BGP peer")
}

func (r *ExternalBGPPeerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// buildCreateExternalBGPPeerRequest builds the create request by setting fields
// by name — the generated constructor's positional order changes on regeneration.
func buildCreateExternalBGPPeerRequest(data *ExternalBGPPeerResourceModel) *durantic.CreateExternalBGPPeerSchema {
	createReq := durantic.NewCreateExternalBGPPeerSchemaWithDefaults()
	createReq.SetMachineUuid(data.MachineUUID.ValueString())
	createReq.SetName(data.Name.ValueString())
	createReq.SetPeerIp(data.PeerIP.ValueString())
	createReq.SetPeerAsn(int32(data.PeerASN.ValueInt64()))
	createReq.SetEnabled(data.Enabled.ValueBool())

	if !data.LocalIP.IsNull() {
		createReq.SetLocalIp(data.LocalIP.ValueString())
	}
	if !data.Password.IsNull() {
		createReq.SetPassword(data.Password.ValueString())
	}
	if !data.ImportPolicyUUID.IsNull() {
		createReq.SetImportPolicyUuid(data.ImportPolicyUUID.ValueString())
	}
	if !data.ExportPolicyUUID.IsNull() {
		createReq.SetExportPolicyUuid(data.ExportPolicyUUID.ValueString())
	}

	return createReq
}

// buildUpdateExternalBGPPeerRequest builds a full-state update. The optional
// fields are sent as an explicit JSON null when unset in the plan: omitting
// them would leave a previously configured value in place on the server, so
// removing `local_ip`, `password` or a policy from the configuration would
// silently do nothing.
func buildUpdateExternalBGPPeerRequest(data *ExternalBGPPeerResourceModel) *durantic.UpdateExternalBGPPeerSchema {
	updateReq := durantic.NewUpdateExternalBGPPeerSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetPeerIp(data.PeerIP.ValueString())
	updateReq.SetPeerAsn(int32(data.PeerASN.ValueInt64()))
	updateReq.SetEnabled(data.Enabled.ValueBool())

	if data.LocalIP.IsNull() {
		updateReq.SetLocalIpNil()
	} else {
		updateReq.SetLocalIp(data.LocalIP.ValueString())
	}
	if data.Password.IsNull() {
		updateReq.SetPasswordNil()
	} else {
		updateReq.SetPassword(data.Password.ValueString())
	}
	if data.ImportPolicyUUID.IsNull() {
		updateReq.SetImportPolicyUuidNil()
	} else {
		updateReq.SetImportPolicyUuid(data.ImportPolicyUUID.ValueString())
	}
	if data.ExportPolicyUUID.IsNull() {
		updateReq.SetExportPolicyUuidNil()
	} else {
		updateReq.SetExportPolicyUuid(data.ExportPolicyUUID.ValueString())
	}

	return updateReq
}

// mapExternalBGPPeerToModel maps the API schema onto the Terraform model.
// Password is deliberately left untouched: it is write-only from Terraform's
// point of view (see the attribute description).
func mapExternalBGPPeerToModel(peer *durantic.ExternalBGPPeerSchema, model *ExternalBGPPeerResourceModel) {
	model.UUID = types.StringValue(peer.GetUuid())
	model.MachineUUID = types.StringValue(peer.GetMachineUuid())
	model.MachineHostname = types.StringValue(peer.GetMachineHostname())
	model.Name = types.StringValue(peer.GetName())
	model.PeerIP = types.StringValue(peer.GetPeerIp())
	model.PeerASN = types.Int64Value(int64(peer.GetPeerAsn()))
	model.Enabled = types.BoolValue(peer.GetEnabled())

	// An empty string means "not set"; keep it null so an unset optional
	// attribute doesn't turn into "" after apply.
	if localIP, ok := peer.GetLocalIpOk(); ok && localIP != nil && *localIP != "" {
		model.LocalIP = types.StringValue(*localIP)
	} else {
		model.LocalIP = types.StringNull()
	}

	if policy, ok := peer.GetImportPolicyOk(); ok && policy != nil {
		model.ImportPolicyUUID = types.StringValue(policy.GetUuid())
		model.ImportPolicyName = types.StringValue(policy.GetName())
	} else {
		model.ImportPolicyUUID = types.StringNull()
		model.ImportPolicyName = types.StringNull()
	}

	if policy, ok := peer.GetExportPolicyOk(); ok && policy != nil {
		model.ExportPolicyUUID = types.StringValue(policy.GetUuid())
		model.ExportPolicyName = types.StringValue(policy.GetName())
	} else {
		model.ExportPolicyUUID = types.StringNull()
		model.ExportPolicyName = types.StringNull()
	}

	model.CreatedAt = types.StringValue(peer.GetCreatedAt())
	model.UpdatedAt = types.StringValue(peer.GetUpdatedAt())
}

// externalBGPPeerRaw is a tolerant JSON struct for parsing API responses that
// the generated client could not decode.
type externalBGPPeerRaw struct {
	UUID            string  `json:"uuid"`
	MachineUUID     string  `json:"machine_uuid"`
	MachineHostname string  `json:"machine_hostname"`
	Name            string  `json:"name"`
	PeerIP          string  `json:"peer_ip"`
	PeerASN         int32   `json:"peer_asn"`
	LocalIP         *string `json:"local_ip"`
	Enabled         bool    `json:"enabled"`
	ImportPolicy    *struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"import_policy"`
	ExportPolicy *struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"export_policy"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func mapRawToExternalBGPPeerModel(raw *externalBGPPeerRaw, model *ExternalBGPPeerResourceModel) {
	model.UUID = types.StringValue(raw.UUID)
	model.MachineUUID = types.StringValue(raw.MachineUUID)
	model.MachineHostname = types.StringValue(raw.MachineHostname)
	model.Name = types.StringValue(raw.Name)
	model.PeerIP = types.StringValue(raw.PeerIP)
	model.PeerASN = types.Int64Value(int64(raw.PeerASN))
	model.Enabled = types.BoolValue(raw.Enabled)

	if raw.LocalIP != nil && *raw.LocalIP != "" {
		model.LocalIP = types.StringValue(*raw.LocalIP)
	} else {
		model.LocalIP = types.StringNull()
	}

	if raw.ImportPolicy != nil {
		model.ImportPolicyUUID = types.StringValue(raw.ImportPolicy.UUID)
		model.ImportPolicyName = types.StringValue(raw.ImportPolicy.Name)
	} else {
		model.ImportPolicyUUID = types.StringNull()
		model.ImportPolicyName = types.StringNull()
	}

	if raw.ExportPolicy != nil {
		model.ExportPolicyUUID = types.StringValue(raw.ExportPolicy.UUID)
		model.ExportPolicyName = types.StringValue(raw.ExportPolicy.Name)
	} else {
		model.ExportPolicyUUID = types.StringNull()
		model.ExportPolicyName = types.StringNull()
	}

	model.CreatedAt = types.StringValue(raw.CreatedAt)
	model.UpdatedAt = types.StringValue(raw.UpdatedAt)
}
