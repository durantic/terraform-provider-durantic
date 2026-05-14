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

var _ resource.Resource = &SecretResource{}
var _ resource.ResourceWithImportState = &SecretResource{}

func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

type SecretResource struct {
	client *durantic.APIClient
}

type SecretResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *SecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

func (r *SecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Durantic account secret — a named sensitive value stored encrypted. The secret value is never returned by the API after creation; Terraform retains the last written value in state.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the secret.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the secret.",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Secret value. Sensitive — stored in state but never returned by the API.",
				Required:            true,
				Sensitive:           true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the secret.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the secret was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the secret was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *SecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := durantic.NewCreateAccountSecretSchema(data.Name.ValueString(), data.Value.ValueString())
	createReq.SetDescription(data.Description.ValueString())

	secret, httpResp, err := r.client.SecretsAPI.
		ControlplaneApiCreateAccountSecret(ctx).
		CreateAccountSecretSchema(*createReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 300 {
			resp.Diagnostics.AddError(
				"Error Creating Secret",
				fmt.Sprintf("Could not create secret, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			var raw accountSecretRaw
			if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating Secret",
					fmt.Sprintf("Could not parse secret response: %s", jsonErr),
				)
				return
			}
			value := data.Value
			mapRawToSecretModel(&raw, &data)
			data.Value = value
			tflog.Trace(ctx, "created secret")
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating Secret",
			fmt.Sprintf("Could not create secret, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	// Preserve value from plan — API never returns the secret value.
	value := data.Value
	mapSecretToModel(secret, &data)
	data.Value = value

	tflog.Trace(ctx, "created secret")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, httpResp, err := r.client.SecretsAPI.
		ControlplaneApiGetAccountSecret(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw accountSecretRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					value := data.Value
					mapRawToSecretModel(&raw, &data)
					data.Value = value
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading Secret",
			fmt.Sprintf("Could not read secret %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Preserve stored value — API never returns the secret value.
	value := data.Value
	mapSecretToModel(secret, &data)
	data.Value = value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := durantic.NewUpdateAccountSecretSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetValue(data.Value.ValueString())
	updateReq.SetDescription(data.Description.ValueString())

	secret, httpResp, err := r.client.SecretsAPI.
		ControlplaneApiUpdateAccountSecret(ctx, data.UUID.ValueString()).
		UpdateAccountSecretSchema(*updateReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw accountSecretRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					value := data.Value
					mapRawToSecretModel(&raw, &data)
					data.Value = value
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Updating Secret",
			fmt.Sprintf("Could not update secret %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Preserve value from plan — API never returns the secret value.
	value := data.Value
	mapSecretToModel(secret, &data)
	data.Value = value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.SecretsAPI.
		ControlplaneApiDeleteAccountSecret(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "secret already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Secret",
			fmt.Sprintf("Could not delete secret %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted secret")
}

func (r *SecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

type accountSecretRaw struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func mapRawToSecretModel(raw *accountSecretRaw, model *SecretResourceModel) {
	model.UUID = types.StringValue(raw.UUID)
	model.Name = types.StringValue(raw.Name)
	model.Description = types.StringValue(raw.Description)
	model.CreatedAt = types.StringValue(raw.CreatedAt)
	model.UpdatedAt = types.StringValue(raw.UpdatedAt)
}

func mapSecretToModel(s *durantic.AccountSecretSchema, model *SecretResourceModel) {
	model.UUID = types.StringValue(s.GetUuid())
	model.Name = types.StringValue(s.GetName())
	model.Description = types.StringValue(s.GetDescription())
	model.CreatedAt = types.StringValue(s.GetCreatedAt())
	model.UpdatedAt = types.StringValue(s.GetUpdatedAt())
}
