package provider

import (
	"context"
	"encoding/json"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &RouteResource{}
var _ resource.ResourceWithImportState = &RouteResource{}

func NewRouteResource() resource.Resource {
	return &RouteResource{}
}

type RouteResource struct {
	client *durantic.APIClient
}

type RouteResourceModel struct {
	UUID         types.String `tfsdk:"uuid"`
	Name         types.String `tfsdk:"name"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Prefixes     types.List   `tfsdk:"prefixes"`
	MachineUUIDs types.List   `tfsdk:"machine_uuids"`
	MachineCount types.Int64  `tfsdk:"machine_count"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (r *RouteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

func (r *RouteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Route resource for Durantic network routing",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the route",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the route",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the route is enabled",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"prefixes": schema.ListAttribute{
				MarkdownDescription: "Network prefixes for this route (at least one required)",
				Required:            true,
				ElementType:         types.StringType,
			},
			"machine_uuids": schema.ListAttribute{
				MarkdownDescription: "UUIDs of machines to associate with this route",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"machine_count": schema.Int64Attribute{
				MarkdownDescription: "Number of machines associated with this route",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the route was created",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the route was last updated",
				Computed:            true,
			},
		},
	}
}

func (r *RouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RouteResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var prefixes []string
	resp.Diagnostics.Append(data.Prefixes.ElementsAs(ctx, &prefixes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := durantic.NewCreateRouteSchema(data.Name.ValueString(), prefixes)
	createReq.SetEnabled(data.Enabled.ValueBool())

	if !data.MachineUUIDs.IsNull() && !data.MachineUUIDs.IsUnknown() {
		var machineUUIDs []string
		resp.Diagnostics.Append(data.MachineUUIDs.ElementsAs(ctx, &machineUUIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.SetMachineUuids(machineUUIDs)
	}

	route, httpResp, err := r.client.RoutesAPI.
		ControlplaneApiCreateRoute(ctx).
		CreateRouteSchema(*createReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 300 {
			resp.Diagnostics.AddError(
				"Error Creating Route",
				fmt.Sprintf("Could not create route, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			var raw routeRaw
			if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating Route",
					fmt.Sprintf("Could not parse route response: %s", jsonErr),
				)
				return
			}
			resp.Diagnostics.Append(mapRawToRouteModel(ctx, &raw, &data)...)
			if resp.Diagnostics.HasError() {
				return
			}
			tflog.Trace(ctx, "created route")
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating Route",
			fmt.Sprintf("Could not create route, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	resp.Diagnostics.Append(mapRouteToModel(ctx, route, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created route")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RouteResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	route, httpResp, err := r.client.RoutesAPI.
		ControlplaneApiGetRoute(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw routeRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					resp.Diagnostics.Append(mapRawToRouteModel(ctx, &raw, &data)...)
					if !resp.Diagnostics.HasError() {
						resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					}
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading Route",
			fmt.Sprintf("Could not read route %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	resp.Diagnostics.Append(mapRouteToModel(ctx, route, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RouteResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := durantic.NewUpdateRouteSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetEnabled(data.Enabled.ValueBool())

	var prefixes []string
	resp.Diagnostics.Append(data.Prefixes.ElementsAs(ctx, &prefixes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateReq.SetPrefixes(prefixes)

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

	route, httpResp, err := r.client.RoutesAPI.
		ControlplaneApiUpdateRoute(ctx, data.UUID.ValueString()).
		UpdateRouteSchema(*updateReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw routeRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					resp.Diagnostics.Append(mapRawToRouteModel(ctx, &raw, &data)...)
					if !resp.Diagnostics.HasError() {
						resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					}
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Updating Route",
			fmt.Sprintf("Could not update route %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	resp.Diagnostics.Append(mapRouteToModel(ctx, route, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RouteResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.RoutesAPI.
		ControlplaneApiDeleteRoute(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "route already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Route",
			fmt.Sprintf("Could not delete route %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted route")
}

func (r *RouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

type routeRaw struct {
	UUID         string   `json:"uuid"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	Prefixes     []string `json:"prefixes"`
	MachineCount int32    `json:"machine_count"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

func mapRawToRouteModel(ctx context.Context, raw *routeRaw, model *RouteResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.UUID = types.StringValue(raw.UUID)
	model.Name = types.StringValue(raw.Name)
	model.Enabled = types.BoolValue(raw.Enabled)
	model.MachineCount = types.Int64Value(int64(raw.MachineCount))
	model.CreatedAt = types.StringValue(raw.CreatedAt)
	model.UpdatedAt = types.StringValue(raw.UpdatedAt)

	prefixList, d := types.ListValueFrom(ctx, types.StringType, raw.Prefixes)
	diags.Append(d...)
	model.Prefixes = prefixList

	return diags
}

// mapRouteToModel maps an API RouteSchema to the Terraform model.
// machine_uuids is not updated here because the response returns machines as full objects
// rather than UUIDs; the write-only machine_uuids field retains its state value.
func mapRouteToModel(ctx context.Context, route *durantic.RouteSchema, model *RouteResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.UUID = types.StringValue(route.GetUuid())
	model.Name = types.StringValue(route.GetName())
	model.Enabled = types.BoolValue(route.GetEnabled())
	model.MachineCount = types.Int64Value(int64(route.GetMachineCount()))
	model.CreatedAt = types.StringValue(route.GetCreatedAt())
	model.UpdatedAt = types.StringValue(route.GetUpdatedAt())

	prefixList, d := types.ListValueFrom(ctx, types.StringType, route.GetPrefixes())
	diags.Append(d...)
	model.Prefixes = prefixList

	return diags
}
