// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"testing"

	durantic "github.com/durantic/controlplane-client-go/durantic"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func testImageModel() *ImageResourceModel {
	return &ImageResourceModel{
		Name:                   types.StringValue("test-image"),
		DockerImageURL:         types.StringValue("registry.example.com/team/app:1.0"),
		Description:            types.StringValue(""),
		RegistryCredentialUUID: types.StringNull(),
	}
}

func TestBuildCreateImageRequest(t *testing.T) {
	t.Run("without a registry credential", func(t *testing.T) {
		m := marshalToMap(t, buildCreateImageRequest(testImageModel()))

		if m["name"] != "test-image" || m["docker_image_url"] != "registry.example.com/team/app:1.0" {
			t.Errorf("name/docker_image_url = %v/%v", m["name"], m["docker_image_url"])
		}
		if v := m["registry_credential_uuid"]; v != nil {
			t.Errorf("registry_credential_uuid = %v, want unset", v)
		}
	})

	t.Run("with a registry credential and description", func(t *testing.T) {
		model := testImageModel()
		model.Description = types.StringValue("an image")
		model.RegistryCredentialUUID = types.StringValue("55555555-5555-5555-5555-555555555555")

		m := marshalToMap(t, buildCreateImageRequest(model))

		if m["description"] != "an image" {
			t.Errorf("description = %v", m["description"])
		}
		if m["registry_credential_uuid"] != "55555555-5555-5555-5555-555555555555" {
			t.Errorf("registry_credential_uuid = %v", m["registry_credential_uuid"])
		}
	})
}

// Removing registry_credential_uuid from configuration has to detach the
// credential, which needs an explicit JSON null — omitting the field would
// leave it attached.
func TestBuildUpdateImageRequest(t *testing.T) {
	t.Run("detaches the credential with an explicit null", func(t *testing.T) {
		m := marshalToMap(t, buildUpdateImageRequest(testImageModel()))

		v, present := m["registry_credential_uuid"]
		if !present {
			t.Fatal("registry_credential_uuid is omitted, want an explicit null")
		}
		if v != nil {
			t.Errorf("registry_credential_uuid = %v, want null", v)
		}
	})

	t.Run("sends every field", func(t *testing.T) {
		model := testImageModel()
		model.Name = types.StringValue("test-image-2")
		model.Description = types.StringValue("updated")
		model.RegistryCredentialUUID = types.StringValue("55555555-5555-5555-5555-555555555555")

		m := marshalToMap(t, buildUpdateImageRequest(model))

		want := map[string]any{
			"name":                     "test-image-2",
			"docker_image_url":         "registry.example.com/team/app:1.0",
			"description":              "updated",
			"registry_credential_uuid": "55555555-5555-5555-5555-555555555555",
		}
		for k, v := range want {
			if m[k] != v {
				t.Errorf("%s = %v, want %v", k, m[k], v)
			}
		}
	})
}

func TestMapImageToResourceModel(t *testing.T) {
	t.Run("maps a private image", func(t *testing.T) {
		var img durantic.ImageSchema
		if err := json.Unmarshal([]byte(`{
			"uuid": "66666666-6666-6666-6666-666666666666",
			"name": "app",
			"docker_image_url": "registry.example.com/team/app:1.0",
			"description": "an image",
			"registry_credential_uuid": "55555555-5555-5555-5555-555555555555",
			"registry_credential_name": "my-registry",
			"is_official": false,
			"created_at": "2026-01-01T00:00:00Z",
			"updated_at": "2026-01-02T00:00:00Z",
			"some_future_field": true
		}`), &img); err != nil {
			t.Fatalf("decoding fixture: %v", err)
		}

		model := &ImageResourceModel{}
		mapImageToResourceModel(&img, model)

		got := map[string]string{
			"uuid":                     model.UUID.ValueString(),
			"name":                     model.Name.ValueString(),
			"docker_image_url":         model.DockerImageURL.ValueString(),
			"description":              model.Description.ValueString(),
			"registry_credential_uuid": model.RegistryCredentialUUID.ValueString(),
			"registry_credential_name": model.RegistryCredentialName.ValueString(),
		}
		want := map[string]string{
			"uuid":                     "66666666-6666-6666-6666-666666666666",
			"name":                     "app",
			"docker_image_url":         "registry.example.com/team/app:1.0",
			"description":              "an image",
			"registry_credential_uuid": "55555555-5555-5555-5555-555555555555",
			"registry_credential_name": "my-registry",
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s = %q, want %q", k, got[k], v)
			}
		}
		if model.IsOfficial.ValueBool() {
			t.Error("is_official = true, want false")
		}
	})

	t.Run("a missing description is an empty string, not null", func(t *testing.T) {
		var img durantic.ImageSchema
		if err := json.Unmarshal([]byte(`{"uuid":"u","name":"n","docker_image_url":"d","created_at":"c","updated_at":"u"}`), &img); err != nil {
			t.Fatalf("decoding fixture: %v", err)
		}
		// Drop the constructor default so the field really is absent.
		img.Description = nil

		model := &ImageResourceModel{}
		mapImageToResourceModel(&img, model)

		if model.Description.IsNull() || model.Description.ValueString() != "" {
			t.Errorf("description = %v, want an empty string", model.Description)
		}
		if !model.RegistryCredentialUUID.IsNull() || !model.RegistryCredentialName.IsNull() {
			t.Errorf("registry credential = (%v, %v), want null", model.RegistryCredentialUUID, model.RegistryCredentialName)
		}
	})
}

