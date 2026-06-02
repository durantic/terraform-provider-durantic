// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	defaultAPIEndpoint        = "https://api.stage.durantic.dev"
	endpointEnvName           = "DURANTIC_ENDPOINT"
	apiTokenEnvName           = "DURANTIC_API_TOKEN"
	insecureSkipVerifyEnvName = "DURANTIC_INSECURE_SKIP_VERIFY"
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
	Endpoint           types.String `tfsdk:"endpoint"`
	ApiToken           types.String `tfsdk:"api_token"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
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
			"insecure_skip_verify": schema.BoolAttribute{
				MarkdownDescription: "Skip TLS certificate verification. Can also be set via DURANTIC_INSECURE_SKIP_VERIFY environment variable. Defaults to false. **WARNING:** This should only be used in development/testing environments.",
				Optional:            true,
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

	// Read endpoint from env var, config, or use default
	endpoint := stringCoalesce(os.Getenv(endpointEnvName), data.Endpoint.ValueString())
	if endpoint == "" {
		endpoint = defaultAPIEndpoint
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
	tflog.Debug(ctx, fmt.Sprintf("Using Durantic API endpoint: %s", parsedURL.String()))

	// Read API token from env var or config
	apiToken := stringCoalesce(os.Getenv(apiTokenEnvName), data.ApiToken.ValueString())

	// Validate that API token is provided
	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			fmt.Sprintf("API token is required. Set it via the api_token provider attribute or the %s environment variable.", apiTokenEnvName),
		)
		return
	}

	// Read insecure_skip_verify from config or env var
	insecureSkipVerify := data.InsecureSkipVerify.ValueBool()

	if v := os.Getenv(insecureSkipVerifyEnvName); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Invalid %s Value", insecureSkipVerifyEnvName),
				fmt.Sprintf("Could not parse %s as boolean: %s", insecureSkipVerifyEnvName, err),
			)
			return
		}
		insecureSkipVerify = b
	}

	// Create Durantic API client configuration
	cfg := durantic.NewConfiguration()
	cfg.Scheme = parsedURL.Scheme
	cfg.Host = parsedURL.Host

	// Configure custom HTTP client if TLS verification should be skipped
	if insecureSkipVerify {
		cfg.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

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
		NewMachineRoleResource,
		NewMachineConfigResource,
		NewMachineDeploymentResource,
		NewMeshNetworkResource,
		NewRouteResource,
		NewVIPResource,
		NewRegistryCredentialResource,
		NewSecretsBackendResource,
		NewSecretResource,
		NewVariableResource,
		NewRoutePolicySetResource,
	}
}

func (p *DuranticProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *DuranticProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewImagesDataSource,
		NewImageDataSource,
		NewMachineDataSource,
	}
}

func (p *DuranticProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &DuranticProvider{
			version: version,
		}
	}
}

// stringCoalesce returns the first non-empty string from the provided values or an empty string if all are empty.
func stringCoalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// extractAPIError extracts error details from API response.
func extractAPIError(httpResp *http.Response, err error) string {
	if err != nil {
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			if body := string(apiErr.Body()); body != "" {
				return fmt.Sprintf("%s: %s", httpResp.Status, body)
			}
			return fmt.Sprintf("%s: %s", httpResp.Status, apiErr.Error())
		}
		return err.Error()
	}
	return "unknown error"
}

// // stringFromNullable maps a nullable string from the API to a Terraform string.
// func stringFromNullable(ns durantic.NullableString) types.String {
// 	if ns.IsSet() && ns.Get() != nil {
// 		return types.StringValue(*ns.Get())
// 	}
// 	return types.StringNull()
// }

// // boolFromPointer maps a bool pointer from the API to a Terraform bool.
// func boolFromPointer(b *bool) types.Bool {
// 	if b != nil {
// 		return types.BoolValue(*b)
// 	}
// 	return types.BoolValue(false)
// }

// // listFromSlice maps a string slice from the API to a Terraform list.
// func listFromSlice(ctx context.Context, diags *diag.Diagnostics, slice []string) types.List {
// 	if len(slice) > 0 {
// 		list, d := types.ListValueFrom(ctx, types.StringType, slice)
// 		diags.Append(d...)
// 		return list
// 	}
// 	return types.ListNull(types.StringType)
// }
