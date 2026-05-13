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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ImagesDataSource{}

func NewImagesDataSource() datasource.DataSource {
	return &ImagesDataSource{}
}

// ImagesDataSource defines the data source implementation.
type ImagesDataSource struct {
	client *durantic.APIClient
}

// ImagesDataSourceModel describes the data source data model.
type ImagesDataSourceModel struct {
	Images []ImageDataSourceModel `tfsdk:"images"`
}

// ImageDataSourceModel describes a single image.
type ImageDataSourceModel struct {
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

func (d *ImagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images"
}

func (d *ImagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all images available to the account (own and official).",

		Attributes: map[string]schema.Attribute{
			"images": schema.ListNestedAttribute{
				MarkdownDescription: "List of images.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the image.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the image.",
							Computed:            true,
						},
						"docker_image_url": schema.StringAttribute{
							MarkdownDescription: "Docker image URL.",
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
				},
			},
		},
	}
}

func (d *ImagesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ImagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ImagesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
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

	data.Images = make([]ImageDataSourceModel, len(images))
	for i, img := range images {
		data.Images[i] = mapImageToDataSourceModel(&img)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func mapImageToDataSourceModel(img *durantic.ImageSchema) ImageDataSourceModel {
	m := ImageDataSourceModel{
		UUID:           types.StringValue(img.GetUuid()),
		Name:           types.StringValue(img.GetName()),
		DockerImageURL: types.StringValue(img.GetDockerImageUrl()),
		CreatedAt:      types.StringValue(img.GetCreatedAt()),
		UpdatedAt:      types.StringValue(img.GetUpdatedAt()),
	}

	if ptr, ok := img.GetRegistryCredentialUuidOk(); ok && ptr != nil {
		m.RegistryCredentialUUID = types.StringValue(*ptr)
	} else {
		m.RegistryCredentialUUID = types.StringNull()
	}

	if ptr, ok := img.GetRegistryCredentialNameOk(); ok && ptr != nil {
		m.RegistryCredentialName = types.StringValue(*ptr)
	} else {
		m.RegistryCredentialName = types.StringNull()
	}

	if img.HasIsOfficial() {
		m.IsOfficial = types.BoolValue(img.GetIsOfficial())
	} else {
		m.IsOfficial = types.BoolValue(false)
	}

	if img.HasDescription() {
		m.Description = types.StringValue(img.GetDescription())
	} else {
		m.Description = types.StringNull()
	}

	return m
}
