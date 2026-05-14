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

func stringListFromPointers(values []*string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
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
	model.WgIPAddress = nullableStringValue(machine.GetWgIpAddressOk())
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

	discoveredIPAddresses := stringListFromPointers(machine.GetDiscoveredIpAddresses())
	discoveredIPs, d := stringListValue(discoveredIPAddresses)
	diags.Append(d...)
	model.DiscoveredIPAddresses = discoveredIPs

	publicIPs, d := stringListValue(discoveredIPAddresses)
	diags.Append(d...)
	model.PublicIPAddresses = publicIPs

	privateIPAddresses := []string{}
	if !model.WgIPAddress.IsNull() && !model.WgIPAddress.IsUnknown() && model.WgIPAddress.ValueString() != "" {
		privateIPAddresses = append(privateIPAddresses, model.WgIPAddress.ValueString())
	}
	privateIPs, d := stringListValue(privateIPAddresses)
	diags.Append(d...)
	model.PrivateIPAddresses = privateIPs

	return diags
}

func mapMachineSchemaToCommonModel(machine *durantic.MachineSchema, model *MachineCommonModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.UUID = types.StringValue(machine.GetUuid())
	model.Hostname = types.StringValue(machine.GetHostname())
	model.NeedsProvisioning = nullableBoolValue(machine.GetNeedsProvisioningOk())
	model.PendingConfigPush = nullableBoolValue(machine.GetPendingConfigPushOk())
	model.WgIPAddress = nullableStringValue(machine.GetWgIpAddressOk())
	model.IsOnline = nullableBoolValue(machine.GetIsOnlineOk())
	model.MeshNetworkUUID = nullableStringValue(machine.GetMeshNetworkUuidOk())
	model.TunnelType = types.StringValue(string(machine.GetTunnelType()))
	model.StunEnabled = nullableBoolValue(machine.GetStunEnabledOk())
	model.AutoUpdate = nullableBoolValue(machine.GetAutoUpdateOk())
	model.InjectAgent = nullableBoolValue(machine.GetInjectAgentOk())
	model.KexecInstaller = nullableBoolValue(machine.GetKexecInstallerOk())
	model.TargetDisk = types.StringNull()

	roleNames, d := stringListValue(machine.GetRoleNames())
	diags.Append(d...)
	model.RoleNames = roleNames

	emptyList, d := stringListValue([]string{})
	diags.Append(d...)
	model.DiscoveredIPAddresses = emptyList
	model.PublicIPAddresses = emptyList

	privateIPAddresses := []string{}
	if !model.WgIPAddress.IsNull() && !model.WgIPAddress.IsUnknown() && model.WgIPAddress.ValueString() != "" {
		privateIPAddresses = append(privateIPAddresses, model.WgIPAddress.ValueString())
	}
	privateIPs, d := stringListValue(privateIPAddresses)
	diags.Append(d...)
	model.PrivateIPAddresses = privateIPs

	return diags
}
