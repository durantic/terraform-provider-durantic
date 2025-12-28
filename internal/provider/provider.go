// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	defaultAPIEndpoint = "https://api.stage.durantic.dev"
)

// Ensure DuranticProvider satisfies various provider interfaces.
var _ provider.Provider = &DuranticProvider{}
var _ provider.ProviderWithFunctions = &DuranticProvider{}
var _ provider.ProviderWithEphemeralResources = &DuranticProvider{}

// DuranticProvider defines the provider implementation.
type DuranticProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// DuranticProviderModel describes the provider data model.
type DuranticProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	ApiToken types.String `tfsdk:"api_token"`
}

func (p *DuranticProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "durantic"
	resp.Version = p.version
}

func (p *DuranticProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Durantic API endpoint URL. Can also be set via DURANTIC_ENDPOINT environment variable. Defaults to https://api.durantic.io",
				Optional:            true,
			},
			"api_token": schema.StringAttribute{
				MarkdownDescription: "API token for Durantic authentication. Can also be set via DURANTIC_API_TOKEN environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *DuranticProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data DuranticProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read endpoint from config, fallback to env var, or use default
	endpoint := defaultAPIEndpoint
	if !data.Endpoint.IsNull() && data.Endpoint.ValueString() != "" {
		endpoint = data.Endpoint.ValueString()
	} else if v := os.Getenv("DURANTIC_ENDPOINT"); v != "" {
		endpoint = v
	}

	// Read API token from config or env var
	apiToken := ""
	if !data.ApiToken.IsNull() && data.ApiToken.ValueString() != "" {
		apiToken = data.ApiToken.ValueString()
	} else if v := os.Getenv("DURANTIC_API_TOKEN"); v != "" {
		apiToken = v
	}

	// Validate that API token is provided
	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"API token is required. Set it via the api_token provider attribute or the DURANTIC_API_TOKEN environment variable.",
		)
		return
	}

	// Parse endpoint URL to extract scheme and host
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Endpoint URL",
			fmt.Sprintf("Could not parse endpoint URL: %s", err),
		)
		return
	}

	// Create Durantic API client configuration
	cfg := durantic.NewConfiguration()
	cfg.Scheme = parsedURL.Scheme
	cfg.Host = parsedURL.Host

	// Add authorization header
	cfg.DefaultHeader["Authorization"] = "Bearer " + apiToken

	// Create API client
	client := durantic.NewAPIClient(cfg)

	// Make client available to resources and data sources
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *DuranticProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMeshNetworkResource,
	}
}

func (p *DuranticProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewExampleEphemeralResource,
	}
}

func (p *DuranticProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func (p *DuranticProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		NewExampleFunction,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DuranticProvider{
			version: version,
		}
	}
}