func TestMapRawToImageResourceModel(t *testing.T) {
	var raw imageRaw
	if err := json.Unmarshal([]byte(`{
		"uuid": "66666666-6666-6666-6666-666666666666",
		"name": "app",
		"docker_image_url": "registry.example.com/team/app:1.0",
		"registry_credential_uuid": "55555555-5555-5555-5555-555555555555",
		"registry_credential_name": "my-registry",
		"is_official": true,
		"created_at": "c",
		"updated_at": "u"
	}`), &raw); err != nil {
		t.Fatalf("decoding fixture: %v", err)
	}

	model := &ImageResourceModel{}
	mapRawToImageResourceModel(&raw, model)

	if model.UUID.ValueString() != "66666666-6666-6666-6666-666666666666" || model.Name.ValueString() != "app" {
		t.Errorf("uuid/name = %v/%v", model.UUID, model.Name)
	}
	if model.Description.IsNull() || model.Description.ValueString() != "" {
		t.Errorf("description = %v, want an empty string", model.Description)
	}
	if model.RegistryCredentialUUID.ValueString() != "55555555-5555-5555-5555-555555555555" {
		t.Errorf("registry_credential_uuid = %v", model.RegistryCredentialUUID)
	}
	if !model.IsOfficial.ValueBool() {
		t.Error("is_official = false, want true")
	}
}

func TestAccImageResource(t *testing.T) {
	const addr = "durantic_image.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without a registry credential
			{
				Config: testAccImageResourceConfig("test-image", "example.invalid/durantic/test-image:1.0", "", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("name"), knownvalue.StringExact("test-image")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("docker_image_url"), knownvalue.StringExact("example.invalid/durantic/test-image:1.0")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("description"), knownvalue.StringExact("")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("registry_credential_uuid"), knownvalue.Null()),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("is_official"), knownvalue.Bool(false)),
				},
			},
			// Import
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name, URL and description, and attach a registry credential
			{
				Config: testAccImageResourceConfig("test-image-updated", "example.invalid/durantic/test-image:2.0", "an image", true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("name"), knownvalue.StringExact("test-image-updated")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("docker_image_url"), knownvalue.StringExact("example.invalid/durantic/test-image:2.0")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("description"), knownvalue.StringExact("an image")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("registry_credential_name"), knownvalue.StringExact("test-image-registry")),
				},
			},
			// Detach the registry credential again
			{
				Config: testAccImageResourceConfig("test-image-updated", "example.invalid/durantic/test-image:2.0", "an image", false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("registry_credential_uuid"), knownvalue.Null()),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("registry_credential_name"), knownvalue.Null()),
				},
			},
		},
	})
}

func testAccImageResourceConfig(name, dockerImageURL, description string, withCredential bool) string {
	credentialAttr := ""
	if withCredential {
		credentialAttr = "registry_credential_uuid = durantic_registry_credential.test.uuid"
	}

	return fmt.Sprintf(`
resource "durantic_registry_credential" "test" {
  name         = "test-image-registry"
  registry_url = "example.invalid"
  username     = "testuser"
  password     = "testpassword"
}

resource "durantic_image" "test" {
  name             = %[1]q
  docker_image_url = %[2]q
  description      = %[3]q
  %[4]s
}
`, name, dockerImageURL, description, credentialAttr)
}
