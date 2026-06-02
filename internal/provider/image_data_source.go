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
)

var _ datasource.DataSource = &ImageDataSource{}

func NewImageDataSource() datasource.DataSource {
	return &ImageDataSource{}
}

type ImageDataSource struct {
	client *durantic.APIClient
}

type ImageLookupDataSourceModel struct {
	UUID                   types.String `tfsdk:"uuid"`
	Name                   types.String `tfsdk:"name"`
	DockerImageURL         types.String `tfsdk:"docker_image_url"`
	RegistryCredentialUUID types.String `tfsdk:"registry_credential_uuid"`
	RegistryCredentialName types.String `tfsdk:"registry_credential_name"`
	IsOfficial             types.Bool   `tfsdk:"is_official"`
	Description            types.String `tfsdk:"description"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

func (d *ImageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_image"
}

func (d *ImageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a single Durantic image by UUID, name, or Docker image URL.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the image. Set exactly one of `uuid`, `name`, or `docker_image_url`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the image. Set exactly one of `uuid`, `name`, or `docker_image_url`.",
				Optional:            true,
				Computed:            true,
			},
			"docker_image_url": schema.StringAttribute{
				MarkdownDescription: "Docker image URL. Set exactly one of `uuid`, `name`, or `docker_image_url`.",
				Optional:            true,
				Computed:            true,
			},
			"registry_credential_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the registry credential associated with this image.",
				Computed:            true,
			},
			"registry_credential_name": schema.StringAttribute{
				MarkdownDescription: "Name of the registry credential associated with this image.",
				Computed:            true,
			},
			"is_official": schema.BoolAttribute{
				MarkdownDescription: "Whether this is an official image.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the image.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the image was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the image was last updated.",
				Computed:            true,
			},
		},
	}
}

func (d *ImageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ImageLookupDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	selectorCount := countSetStrings(data.UUID, data.Name, data.DockerImageURL)
	if selectorCount != 1 {
		resp.Diagnostics.AddError(
			"Invalid Image Lookup",
			"Exactly one of uuid, name, or docker_image_url must be set.",
		)
		return
	}

	if isKnownString(data.UUID) {
		image, httpResp, err := d.client.ImagesAPI.
			ProvisioningApiGetImage(ctx, data.UUID.ValueString()).
			Execute()
		if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
			resp.Diagnostics.AddError(
				"Error Reading Image",
				fmt.Sprintf("Could not read image %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
			)
			return
		}
		mapImageLookupToModel(image, &data)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	images, httpResp, err := d.client.ImagesAPI.ProvisioningApiListImages(ctx).Execute()
	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		resp.Diagnostics.AddError(
			"Error Listing Images",
			fmt.Sprintf("Could not list images: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	matches := make([]durantic.ImageSchema, 0, 1)
	for _, image := range images {
		switch {
		case isKnownString(data.Name) && image.GetName() == data.Name.ValueString():
			matches = append(matches, image)
		case isKnownString(data.DockerImageURL) && image.GetDockerImageUrl() == data.DockerImageURL.ValueString():
			matches = append(matches, image)
		}
	}

	if len(matches) == 0 {
		resp.Diagnostics.AddError(
			"No Image Found",
			"Could not find an image matching the configured selector.",
		)
		return
	}

	if len(matches) > 1 {
		resp.Diagnostics.AddError(
			"Multiple Images Found",
			fmt.Sprintf("Found %d images matching the configured selector. Use uuid for an unambiguous lookup.", len(matches)),
		)
		return
	}

	mapImageLookupToModel(&matches[0], &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func mapImageLookupToModel(img *durantic.ImageSchema, model *ImageLookupDataSourceModel) {
	imageModel := mapImageToDataSourceModel(img)
	model.UUID = imageModel.UUID
	model.Name = imageModel.Name
	model.DockerImageURL = imageModel.DockerImageURL
	model.RegistryCredentialUUID = imageModel.RegistryCredentialUUID
	model.RegistryCredentialName = imageModel.RegistryCredentialName
	model.IsOfficial = imageModel.IsOfficial
	model.Description = imageModel.Description
	model.CreatedAt = imageModel.CreatedAt
	model.UpdatedAt = imageModel.UpdatedAt
}
