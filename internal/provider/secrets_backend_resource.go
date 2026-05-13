// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &SecretsBackendResource{}
var _ resource.ResourceWithImportState = &SecretsBackendResource{}

func NewSecretsBackendResource() resource.Resource {
	return &SecretsBackendResource{}
}

type SecretsBackendResource struct {
	client *durantic.APIClient
}

type SecretsBackendResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	BackendType types.String `tfsdk:"backend_type"`
	URL         types.String `tfsdk:"url"`
	Config      types.Map    `tfsdk:"config"`
	CACert      types.String `tfsdk:"ca_cert"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *SecretsBackendResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secrets_backend"
}

func (r *SecretsBackendResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Durantic secrets backend — an external secrets store (Vault, HTTP) that supplies secrets to workloads.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the secrets backend.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the secrets backend.",
				Required:            true,
			},
			"backend_type": schema.StringAttribute{
				MarkdownDescription: "Type of the secrets backend. Supported values: `vault`, `http`.",
				Required:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "URL of the secrets backend endpoint.",
				Required:            true,
			},
			"config": schema.MapAttribute{
				MarkdownDescription: "Backend-specific string key/value configuration (e.g. auth paths, mount paths).",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"ca_cert": schema.StringAttribute{
				MarkdownDescription: "PEM-encoded CA certificate used to verify the backend's TLS certificate.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the secrets backend is enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the secrets backend was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the secrets backend was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *SecretsBackendResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SecretsBackendResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SecretsBackendResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backendType, err := durantic.NewSecretsBackendTypeFromValue(data.BackendType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Backend Type",
			fmt.Sprintf("Invalid backend_type %q: %s", data.BackendType.ValueString(), err),
		)
		return
	}

	createReq := durantic.NewCreateSecretsBackendSchema(data.Name.ValueString(), *backendType, data.URL.ValueString())
	createReq.SetCaCert(data.CACert.ValueString())
	createReq.SetEnabled(data.Enabled.ValueBool())

	if !data.Config.IsNull() && !data.Config.IsUnknown() {
		apiConfig, diags := configMapToAPI(ctx, data.Config)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.SetConfig(apiConfig)
	}

	backend, httpResp, apiErr := r.client.SecretsBackendsAPI.
		ControlplaneApiCreateSecretsBackend(ctx).
		CreateSecretsBackendSchema(*createReq).
		Execute()

	if apiErr != nil {
		resp.Diagnostics.AddError(
			"Error Creating Secrets Backend",
			fmt.Sprintf("Could not create secrets backend, unexpected error: %s", extractAPIError(httpResp, apiErr)),
		)
		return
	}

	resp.Diagnostics.Append(mapSecretsBackendToModel(ctx, backend, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created secrets backend")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretsBackendResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SecretsBackendResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backend, httpResp, err := r.client.SecretsBackendsAPI.
		ControlplaneApiGetSecretsBackend(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Secrets Backend",
			fmt.Sprintf("Could not read secrets backend %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	resp.Diagnostics.Append(mapSecretsBackendToModel(ctx, backend, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretsBackendResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SecretsBackendResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backendType, err := durantic.NewSecretsBackendTypeFromValue(data.BackendType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Backend Type",
			fmt.Sprintf("Invalid backend_type %q: %s", data.BackendType.ValueString(), err),
		)
		return
	}

	updateReq := durantic.NewUpdateSecretsBackendSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetBackendType(*backendType)
	updateReq.SetUrl(data.URL.ValueString())
	updateReq.SetCaCert(data.CACert.ValueString())
	updateReq.SetEnabled(data.Enabled.ValueBool())

	if !data.Config.IsNull() && !data.Config.IsUnknown() {
		apiConfig, diags := configMapToAPI(ctx, data.Config)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.SetConfig(apiConfig)
	} else {
		updateReq.SetConfig(map[string]interface{}{})
	}

	backend, httpResp, err := r.client.SecretsBackendsAPI.
		ControlplaneApiUpdateSecretsBackend(ctx, data.UUID.ValueString()).
		UpdateSecretsBackendSchema(*updateReq).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		resp.Diagnostics.AddError(
			"Error Updating Secrets Backend",
			fmt.Sprintf("Could not update secrets backend %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	resp.Diagnostics.Append(mapSecretsBackendToModel(ctx, backend, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SecretsBackendResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SecretsBackendResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.SecretsBackendsAPI.
		ControlplaneApiDeleteSecretsBackend(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "secrets backend already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Secrets Backend",
			fmt.Sprintf("Could not delete secrets backend %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted secrets backend")
}

func (r *SecretsBackendResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// configMapToAPI converts a Terraform types.Map (string values) to map[string]interface{} for the API.
func configMapToAPI(ctx context.Context, m types.Map) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics
	var stringMap map[string]string
	diags.Append(m.ElementsAs(ctx, &stringMap, false)...)
	result := make(map[string]interface{}, len(stringMap))
	for k, v := range stringMap {
		result[k] = v
	}
	return result, diags
}

func mapSecretsBackendToModel(ctx context.Context, b *durantic.SecretsBackendSchema, model *SecretsBackendResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.UUID = types.StringValue(b.GetUuid())
	model.Name = types.StringValue(b.GetName())
	model.BackendType = types.StringValue(b.GetBackendType())
	model.URL = types.StringValue(b.GetUrl())
	model.CACert = types.StringValue(b.GetCaCert())
	model.Enabled = types.BoolValue(b.GetEnabled())
	model.CreatedAt = types.StringValue(b.GetCreatedAt())
	model.UpdatedAt = types.StringValue(b.GetUpdatedAt())

	if len(b.GetConfig()) == 0 {
		model.Config = types.MapNull(types.StringType)
	} else {
		stringConfig := make(map[string]string, len(b.GetConfig()))
		for k, v := range b.GetConfig() {
			if s, ok := v.(string); ok {
				stringConfig[k] = s
			}
		}
		configMap, d := types.MapValueFrom(ctx, types.StringType, stringConfig)
		diags.Append(d...)
		model.Config = configMap
	}

	return diags
}
