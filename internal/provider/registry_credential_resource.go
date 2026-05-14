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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &RegistryCredentialResource{}
var _ resource.ResourceWithImportState = &RegistryCredentialResource{}

func NewRegistryCredentialResource() resource.Resource {
	return &RegistryCredentialResource{}
}

type RegistryCredentialResource struct {
	client *durantic.APIClient
}

type RegistryCredentialResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	RegistryURL types.String `tfsdk:"registry_url"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	Description types.String `tfsdk:"description"`
	ImageCount  types.Int64  `tfsdk:"image_count"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *RegistryCredentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_registry_credential"
}

func (r *RegistryCredentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Durantic registry credential — authentication details for a container image registry.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the registry credential.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the registry credential.",
				Required:            true,
			},
			"registry_url": schema.StringAttribute{
				MarkdownDescription: "URL of the container registry (e.g. `registry.example.com`).",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Registry username.",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Registry password or access token. Sensitive — stored in state but never returned by the API.",
				Required:            true,
				Sensitive:           true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the registry credential.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"image_count": schema.Int64Attribute{
				MarkdownDescription: "Number of images associated with this credential.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the registry credential was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the registry credential was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *RegistryCredentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RegistryCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RegistryCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := durantic.NewCreateRegistryCredentialSchema(
		data.Name.ValueString(),
		data.RegistryURL.ValueString(),
		data.Username.ValueString(),
		data.Password.ValueString(),
	)
	createReq.SetDescription(data.Description.ValueString())

	cred, httpResp, err := r.client.RegistryCredentialsAPI.
		ProvisioningApiCreateRegistryCredential(ctx).
		CreateRegistryCredentialSchema(*createReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 300 {
			resp.Diagnostics.AddError(
				"Error Creating Registry Credential",
				fmt.Sprintf("Could not create registry credential, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			var raw registryCredentialRaw
			if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating Registry Credential",
					fmt.Sprintf("Could not parse registry credential response: %s", jsonErr),
				)
				return
			}
			password := data.Password
			mapRawToRegistryCredentialModel(&raw, &data)
			data.Password = password
			tflog.Trace(ctx, "created registry credential")
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating Registry Credential",
			fmt.Sprintf("Could not create registry credential, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	// Preserve password — API never returns it.
	password := data.Password
	mapRegistryCredentialToModel(cred, &data)
	data.Password = password

	tflog.Trace(ctx, "created registry credential")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegistryCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RegistryCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cred, httpResp, err := r.client.RegistryCredentialsAPI.
		ProvisioningApiGetRegistryCredential(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw registryCredentialRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					password := data.Password
					mapRawToRegistryCredentialModel(&raw, &data)
					data.Password = password
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading Registry Credential",
			fmt.Sprintf("Could not read registry credential %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Preserve stored password — API never returns it.
	password := data.Password
	mapRegistryCredentialToModel(cred, &data)
	data.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegistryCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RegistryCredentialResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := durantic.NewUpdateRegistryCredentialSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetRegistryUrl(data.RegistryURL.ValueString())
	updateReq.SetUsername(data.Username.ValueString())
	updateReq.SetPassword(data.Password.ValueString())
	updateReq.SetDescription(data.Description.ValueString())

	cred, httpResp, err := r.client.RegistryCredentialsAPI.
		ProvisioningApiUpdateRegistryCredential(ctx, data.UUID.ValueString()).
		UpdateRegistryCredentialSchema(*updateReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw registryCredentialRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					password := data.Password
					mapRawToRegistryCredentialModel(&raw, &data)
					data.Password = password
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Updating Registry Credential",
			fmt.Sprintf("Could not update registry credential %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Preserve password — API never returns it.
	password := data.Password
	mapRegistryCredentialToModel(cred, &data)
	data.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RegistryCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RegistryCredentialResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.RegistryCredentialsAPI.
		ProvisioningApiDeleteRegistryCredential(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "registry credential already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Registry Credential",
			fmt.Sprintf("Could not delete registry credential %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted registry credential")
}

func (r *RegistryCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

type registryCredentialRaw struct {
	UUID        string  `json:"uuid"`
	Name        string  `json:"name"`
	RegistryURL string  `json:"registry_url"`
	Username    string  `json:"username"`
	Description *string `json:"description,omitempty"`
	ImageCount  *int32  `json:"image_count,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func mapRawToRegistryCredentialModel(raw *registryCredentialRaw, model *RegistryCredentialResourceModel) {
	model.UUID = types.StringValue(raw.UUID)
	model.Name = types.StringValue(raw.Name)
	model.RegistryURL = types.StringValue(raw.RegistryURL)
	model.Username = types.StringValue(raw.Username)

	if raw.Description != nil {
		model.Description = types.StringValue(*raw.Description)
	} else {
		model.Description = types.StringValue("")
	}

	if raw.ImageCount != nil {
		model.ImageCount = types.Int64Value(int64(*raw.ImageCount))
	} else {
		model.ImageCount = types.Int64Value(0)
	}

	model.CreatedAt = types.StringValue(raw.CreatedAt)
	model.UpdatedAt = types.StringValue(raw.UpdatedAt)
}

func mapRegistryCredentialToModel(c *durantic.RegistryCredentialSchema, model *RegistryCredentialResourceModel) {
	model.UUID = types.StringValue(c.GetUuid())
	model.Name = types.StringValue(c.GetName())
	model.RegistryURL = types.StringValue(c.GetRegistryUrl())
	model.Username = types.StringValue(c.GetUsername())

	if desc, ok := c.GetDescriptionOk(); ok && desc != nil {
		model.Description = types.StringValue(*desc)
	} else {
		model.Description = types.StringValue("")
	}

	if count, ok := c.GetImageCountOk(); ok && count != nil {
		model.ImageCount = types.Int64Value(int64(*count))
	} else {
		model.ImageCount = types.Int64Value(0)
	}

	model.CreatedAt = types.StringValue(c.GetCreatedAt())
	model.UpdatedAt = types.StringValue(c.GetUpdatedAt())
}
