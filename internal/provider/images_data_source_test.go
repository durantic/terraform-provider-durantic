// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccImagesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "durantic_images" "all" {}`,
				ConfigStateChecks: []statecheck.StateCheck{
					// images list must be present (not null)
					statecheck.ExpectKnownValue(
						"data.durantic_images.all",
						tfjsonpath.New("images"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccImagesDataSource_Fields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "durantic_images" "all" {}`,
				ConfigStateChecks: []statecheck.StateCheck{
					// First image must have non-empty uuid, name, docker_image_url, created_at, updated_at
					statecheck.ExpectKnownValue(
						"data.durantic_images.all",
						tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("uuid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.durantic_images.all",
						tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("name"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.durantic_images.all",
						tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("docker_image_url"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.durantic_images.all",
						tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("created_at"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.durantic_images.all",
						tfjsonpath.New("images").AtSliceIndex(0).AtMapKey("updated_at"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}
