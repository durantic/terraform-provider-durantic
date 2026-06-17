# Copyright (c) HashiCorp, Inc.

# SPDX-License-Identifier: MPL-2.0

data "durantic_machine" "web" {
  hostname = "web-01"
}

resource "durantic_machine_deployment" "web" {
  machine_uuid      = data.durantic_machine.web.uuid
  mesh_network_uuid = "00000000-0000-0000-0000-000000000000"
  role_names        = ["base-ubuntu", "nginx"]

  # Bump this value to force re-provision all machines in the group
  # without changing their configuration.
  force_provision = "v1"
}
