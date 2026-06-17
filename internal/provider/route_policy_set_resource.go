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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &RoutePolicySetResource{}
var _ resource.ResourceWithImportState = &RoutePolicySetResource{}
var _ resource.ResourceWithModifyPlan = &RoutePolicySetResource{}

func NewRoutePolicySetResource() resource.Resource {
	return &RoutePolicySetResource{}
}

type RoutePolicySetResource struct {
	client *durantic.APIClient
}

type RoutePolicySetResourceModel struct {
	UUID          types.String           `tfsdk:"uuid"`
	Name          types.String           `tfsdk:"name"`
	Description   types.String           `tfsdk:"description"`
	DefaultAction types.String           `tfsdk:"default_action"`
	LocalPref     types.Int64            `tfsdk:"local_pref"`
	AdvancedMode  types.Bool             `tfsdk:"advanced_mode"`
	Rules         []RoutePolicyRuleModel `tfsdk:"rules"`
	CreatedAt     types.String           `tfsdk:"created_at"`
	UpdatedAt     types.String           `tfsdk:"updated_at"`
}

type RoutePolicyRuleModel struct {
	UUID              types.String `tfsdk:"uuid"`
	Sequence          types.Int64  `tfsdk:"sequence"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	Description       types.String `tfsdk:"description"`
	MatchType         types.String `tfsdk:"match_type"`
	MatchPrefixes     types.List   `tfsdk:"match_prefixes"`
	MatchPrefixLenMin types.Int64  `tfsdk:"match_prefix_len_min"`
	MatchPrefixLenMax types.Int64  `tfsdk:"match_prefix_len_max"`
	MatchCommunities  types.List   `tfsdk:"match_communities"`
	MatchAsPathRegex  types.String `tfsdk:"match_as_path_regex"`
	Action            types.String `tfsdk:"action"`
	SetLocalPref      types.Int64  `tfsdk:"set_local_pref"`
	SetMed            types.Int64  `tfsdk:"set_med"`
	SetCommunities    types.List   `tfsdk:"set_communities"`
	AddCommunities    types.List   `tfsdk:"add_communities"`
	RemoveCommunities types.List   `tfsdk:"remove_communities"`
	PrependAsCount    types.Int64  `tfsdk:"prepend_as_count"`
}

func (r *RoutePolicySetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route_policy_set"
}

func (r *RoutePolicySetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	ruleAttrs := map[string]schema.Attribute{
		"uuid": schema.StringAttribute{
			MarkdownDescription: "Unique identifier for the rule.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"sequence": schema.Int64Attribute{
			MarkdownDescription: "Rule evaluation order. Lower numbers are evaluated first.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(100),
		},
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether this rule is active.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Human-readable description of the rule.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"match_type": schema.StringAttribute{
			MarkdownDescription: "How to match routes: `any`, `prefix_list`, `community`, or `as_path`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("any"),
		},
		"match_prefixes": schema.ListAttribute{
			MarkdownDescription: "Network prefixes to match.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"match_prefix_len_min": schema.Int64Attribute{
			MarkdownDescription: "Minimum prefix length to match.",
			Optional:            true,
		},
		"match_prefix_len_max": schema.Int64Attribute{
			MarkdownDescription: "Maximum prefix length to match.",
			Optional:            true,
		},
		"match_communities": schema.ListAttribute{
			MarkdownDescription: "BGP communities to match.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"match_as_path_regex": schema.StringAttribute{
			MarkdownDescription: "Regular expression to match against the AS path.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(""),
		},
		"action": schema.StringAttribute{
			MarkdownDescription: "Action to take when the rule matches: `accept` or `reject`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("accept"),
		},
		"set_local_pref": schema.Int64Attribute{
			MarkdownDescription: "Set the LOCAL_PREF BGP attribute on matched routes.",
			Optional:            true,
		},
		"set_med": schema.Int64Attribute{
			MarkdownDescription: "Set the MED (MULTI_EXIT_DISC) BGP attribute on matched routes.",
			Optional:            true,
		},
		"set_communities": schema.ListAttribute{
			MarkdownDescription: "Replace BGP communities on matched routes.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"add_communities": schema.ListAttribute{
			MarkdownDescription: "Add BGP communities to matched routes.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"remove_communities": schema.ListAttribute{
			MarkdownDescription: "Remove BGP communities from matched routes.",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"prepend_as_count": schema.Int64Attribute{
			MarkdownDescription: "Number of times to prepend the local AS to the AS path.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(0),
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Durantic route policy set — a named ordered list of BGP route policy rules applied during route exchange.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the route policy set.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the route policy set.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description of the route policy set.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"default_action": schema.StringAttribute{
				MarkdownDescription: "Action to take when no rule matches: `accept` or `reject`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("accept"),
			},
			"local_pref": schema.Int64Attribute{
				MarkdownDescription: "Default LOCAL_PREF value applied to all routes in this policy set.",
				Optional:            true,
			},
			"advanced_mode": schema.BoolAttribute{
				MarkdownDescription: "Enable advanced mode for full BGP attribute manipulation.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the route policy set was created.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the route policy set was last updated.",
				Computed:            true,
			},
		},

		Blocks: map[string]schema.Block{
			"rules": schema.ListNestedBlock{
				MarkdownDescription: "Ordered list of policy rules evaluated in sequence order.",
				NestedObject: schema.NestedBlockObject{
					Attributes: ruleAttrs,
				},
			},
		},
	}
}

func (r *RoutePolicySetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoutePolicySetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoutePolicySetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := durantic.NewCreateRoutePolicySetSchema(data.Name.ValueString())
	createReq.SetDescription(data.Description.ValueString())
	createReq.SetDefaultAction(data.DefaultAction.ValueString())
	createReq.SetAdvancedMode(data.AdvancedMode.ValueBool())

	if !data.LocalPref.IsNull() {
		createReq.SetLocalPref(int32(data.LocalPref.ValueInt64()))
	}

	policySet, httpResp, err := r.client.RoutePolicySetsAPI.
		ControlplaneApiCreateRoutePolicySet(ctx).
		CreateRoutePolicySetSchema(*createReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode >= 300 {
			resp.Diagnostics.AddError(
				"Error Creating Route Policy Set",
				fmt.Sprintf("Could not create route policy set, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
			var raw routePolicySetRaw
			if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr != nil {
				resp.Diagnostics.AddError(
					"Error Creating Route Policy Set",
					fmt.Sprintf("Could not parse route policy set response: %s", jsonErr),
				)
				return
			}
			mapRawToRoutePolicySetModel(&raw, &data)
		} else {
			resp.Diagnostics.AddError(
				"Error Creating Route Policy Set",
				fmt.Sprintf("Could not create route policy set, unexpected error: %s", extractAPIError(httpResp, err)),
			)
			return
		}
	} else {
		mapRoutePolicySetToModel(policySet, &data)
	}

	createdRules := make([]RoutePolicyRuleModel, 0, len(data.Rules))
	for _, ruleModel := range data.Rules {
		ruleReq, diags := buildCreateRuleRequest(ctx, ruleModel)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		ruleResp, ruleHTTPResp, ruleErr := r.client.RoutePolicySetsAPI.
			ControlplaneApiCreateRoutePolicyRule(ctx, data.UUID.ValueString()).
			CreateRoutePolicyRuleSchema(*ruleReq).
			Execute()

		var rm RoutePolicyRuleModel
		if ruleErr != nil {
			if ruleHTTPResp != nil && ruleHTTPResp.StatusCode >= 300 {
				resp.Diagnostics.AddError(
					"Error Creating Route Policy Rule",
					fmt.Sprintf("Could not create rule for policy set %s: %s", data.UUID.ValueString(), extractAPIError(ruleHTTPResp, ruleErr)),
				)
				return
			}
			if ruleAPIErr, ok := ruleErr.(*durantic.GenericOpenAPIError); ok {
				var rawRule routePolicyRuleRaw
				if jsonErr := json.Unmarshal(ruleAPIErr.Body(), &rawRule); jsonErr != nil {
					resp.Diagnostics.AddError(
						"Error Creating Route Policy Rule",
						fmt.Sprintf("Could not parse rule response: %s", jsonErr),
					)
					return
				}
				var d diag.Diagnostics
				rm, d = mapRawToRuleModel(ctx, &rawRule)
				resp.Diagnostics.Append(d...)
			} else {
				resp.Diagnostics.AddError(
					"Error Creating Route Policy Rule",
					fmt.Sprintf("Could not create rule for policy set %s: %s", data.UUID.ValueString(), extractAPIError(ruleHTTPResp, ruleErr)),
				)
				return
			}
		} else {
			var d diag.Diagnostics
			rm, d = mapRuleAPIToModel(ctx, ruleResp)
			resp.Diagnostics.Append(d...)
		}
		if resp.Diagnostics.HasError() {
			return
		}
		createdRules = append(createdRules, rm)
	}
	data.Rules = createdRules

	tflog.Trace(ctx, "created route policy set")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoutePolicySetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoutePolicySetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policySet, httpResp, err := r.client.RoutePolicySetsAPI.
		ControlplaneApiGetRoutePolicySet(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw routePolicySetRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToRoutePolicySetModel(&raw, &data)
					ruleModels := make([]RoutePolicyRuleModel, 0, len(raw.Rules))
					for i := range raw.Rules {
						rm, d := mapRawToRuleModel(ctx, &raw.Rules[i])
						resp.Diagnostics.Append(d...)
						ruleModels = append(ruleModels, rm)
					}
					if !resp.Diagnostics.HasError() {
						data.Rules = ruleModels
						resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
					}
					return
				}
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading Route Policy Set",
			fmt.Sprintf("Could not read route policy set %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	mapRoutePolicySetToModel(policySet, &data)

	apiRules := policySet.GetRules()
	ruleModels := make([]RoutePolicyRuleModel, 0, len(apiRules))
	for i := range apiRules {
		rm, diags := mapRuleAPIToModel(ctx, &apiRules[i])
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		ruleModels = append(ruleModels, rm)
	}
	data.Rules = ruleModels

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoutePolicySetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RoutePolicySetResourceModel
	var state RoutePolicySetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := durantic.NewUpdateRoutePolicySetSchema()
	updateReq.SetName(data.Name.ValueString())
	updateReq.SetDescription(data.Description.ValueString())
	updateReq.SetDefaultAction(data.DefaultAction.ValueString())
	updateReq.SetAdvancedMode(data.AdvancedMode.ValueBool())

	if !data.LocalPref.IsNull() {
		updateReq.SetLocalPref(int32(data.LocalPref.ValueInt64()))
	} else {
		updateReq.SetLocalPrefNil()
	}

	policySet, httpResp, err := r.client.RoutePolicySetsAPI.
		ControlplaneApiUpdateRoutePolicySet(ctx, data.UUID.ValueString()).
		UpdateRoutePolicySetSchema(*updateReq).
		Execute()

	if err != nil {
		if httpResp != nil && httpResp.StatusCode < 300 {
			if apiErr, ok := err.(*durantic.GenericOpenAPIError); ok {
				var raw routePolicySetRaw
				if jsonErr := json.Unmarshal(apiErr.Body(), &raw); jsonErr == nil {
					mapRawToRoutePolicySetModel(&raw, &data)
				} else {
					resp.Diagnostics.AddError(
						"Error Updating Route Policy Set",
						fmt.Sprintf("Could not parse route policy set response: %s", jsonErr),
					)
					return
				}
			} else {
				resp.Diagnostics.AddError(
					"Error Updating Route Policy Set",
					fmt.Sprintf("Could not update route policy set %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
				)
				return
			}
		} else {
			resp.Diagnostics.AddError(
				"Error Updating Route Policy Set",
				fmt.Sprintf("Could not update route policy set %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
			)
			return
		}
	} else {
		mapRoutePolicySetToModel(policySet, &data)
	}

	// Build set of plan rule UUIDs (known, non-empty = existing rules to keep/update).
	planUUIDs := make(map[string]struct{}, len(data.Rules))
	for _, pr := range data.Rules {
		if !pr.UUID.IsNull() && !pr.UUID.IsUnknown() && pr.UUID.ValueString() != "" {
			planUUIDs[pr.UUID.ValueString()] = struct{}{}
		}
	}

	// Delete state rules not present in the plan.
	for _, sr := range state.Rules {
		if sr.UUID.IsNull() || sr.UUID.IsUnknown() || sr.UUID.ValueString() == "" {
			continue
		}
		if _, keep := planUUIDs[sr.UUID.ValueString()]; !keep {
			delHTTPResp, delErr := r.client.RoutePolicySetsAPI.
				ControlplaneApiDeleteRoutePolicyRule(ctx, data.UUID.ValueString(), sr.UUID.ValueString()).
				Execute()
			if delErr != nil && (delHTTPResp == nil || delHTTPResp.StatusCode != 404) {
				resp.Diagnostics.AddError(
					"Error Deleting Route Policy Rule",
					fmt.Sprintf("Could not delete rule %s: %s", sr.UUID.ValueString(), extractAPIError(delHTTPResp, delErr)),
				)
				return
			}
		}
	}

	// Update existing rules and create new ones.
	updatedRules := make([]RoutePolicyRuleModel, 0, len(data.Rules))
	for _, pr := range data.Rules {
		if !pr.UUID.IsNull() && !pr.UUID.IsUnknown() && pr.UUID.ValueString() != "" {
			updateRuleReq, diags := buildUpdateRuleRequest(ctx, pr)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			ruleResp, ruleHTTPResp, ruleErr := r.client.RoutePolicySetsAPI.
				ControlplaneApiUpdateRoutePolicyRule(ctx, data.UUID.ValueString(), pr.UUID.ValueString()).
				UpdateRoutePolicyRuleSchema(*updateRuleReq).
				Execute()

			var rm RoutePolicyRuleModel
			if ruleErr != nil {
				if ruleHTTPResp != nil && ruleHTTPResp.StatusCode >= 300 {
					resp.Diagnostics.AddError(
						"Error Updating Route Policy Rule",
						fmt.Sprintf("Could not update rule %s: %s", pr.UUID.ValueString(), extractAPIError(ruleHTTPResp, ruleErr)),
					)
					return
				}
				if ruleAPIErr, ok := ruleErr.(*durantic.GenericOpenAPIError); ok {
					var rawRule routePolicyRuleRaw
					if jsonErr := json.Unmarshal(ruleAPIErr.Body(), &rawRule); jsonErr != nil {
						resp.Diagnostics.AddError(
							"Error Updating Route Policy Rule",
							fmt.Sprintf("Could not parse rule response: %s", jsonErr),
						)
						return
					}
					var d diag.Diagnostics
					rm, d = mapRawToRuleModel(ctx, &rawRule)
					resp.Diagnostics.Append(d...)
				} else {
					resp.Diagnostics.AddError(
						"Error Updating Route Policy Rule",
						fmt.Sprintf("Could not update rule %s: %s", pr.UUID.ValueString(), extractAPIError(ruleHTTPResp, ruleErr)),
					)
					return
				}
			} else {
				var d diag.Diagnostics
				rm, d = mapRuleAPIToModel(ctx, ruleResp)
				resp.Diagnostics.Append(d...)
			}
			if resp.Diagnostics.HasError() {
				return
			}
			updatedRules = append(updatedRules, rm)
		} else {
			createRuleReq, diags := buildCreateRuleRequest(ctx, pr)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			ruleResp, ruleHTTPResp, ruleErr := r.client.RoutePolicySetsAPI.
				ControlplaneApiCreateRoutePolicyRule(ctx, data.UUID.ValueString()).
				CreateRoutePolicyRuleSchema(*createRuleReq).
				Execute()

			var rm RoutePolicyRuleModel
			if ruleErr != nil {
				if ruleHTTPResp != nil && ruleHTTPResp.StatusCode >= 300 {
					resp.Diagnostics.AddError(
						"Error Creating Route Policy Rule",
						fmt.Sprintf("Could not create rule for policy set %s: %s", data.UUID.ValueString(), extractAPIError(ruleHTTPResp, ruleErr)),
					)
					return
				}
				if ruleAPIErr, ok := ruleErr.(*durantic.GenericOpenAPIError); ok {
					var rawRule routePolicyRuleRaw
					if jsonErr := json.Unmarshal(ruleAPIErr.Body(), &rawRule); jsonErr != nil {
						resp.Diagnostics.AddError(
							"Error Creating Route Policy Rule",
							fmt.Sprintf("Could not parse rule response: %s", jsonErr),
						)
						return
					}
					var d diag.Diagnostics
					rm, d = mapRawToRuleModel(ctx, &rawRule)
					resp.Diagnostics.Append(d...)
				} else {
					resp.Diagnostics.AddError(
						"Error Creating Route Policy Rule",
						fmt.Sprintf("Could not create rule for policy set %s: %s", data.UUID.ValueString(), extractAPIError(ruleHTTPResp, ruleErr)),
					)
					return
				}
			} else {
				var d diag.Diagnostics
				rm, d = mapRuleAPIToModel(ctx, ruleResp)
				resp.Diagnostics.Append(d...)
			}
			if resp.Diagnostics.HasError() {
				return
			}
			updatedRules = append(updatedRules, rm)
		}
	}
	data.Rules = updatedRules

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoutePolicySetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoutePolicySetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.RoutePolicySetsAPI.
		ControlplaneApiDeleteRoutePolicySet(ctx, data.UUID.ValueString()).
		Execute()

	if err != nil && (httpResp == nil || httpResp.StatusCode >= 300) {
		if httpResp != nil && httpResp.StatusCode == 404 {
			tflog.Trace(ctx, "route policy set already deleted")
			return
		}

		resp.Diagnostics.AddError(
			"Error Deleting Route Policy Set",
			fmt.Sprintf("Could not delete route policy set %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
		)
		return
	}

	tflog.Trace(ctx, "deleted route policy set")
}

func (r *RoutePolicySetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// ModifyPlan marks uuid as unknown for new rule blocks.
// schema.ListNestedBlock does not auto-mark Computed attributes as unknown for
// new elements the way schema.ListNestedAttribute does, so we do it here.
func (r *RoutePolicySetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan RoutePolicySetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	modified := false
	for i := range plan.Rules {
		if plan.Rules[i].UUID.IsNull() {
			plan.Rules[i].UUID = types.StringUnknown()
			modified = true
		}
	}

	if modified {
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	}
}

func mapRoutePolicySetToModel(ps *durantic.RoutePolicySetSchema, model *RoutePolicySetResourceModel) {
	model.UUID = types.StringValue(ps.GetUuid())
	model.Name = types.StringValue(ps.GetName())
	model.Description = types.StringValue(ps.GetDescription())
	model.DefaultAction = types.StringValue(ps.GetDefaultAction())
	model.AdvancedMode = types.BoolValue(ps.GetAdvancedMode())
	model.CreatedAt = types.StringValue(ps.GetCreatedAt())
	model.UpdatedAt = types.StringValue(ps.GetUpdatedAt())

	if lp, ok := ps.GetLocalPrefOk(); ok && lp != nil {
		model.LocalPref = types.Int64Value(int64(*lp))
	} else {
		model.LocalPref = types.Int64Null()
	}
}

func mapRuleAPIToModel(ctx context.Context, rule *durantic.RoutePolicyRuleSchema) (RoutePolicyRuleModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var model RoutePolicyRuleModel

	model.UUID = types.StringValue(rule.GetUuid())
	model.Sequence = types.Int64Value(int64(rule.GetSequence()))
	model.Enabled = types.BoolValue(rule.GetEnabled())
	model.Description = types.StringValue(rule.GetDescription())
	model.MatchType = types.StringValue(rule.GetMatchType())
	model.MatchAsPathRegex = types.StringValue(rule.GetMatchAsPathRegex())
	model.Action = types.StringValue(rule.GetAction())
	model.PrependAsCount = types.Int64Value(int64(rule.GetPrependAsCount()))

	if v, isSet := rule.GetMatchPrefixLenMinOk(); isSet && v != nil {
		model.MatchPrefixLenMin = types.Int64Value(int64(*v))
	} else {
		model.MatchPrefixLenMin = types.Int64Null()
	}

	if v, isSet := rule.GetMatchPrefixLenMaxOk(); isSet && v != nil {
		model.MatchPrefixLenMax = types.Int64Value(int64(*v))
	} else {
		model.MatchPrefixLenMax = types.Int64Null()
	}

	if v, isSet := rule.GetSetLocalPrefOk(); isSet && v != nil {
		model.SetLocalPref = types.Int64Value(int64(*v))
	} else {
		model.SetLocalPref = types.Int64Null()
	}

	if v, isSet := rule.GetSetMedOk(); isSet && v != nil {
		model.SetMed = types.Int64Value(int64(*v))
	} else {
		model.SetMed = types.Int64Null()
	}

	model.MatchPrefixes = listFromStrings(ctx, rule.GetMatchPrefixes(), &diags)
	model.MatchCommunities = listFromStrings(ctx, rule.GetMatchCommunities(), &diags)
	model.SetCommunities = listFromStrings(ctx, rule.GetSetCommunities(), &diags)
	model.AddCommunities = listFromStrings(ctx, rule.GetAddCommunities(), &diags)
	model.RemoveCommunities = listFromStrings(ctx, rule.GetRemoveCommunities(), &diags)

	return model, diags
}

func buildCreateRuleRequest(ctx context.Context, model RoutePolicyRuleModel) (*durantic.CreateRoutePolicyRuleSchema, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := durantic.NewCreateRoutePolicyRuleSchema()

	req.SetSequence(int32(model.Sequence.ValueInt64()))
	req.SetEnabled(model.Enabled.ValueBool())
	req.SetDescription(model.Description.ValueString())
	req.SetMatchType(model.MatchType.ValueString())
	req.SetMatchAsPathRegex(model.MatchAsPathRegex.ValueString())
	req.SetAction(model.Action.ValueString())
	req.SetPrependAsCount(int32(model.PrependAsCount.ValueInt64()))

	if !model.MatchPrefixLenMin.IsNull() {
		req.SetMatchPrefixLenMin(int32(model.MatchPrefixLenMin.ValueInt64()))
	}
	if !model.MatchPrefixLenMax.IsNull() {
		req.SetMatchPrefixLenMax(int32(model.MatchPrefixLenMax.ValueInt64()))
	}
	if !model.SetLocalPref.IsNull() {
		req.SetSetLocalPref(int32(model.SetLocalPref.ValueInt64()))
	}
	if !model.SetMed.IsNull() {
		req.SetSetMed(int32(model.SetMed.ValueInt64()))
	}

	if !model.MatchPrefixes.IsNull() && !model.MatchPrefixes.IsUnknown() {
		var s []string
		diags.Append(model.MatchPrefixes.ElementsAs(ctx, &s, false)...)
		req.SetMatchPrefixes(s)
	}
	if !model.MatchCommunities.IsNull() && !model.MatchCommunities.IsUnknown() {
		var s []string
		diags.Append(model.MatchCommunities.ElementsAs(ctx, &s, false)...)
		req.SetMatchCommunities(s)
	}
	if !model.SetCommunities.IsNull() && !model.SetCommunities.IsUnknown() {
		var s []string
		diags.Append(model.SetCommunities.ElementsAs(ctx, &s, false)...)
		req.SetSetCommunities(s)
	}
	if !model.AddCommunities.IsNull() && !model.AddCommunities.IsUnknown() {
		var s []string
		diags.Append(model.AddCommunities.ElementsAs(ctx, &s, false)...)
		req.SetAddCommunities(s)
	}
	if !model.RemoveCommunities.IsNull() && !model.RemoveCommunities.IsUnknown() {
		var s []string
		diags.Append(model.RemoveCommunities.ElementsAs(ctx, &s, false)...)
		req.SetRemoveCommunities(s)
	}

	return req, diags
}

func buildUpdateRuleRequest(ctx context.Context, model RoutePolicyRuleModel) (*durantic.UpdateRoutePolicyRuleSchema, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := durantic.NewUpdateRoutePolicyRuleSchema()

	req.SetSequence(int32(model.Sequence.ValueInt64()))
	req.SetEnabled(model.Enabled.ValueBool())
	req.SetDescription(model.Description.ValueString())
	req.SetMatchType(model.MatchType.ValueString())
	req.SetMatchAsPathRegex(model.MatchAsPathRegex.ValueString())
	req.SetAction(model.Action.ValueString())
	req.SetPrependAsCount(int32(model.PrependAsCount.ValueInt64()))

	if !model.MatchPrefixLenMin.IsNull() {
		req.SetMatchPrefixLenMin(int32(model.MatchPrefixLenMin.ValueInt64()))
	} else {
		req.SetMatchPrefixLenMinNil()
	}
	if !model.MatchPrefixLenMax.IsNull() {
		req.SetMatchPrefixLenMax(int32(model.MatchPrefixLenMax.ValueInt64()))
	} else {
		req.SetMatchPrefixLenMaxNil()
	}
	if !model.SetLocalPref.IsNull() {
		req.SetSetLocalPref(int32(model.SetLocalPref.ValueInt64()))
	} else {
		req.SetSetLocalPrefNil()
	}
	if !model.SetMed.IsNull() {
		req.SetSetMed(int32(model.SetMed.ValueInt64()))
	} else {
		req.SetSetMedNil()
	}

	if !model.MatchPrefixes.IsNull() && !model.MatchPrefixes.IsUnknown() {
		var s []string
		diags.Append(model.MatchPrefixes.ElementsAs(ctx, &s, false)...)
		req.SetMatchPrefixes(s)
	} else {
		req.SetMatchPrefixes([]string{})
	}
	if !model.MatchCommunities.IsNull() && !model.MatchCommunities.IsUnknown() {
		var s []string
		diags.Append(model.MatchCommunities.ElementsAs(ctx, &s, false)...)
		req.SetMatchCommunities(s)
	} else {
		req.SetMatchCommunities([]string{})
	}
	if !model.SetCommunities.IsNull() && !model.SetCommunities.IsUnknown() {
		var s []string
		diags.Append(model.SetCommunities.ElementsAs(ctx, &s, false)...)
		req.SetSetCommunities(s)
	} else {
		req.SetSetCommunities([]string{})
	}
	if !model.AddCommunities.IsNull() && !model.AddCommunities.IsUnknown() {
		var s []string
		diags.Append(model.AddCommunities.ElementsAs(ctx, &s, false)...)
		req.SetAddCommunities(s)
	} else {
		req.SetAddCommunities([]string{})
	}
	if !model.RemoveCommunities.IsNull() && !model.RemoveCommunities.IsUnknown() {
		var s []string
		diags.Append(model.RemoveCommunities.ElementsAs(ctx, &s, false)...)
		req.SetRemoveCommunities(s)
	} else {
		req.SetRemoveCommunities([]string{})
	}

	return req, diags
}

type routePolicyRuleRaw struct {
	UUID              string   `json:"uuid"`
	Sequence          int32    `json:"sequence"`
	Enabled           bool     `json:"enabled"`
	Description       string   `json:"description"`
	MatchType         string   `json:"match_type"`
	MatchPrefixes     []string `json:"match_prefixes"`
	MatchPrefixLenMin *int32   `json:"match_prefix_len_min"`
	MatchPrefixLenMax *int32   `json:"match_prefix_len_max"`
	MatchCommunities  []string `json:"match_communities"`
	MatchAsPathRegex  string   `json:"match_as_path_regex"`
	Action            string   `json:"action"`
	SetLocalPref      *int32   `json:"set_local_pref"`
	SetMed            *int32   `json:"set_med"`
	SetCommunities    []string `json:"set_communities"`
	AddCommunities    []string `json:"add_communities"`
	RemoveCommunities []string `json:"remove_communities"`
	PrependAsCount    int32    `json:"prepend_as_count"`
}

type routePolicySetRaw struct {
	UUID          string               `json:"uuid"`
	Name          string               `json:"name"`
	Description   string               `json:"description"`
	DefaultAction string               `json:"default_action"`
	LocalPref     *int32               `json:"local_pref"`
	AdvancedMode  bool                 `json:"advanced_mode"`
	Rules         []routePolicyRuleRaw `json:"rules"`
	CreatedAt     string               `json:"created_at"`
	UpdatedAt     string               `json:"updated_at"`
}

func mapRawToRoutePolicySetModel(raw *routePolicySetRaw, model *RoutePolicySetResourceModel) {
	model.UUID = types.StringValue(raw.UUID)
	model.Name = types.StringValue(raw.Name)
	model.Description = types.StringValue(raw.Description)
	model.DefaultAction = types.StringValue(raw.DefaultAction)
	model.AdvancedMode = types.BoolValue(raw.AdvancedMode)
	model.CreatedAt = types.StringValue(raw.CreatedAt)
	model.UpdatedAt = types.StringValue(raw.UpdatedAt)

	if raw.LocalPref != nil {
		model.LocalPref = types.Int64Value(int64(*raw.LocalPref))
	} else {
		model.LocalPref = types.Int64Null()
	}
}

func mapRawToRuleModel(ctx context.Context, raw *routePolicyRuleRaw) (RoutePolicyRuleModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var model RoutePolicyRuleModel

	model.UUID = types.StringValue(raw.UUID)
	model.Sequence = types.Int64Value(int64(raw.Sequence))
	model.Enabled = types.BoolValue(raw.Enabled)
	model.Description = types.StringValue(raw.Description)
	model.MatchType = types.StringValue(raw.MatchType)
	model.MatchAsPathRegex = types.StringValue(raw.MatchAsPathRegex)
	model.Action = types.StringValue(raw.Action)
	model.PrependAsCount = types.Int64Value(int64(raw.PrependAsCount))

	if raw.MatchPrefixLenMin != nil {
		model.MatchPrefixLenMin = types.Int64Value(int64(*raw.MatchPrefixLenMin))
	} else {
		model.MatchPrefixLenMin = types.Int64Null()
	}

	if raw.MatchPrefixLenMax != nil {
		model.MatchPrefixLenMax = types.Int64Value(int64(*raw.MatchPrefixLenMax))
	} else {
		model.MatchPrefixLenMax = types.Int64Null()
	}

	if raw.SetLocalPref != nil {
		model.SetLocalPref = types.Int64Value(int64(*raw.SetLocalPref))
	} else {
		model.SetLocalPref = types.Int64Null()
	}

	if raw.SetMed != nil {
		model.SetMed = types.Int64Value(int64(*raw.SetMed))
	} else {
		model.SetMed = types.Int64Null()
	}

	model.MatchPrefixes = listFromStrings(ctx, raw.MatchPrefixes, &diags)
	model.MatchCommunities = listFromStrings(ctx, raw.MatchCommunities, &diags)
	model.SetCommunities = listFromStrings(ctx, raw.SetCommunities, &diags)
	model.AddCommunities = listFromStrings(ctx, raw.AddCommunities, &diags)
	model.RemoveCommunities = listFromStrings(ctx, raw.RemoveCommunities, &diags)

	return model, diags
}

// listFromStrings maps an API []string to a Terraform types.List.
// Returns null when the slice is empty so that unset optional list attributes
// remain null in state, consistent with the null value in the plan.
func listFromStrings(ctx context.Context, s []string, diags *diag.Diagnostics) types.List {
	if len(s) == 0 {
		return types.ListNull(types.StringType)
	}
	v, d := types.ListValueFrom(ctx, types.StringType, s)
	diags.Append(d...)
	return v
}
