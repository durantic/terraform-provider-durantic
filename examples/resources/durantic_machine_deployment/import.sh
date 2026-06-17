# SPDX-License-Identifier: MPL-2.0

# Import an existing machine deployment by machine UUID.
# Note: force_provision and provision_uuid/provision_status are not imported
# and should be set manually after import.
terraform import durantic_machine_deployment.example "00000000-0000-0000-0000-000000000000"
