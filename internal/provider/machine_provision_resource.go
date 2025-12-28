// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &MachineProvisionResource{}

func NewMachineProvisionResource() resource.Resource {
	return &MachineProvisionResource{}
}

// MachineProvisionResource defines the resource implementation.
type MachineProvisionResource struct {
	client *durantic.APIClient
}

// MachineProvisionResourceModel describes the resource data model.
type MachineProvisionResourceModel struct {
	ID              types.String `tfsdk:"id"`
	MachineUuid     types.String `tfsdk:"machine_uuid"`
	Mode            types.String `tfsdk:"mode"`
	Triggers        types.Map    `tfsdk:"triggers"`
	LastProvisioned types.String `tfsdk:"last_provisioned"`
	Status          types.String `tfsdk:"status"`
}

func (r *MachineProvisionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine_provision"
}

func (r *MachineProvisionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Triggers provisioning actions on a machine. This is an action-based resource - any change to mode, machine_uuid, or triggers will trigger a new provisioning action.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for this provisioning action (format: {machine_uuid}:{mode}:{timestamp})",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"machine_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the machine to provision",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mode": schema.StringAttribute{
				MarkdownDescription: "Provisioning mode: 'rebuild' (install OS), 'discover' (discovery mode), or 'clear' (clear provisioning flag)",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("rebuild", "discover", "clear"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"triggers": schema.MapAttribute{
				MarkdownDescription: "Arbitrary map of values that, when changed, will trigger a new provisioning action",
				Optional:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"last_provisioned": schema.StringAttribute{
				MarkdownDescription: "Timestamp when provisioning was last triggered (RFC3339 format)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Status message from the last provisioning action",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *MachineProvisionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *MachineProvisionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MachineProvisionResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Build provision request
	provisionReq := durantic.NewMachineProvisionSchema()
	mode := data.Mode.ValueString()
	provisionReq.SetMode(mode)

	// Call API to trigger provisioning
	result, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiProvisionMachine(ctx, data.MachineUuid.ValueString()).
		MachineProvisionSchema(*provisionReq).
		Execute()

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Provisioning Machine",
			fmt.Sprintf("Could not provision machine %s with mode %s: %s",
				data.MachineUuid.ValueString(), mode, extractAPIError(httpResp, err)),
		)
		return
	}

	// Generate ID and set metadata
	now := time.Now()
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%d",
		data.MachineUuid.ValueString(), mode, now.Unix()))
	data.LastProvisioned = types.StringValue(now.Format(time.RFC3339))

	// Extract status from API response
	if result != nil {
		if statusMsg, ok := result["message"].(string); ok {
			data.Status = types.StringValue(statusMsg)
		} else {
			data.Status = types.StringValue(fmt.Sprintf("Provisioning triggered successfully (mode: %s)", mode))
		}
	} else {
		data.Status = types.StringValue(fmt.Sprintf("Provisioning triggered successfully (mode: %s)", mode))
	}

	tflog.Trace(ctx, fmt.Sprintf("triggered provisioning for machine %s with mode %s",
		data.MachineUuid.ValueString(), mode))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineProvisionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MachineProvisionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Verify machine still exists
	_, httpResp, err := r.client.MachinesAPI.
		ProvisioningApiGetMachine(ctx, data.MachineUuid.ValueString()).
		Execute()

	if err != nil {
		// Handle 404 - machine deleted outside Terraform
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error Reading Machine",
			fmt.Sprintf("Could not verify machine %s: %s",
				data.MachineUuid.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	// Note: We don't update any fields on read for action-based resources
	// The state reflects the last provisioning action, not current machine state

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *MachineProvisionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This should never be called because all attributes have RequiresReplace
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"Machine provisioning updates are not supported. All changes require replacement (re-provisioning).",
	)
}

func (r *MachineProvisionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MachineProvisionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Provisioning is an action, not persistent state
	// No API call needed - just remove from state
	tflog.Trace(ctx, fmt.Sprintf("removed provision action for machine %s from state",
		data.MachineUuid.ValueString()))
}

func (r *MachineProvisionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import is not recommended for action-based resources
	// But we'll support it by treating the import ID as machine_uuid
	resp.Diagnostics.AddWarning(
		"Import Not Recommended",
		"Importing machine provision resources is not recommended as they represent actions, not state. "+
			"The resource will be imported with mode='rebuild' by default.",
	)

	// Set machine_uuid from import ID
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("machine_uuid"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("mode"), "rebuild")...)

	// Set computed fields to placeholder values
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"),
		fmt.Sprintf("%s:rebuild:imported", req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("last_provisioned"),
		time.Now().Format(time.RFC3339))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status"), "imported")...)
}
