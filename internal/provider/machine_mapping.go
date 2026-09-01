// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringListValue(values []string) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(context.Background(), types.StringType, values)
}

func nullableStringValue(value *string, ok bool) types.String {
	if ok && value != nil {
		return types.StringValue(*value)
	}
	return types.StringNull()
}

func nullableBoolValue(value *bool, ok bool) types.Bool {
	if ok && value != nil {
		return types.BoolValue(*value)
	}
	return types.BoolNull()
}

func mapMachineResponseToCommonModel(machine *durantic.MachineResponseSchema, model *MachineCommonModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.UUID = types.StringValue(machine.GetUuid())
	model.Hostname = types.StringValue(machine.GetHostname())
	model.NeedsProvisioning = types.BoolValue(machine.GetNeedsProvisioning())
	model.PendingConfigPush = nullableBoolValue(machine.GetPendingConfigPushOk())
	// One source, two attributes. The API field is `mesh_ip_address` since
	// controlplane#203; `wg_ip_address` is a deprecated alias so a practitioner's
	// existing config keeps resolving instead of silently going empty.
	model.MeshIPAddress = nullableStringValue(machine.GetMeshIpAddressOk())
	model.WgIPAddress = model.MeshIPAddress
	model.IsOnline = nullableBoolValue(machine.GetIsOnlineOk())
	model.TunnelType = types.StringValue(string(machine.GetTunnelType()))
	model.StunEnabled = nullableBoolValue(machine.GetStunEnabledOk())
	model.TargetDisk = nullableStringValue(machine.GetTargetDiskOk())
	model.AutoUpdate = nullableBoolValue(machine.GetAutoUpdateOk())
	model.InjectAgent = nullableBoolValue(machine.GetInjectAgentOk())
	model.KexecInstaller = nullableBoolValue(machine.GetKexecInstallerOk())

	if meshNetwork, ok := machine.GetMeshNetworkOk(); ok && meshNetwork != nil {
		model.MeshNetworkUUID = types.StringValue(meshNetwork.GetUuid())
	} else {
		model.MeshNetworkUUID = types.StringNull()
	}

	roleNames, d := stringListValue(machine.GetRoleNames())
	diags.Append(d...)
	model.RoleNames = roleNames

	// The generated getter returns []string now, not []*string.
	discoveredIPAddresses := machine.GetDiscoveredIpAddresses()
	discoveredIPs, d := stringListValue(discoveredIPAddresses)
	diags.Append(d...)
	model.DiscoveredIPAddresses = discoveredIPs

	publicIPs, d := stringListValue(discoveredIPAddresses)
	diags.Append(d...)
	model.PublicIPAddresses = publicIPs

	privateIPAddresses := []string{}
	if !model.MeshIPAddress.IsNull() && !model.MeshIPAddress.IsUnknown() && model.MeshIPAddress.ValueString() != "" {
		privateIPAddresses = append(privateIPAddresses, model.MeshIPAddress.ValueString())
	}
	privateIPs, d := stringListValue(privateIPAddresses)
	diags.Append(d...)
	model.PrivateIPAddresses = privateIPs

	return diags
}
