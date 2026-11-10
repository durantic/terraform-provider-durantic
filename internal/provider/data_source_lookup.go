// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// requireUUIDOrName checks that exactly one of uuid and name is set on a lookup
// data source, and reports an error diagnostic otherwise. label is the
// human-readable object type (e.g. "VIP").
func requireUUIDOrName(label string, uuid, name types.String, diags *diag.Diagnostics) bool {
	if countSetStrings(uuid, name) != 1 {
		diags.AddError(
			fmt.Sprintf("Invalid %s Lookup", label),
			"Exactly one of uuid or name must be set.",
		)
		return false
	}
	return true
}

// findUniqueByName returns the one item whose name equals name. It reports an
// error diagnostic when nothing matches or when the name is ambiguous, since a
// data source must resolve to a single object.
func findUniqueByName[T any](label, name string, items []T, nameOf func(T) string, diags *diag.Diagnostics) (T, bool) {
	var matches []T
	for _, item := range items {
		if nameOf(item) == name {
			matches = append(matches, item)
		}
	}

	var zero T
	switch len(matches) {
	case 0:
		diags.AddError(
			fmt.Sprintf("No %s Found", label),
			fmt.Sprintf("Could not find a %s named %q.", label, name),
		)
		return zero, false
	case 1:
		return matches[0], true
	default:
		diags.AddError(
			fmt.Sprintf("Multiple %ss Found", label),
			fmt.Sprintf("Found %d %ss named %q. Use uuid for an unambiguous lookup.", len(matches), label, name),
		)
		return zero, false
	}
}
