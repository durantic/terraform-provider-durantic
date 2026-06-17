#!/bin/bash
# Copyright (c) HashiCorp, Inc.

# Note: the secret value is not returned by the API.
# After import, set the value attribute manually in your configuration.
terraform import durantic_secret.example <UUID>
