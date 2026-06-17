package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccImageDataSource_NotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "durantic_image" "missing" {
  docker_image_url = "example.invalid/durantic/terraform-provider-missing-image:latest"
}
`,
				ExpectError: regexp.MustCompile("No Image Found"),
			},
		},
	})
}

func TestAccImageDataSource_ByUUID(t *testing.T) {
	imageUUID := os.Getenv("DURANTIC_TEST_IMAGE_UUID")
	if imageUUID == "" {
		t.Skip("DURANTIC_TEST_IMAGE_UUID must be set for this acceptance test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "durantic_image" "test" {
  uuid = %[1]q
}
`, imageUUID),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.durantic_image.test",
						tfjsonpath.New("uuid"),
						knownvalue.StringExact(imageUUID),
					),
					statecheck.ExpectKnownValue(
						"data.durantic_image.test",
						tfjsonpath.New("docker_image_url"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccImageDataSource_ByName(t *testing.T) {
	imageName := os.Getenv("DURANTIC_TEST_IMAGE_NAME")
	if imageName == "" {
		t.Skip("DURANTIC_TEST_IMAGE_NAME must be set for this acceptance test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "durantic_image" "test" {
  name = %[1]q
}
`, imageName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.durantic_image.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(imageName),
					),
					statecheck.ExpectKnownValue(
						"data.durantic_image.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccImageDataSource_ByDockerImageURL(t *testing.T) {
	dockerImageURL := os.Getenv("DURANTIC_TEST_IMAGE_DOCKER_IMAGE_URL")
	if dockerImageURL == "" {
		t.Skip("DURANTIC_TEST_IMAGE_DOCKER_IMAGE_URL must be set for this acceptance test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "durantic_image" "test" {
  docker_image_url = %[1]q
}
`, dockerImageURL),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.durantic_image.test",
						tfjsonpath.New("docker_image_url"),
						knownvalue.StringExact(dockerImageURL),
					),
					statecheck.ExpectKnownValue(
						"data.durantic_image.test",
						tfjsonpath.New("uuid"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}
