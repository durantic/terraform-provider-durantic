// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RoutePolicySetDataSource{}

func NewRoutePolicySetDataSource() datasource.DataSource {
	return &RoutePolicySetDataSource{}
}

type RoutePolicySetDataSource struct {
	client *durantic.APIClient
}

func (d *RoutePolicySetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route_policy_set"
}

func (d *RoutePolicySetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	stringList := func(description string) schema.ListAttribute {
		return schema.ListAttribute{
			MarkdownDescription: description,
			Computed:            true,
			ElementType:         types.StringType,
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a Durantic route policy set by UUID or name.",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for the route policy set. Set either `uuid` or `name`.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the route policy set. Set either `uuid` or `name`.",
				Optional:            true,
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the route policy set.",
				Computed:            true,
			},
			"default_action": schema.StringAttribute{
				MarkdownDescription: "Action taken when no rule matches.",
				Computed:            true,
			},
			"local_pref": schema.Int64Attribute{
				MarkdownDescription: "Local preference applied to routes accepted by the policy set, if any.",
				Computed:            true,
			},
			"advanced_mode": schema.BoolAttribute{
				MarkdownDescription: "Whether the policy set uses advanced mode (explicit rules).",
				Computed:            true,
			},
			"rules": schema.ListNestedAttribute{
				MarkdownDescription: "Policy rules in evaluation order.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for the rule.",
							Computed:            true,
						},
						"sequence": schema.Int64Attribute{
							MarkdownDescription: "Rule evaluation order. Lower numbers are evaluated first.",
							Computed:            true,
						},
						"enabled": schema.BoolAttribute{
							MarkdownDescription: "Whether this rule is active.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Description of the rule.",
							Computed:            true,
						},
						"match_type": schema.StringAttribute{
							MarkdownDescription: "What the rule matches on.",
							Computed:            true,
						},
						"match_prefixes":       stringList("Prefixes the rule matches."),
						"match_prefix_len_min": schema.Int64Attribute{MarkdownDescription: "Minimum prefix length to match.", Computed: true},
						"match_prefix_len_max": schema.Int64Attribute{MarkdownDescription: "Maximum prefix length to match.", Computed: true},
						"match_communities":    stringList("BGP communities the rule matches."),
						"match_as_path_regex": schema.StringAttribute{
							MarkdownDescription: "AS path regular expression the rule matches.",
							Computed:            true,
						},
						"action": schema.StringAttribute{
							MarkdownDescription: "Action taken when the rule matches (`accept` or `reject`).",
							Computed:            true,
						},
						"set_local_pref":     schema.Int64Attribute{MarkdownDescription: "Local preference to set on matching routes.", Computed: true},
						"set_med":            schema.Int64Attribute{MarkdownDescription: "MED to set on matching routes.", Computed: true},
						"set_communities":    stringList("Communities to set on matching routes, replacing existing ones."),
						"add_communities":    stringList("Communities to add to matching routes."),
						"remove_communities": stringList("Communities to remove from matching routes."),
						"prepend_as_count": schema.Int64Attribute{
							MarkdownDescription: "Number of times to prepend the local AS to matching routes.",
							Computed:            true,
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the route policy set was created.",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when the route policy set was last updated.",
				Computed:            true,
			},
		},
	}
}

func (d *RoutePolicySetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RoutePolicySetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RoutePolicySetResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !requireUUIDOrName("Route Policy Set", data.UUID, data.Name, &resp.Diagnostics) {
		return
	}

	var policySet *durantic.RoutePolicySetSchema

	if isKnownString(data.UUID) {
		got, httpResp, err := d.client.RoutePolicySetsAPI.
			ControlplaneApiGetRoutePolicySet(ctx, data.UUID.ValueString()).
			Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Route Policy Set",
				fmt.Sprintf("Could not read route policy set %s: %s", data.UUID.ValueString(), extractAPIError(httpResp, err)),
			)
			return
		}
		policySet = got
	} else {
		policySets, httpResp, err := d.client.RoutePolicySetsAPI.ControlplaneApiListRoutePolicySets(ctx).Execute()
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Listing Route Policy Sets",
				fmt.Sprintf("Could not list route policy sets: %s", extractAPIError(httpResp, err)),
			)
			return
		}
		match, ok := findUniqueByName("Route Policy Set", data.Name.ValueString(), policySets, func(p durantic.RoutePolicySetSchema) string { return p.GetName() }, &resp.Diagnostics)
		if !ok {
			return
		}
		policySet = &match
	}

	resp.Diagnostics.Append(mapRoutePolicySetToDataSourceModel(ctx, policySet, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// mapRoutePolicySetToDataSourceModel maps a policy set and all its rules onto
// the shared policy set model, reusing the resource mappers.
func mapRoutePolicySetToDataSourceModel(ctx context.Context, ps *durantic.RoutePolicySetSchema, model *RoutePolicySetResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	mapRoutePolicySetToModel(ps, model)

	apiRules := ps.GetRules()
	rules := make([]RoutePolicyRuleModel, 0, len(apiRules))
	for i := range apiRules {
		rule, d := mapRuleAPIToModel(ctx, &apiRules[i])
		diags.Append(d...)
		rules = append(rules, rule)
	}
	model.Rules = rules

	return diags
}
