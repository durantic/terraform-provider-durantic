#!/bin/bash

# Import a machine by UUID
# Machines are auto-discovered and cannot be created via Terraform
# You must import them first to manage their configuration

MACHINE_UUID="550e8400-e29b-41d4-a716-446655440000"  # Replace with actual UUID

terraform import durantic_machine.example "${MACHINE_UUID}"

# Import multiple machines
# terraform import durantic_machine.web "web-machine-uuid"
# terraform import durantic_machine.db "db-machine-uuid"
# terraform import durantic_machine.app "app-machine-uuid"
