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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ImageResource{}
var _ resource.ResourceWithImportState = &ImageResource{}

func NewImageResource() resource.Resource {
	return &ImageResource{}
}

type ImageResource struct {
	client *durantic.APIClient
}

type ImageResourceModel struct {
	UUID                   types.String `tfsdk:"uuid"`
	Name                   types.String `tfsdk:"name"`
	DockerImageURL         types.String `tfsdk:"docker_image_url"`
	Description            types.String `tfsdk:"description"`
	RegistryCredentialUUID types.String `tfsdk:"registry_credential_uuid"`
	RegistryCredentialName types.String `tfsdk:"registry_credential_name"`
	IsOfficial             types.Bool   `tfsdk:"is_official"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

func (r *ImageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (r *ImageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Durantic image — a container image that machine roles install onto machines. Official images are managed by Durantic and cannot be changed or deleted through this resource.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the image.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the image.",
				Required:            true,
			},
			"docker_image_url": schema.StringAttribute{
				MarkdownDescription: "Docker/OCI image URL (e.g. `registry.example.com/team/app:1.2.3`).",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the image.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"registry_credential_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the `durantic_registry_credential` used to pull the image. Leave unset for images that need no authentication.",
				Optional:            true,
			},
			"registry_credential_name": schema.StringAttribute{
				MarkdownDescription: "Name of the registry credential used to pull the image, if any.",
				Computed:            true,
			},
			"is_official": schema.BoolAttribute{
				MarkdownDescription: "Whether this is an official Durantic image.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the image was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the image was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *ImageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ImageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, httpResp, err := r.client.ImagesAPI.
		ProvisioningApiCreateImage(ctx).
		CreateImageSchema(*buildCreateImageRequest(&data)).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 300 {
			resp.Diagnostics.AddError(
				"Error Creating Image",
				fmt.Sprintf("Could not create image, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		// 2xx with deserialization error — parse raw body manually
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			var raw imageRaw
			if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating Image",
					fmt.Sprintf("Could not parse image response: %s", jsonErr),
				)
				return
			}
			mapRawToImageResourceModel(&raw, &data)
			tflog.Trace(ctx, "created image")
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating Image",
			fmt.Sprintf("Could not create image, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	mapImageToResourceModel(image, &data)

	tflog.Trace(ctx, "created image")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ImageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, httpResp, err := r.client.ImagesAPI.
		ProvisioningApiGetImage(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw imageRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToImageResourceModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading Image",
			fmt.Sprintf("Could not read image %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapImageToResourceModel(image, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ImageResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, httpResp, err := r.client.ImagesAPI.
		ProvisioningApiUpdateImage(ctx, data.UUID.ValueString()).
		UpdateImageSchema(*buildUpdateImageRequest(&data)).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw imageRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToImageResourceModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Updating Image",
			fmt.Sprintf("Could not update image %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapImageToResourceModel(image, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ImageResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.ImagesAPI.
		ProvisioningApiDeleteImage(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "image already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Image",
			fmt.Sprintf("Could not delete image %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted image")
}

func (r *ImageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// buildCreateImageRequest sets fields by name — the generated constructor's
// positional order changes on regeneration.
func buildCreateImageRequest(data *ImageResourceModel) *durantic.CreateImageSchema {
	createReq := durantic.NewCreateImageSchemaWithDefaults()
	createReq.SetName(data.Name.ValueString())
	createReq.SetDockerImageUrl(data.DockerImageURL.ValueString())
	createReq.SetDescription(data.Description.ValueString())

	if !data.RegistryCredentialUUID.IsNull() {
		createReq.SetRegistryCredentialUuid(data.RegistryCredentialUUID.ValueString())
	}

	return createReq
}

// buildUpdateImageRequest builds a full-state update. An unset
// registry_credential_uuid is sent as an explicit JSON null: omitting it would
// leave the previous credential attached when it is removed from configuration.
func buildUpdateImageRequest(data *ImageResourceModel) *durantic.UpdateImageSchema {
	updateReq := durantic.NewUpdateImageSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetDockerImageUrl(data.DockerImageURL.ValueString())
	updateReq.SetDescription(data.Description.ValueString())

	if data.RegistryCredentialUUID.IsNull() {
		updateReq.SetRegistryCredentialUuidNil()
	} else {
		updateReq.SetRegistryCredentialUuid(data.RegistryCredentialUUID.ValueString())
	}

	return updateReq
}

// mapImageToResourceModel and mapRawToImageResourceModel reuse the data source
// mappers (and its tolerant imageRaw struct); only the copy below is specific to
// the resource.
func mapImageToResourceModel(img *durantic.ImageSchema, model *ImageResourceModel) {
	applyImageDataSourceModel(mapImageToDataSourceModel(img), model)
}

func mapRawToImageResourceModel(raw *imageRaw, model *ImageResourceModel) {
	applyImageDataSourceModel(mapRawToImageDataSourceModel(raw), model)
}

// applyImageDataSourceModel copies a mapped image onto the resource model. The
// one difference is the description: the resource defaults it to "" so it must
// never be null.
func applyImageDataSourceModel(m ImageDataSourceModel, model *ImageResourceModel) {
	model.UUID = m.UUID
	model.Name = m.Name
	model.DockerImageURL = m.DockerImageURL
	model.RegistryCredentialUUID = m.RegistryCredentialUUID
	model.RegistryCredentialName = m.RegistryCredentialName
	model.IsOfficial = m.IsOfficial
	model.CreatedAt = m.CreatedAt
	model.UpdatedAt = m.UpdatedAt

	if m.Description.IsNull() {
		model.Description = types.StringValue("")
	} else {
		model.Description = m.Description
	}
}
