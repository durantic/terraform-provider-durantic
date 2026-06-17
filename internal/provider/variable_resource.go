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

var _ resource.Resource = &VariableResource{}
var _ resource.ResourceWithImportState = &VariableResource{}

func NewVariableResource() resource.Resource {
	return &VariableResource{}
}

type VariableResource struct {
	client *durantic.APIClient
}

type VariableResourceModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *VariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable"
}

func (r *VariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Durantic account variable — a named key/value pair available to workloads.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the variable.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the variable.",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value of the variable.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the variable.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the variable was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the variable was last updated.",
				Computed:            true,
			},
		},
	}
}

func (r *VariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := durantic.NewCreateAccountVariableSchema(data.Name.ValueString(), data.Value.ValueString())
	createReq.SetDescription(data.Description.ValueString())

	variable, httpResp, err := r.client.VariablesAPI.
		ControlplaneApiCreateAccountVariable(ctx).
		CreateAccountVariableSchema(*createReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 300 {
			resp.Diagnostics.AddError(
				"Error Creating Variable",
				fmt.Sprintf("Could not create variable, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			var raw accountVariableRaw
			if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating Variable",
					fmt.Sprintf("Could not parse variable response: %s", jsonErr),
				)
				return
			}
			mapRawToVariableModel(&raw, &data)
			tflog.Trace(ctx, "created variable")
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Error Creating Variable",
			fmt.Sprintf("Could not create variable, unexpected error: %s", extractAPIError(httpResp, err)),
		)
		return
	}

	mapVariableToModel(variable, &data)

	tflog.Trace(ctx, "created variable")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variable, httpResp, err := r.client.VariablesAPI.
		ControlplaneApiGetAccountVariable(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw accountVariableRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToVariableModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading Variable",
			fmt.Sprintf("Could not read variable %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapVariableToModel(variable, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := durantic.NewUpdateAccountVariableSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetValue(data.Value.ValueString())
	updateReq.SetDescription(data.Description.ValueString())

	variable, httpResp, err := r.client.VariablesAPI.
		ControlplaneApiUpdateAccountVariable(ctx, data.UUID.ValueString()).
		UpdateAccountVariableSchema(*updateReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw accountVariableRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToVariableModel(&raw, &data)
					resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Updating Variable",
			fmt.Sprintf("Could not update variable %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapVariableToModel(variable, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.VariablesAPI.
		ControlplaneApiDeleteAccountVariable(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "variable already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Variable",
			fmt.Sprintf("Could not delete variable %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted variable")
}

func (r *VariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

type accountVariableRaw struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func mapRawToVariableModel(raw *accountVariableRaw, model *VariableResourceModel) {
	model.UUID = types.StringValue(raw.UUID)
	model.Name = types.StringValue(raw.Name)
	model.Value = types.StringValue(raw.Value)
	model.Description = types.StringValue(raw.Description)
	model.CreatedAt = types.StringValue(raw.CreatedAt)
	model.UpdatedAt = types.StringValue(raw.UpdatedAt)
}

func mapVariableToModel(v *durantic.AccountVariableSchema, model *VariableResourceModel) {
	model.UUID = types.StringValue(v.GetUuid())
	model.Name = types.StringValue(v.GetName())
	model.Value = types.StringValue(v.GetValue())
	model.Description = types.StringValue(v.GetDescription())
	model.CreatedAt = types.StringValue(v.GetCreatedAt())
	model.UpdatedAt = types.StringValue(v.GetUpdatedAt())
}
