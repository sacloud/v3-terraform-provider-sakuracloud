// Copyright 2016-2025 terraform-provider-sakuracloud authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sakura

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	v1 "github.com/sacloud/secretmanager-api-go/apis/v1"
)

func TestAccSakuraDataSourceSecretManager_basic(t *testing.T) {
	resourceName := "data.sakura_secretmanager.foobar"
	rand := randomName()

	var vault v1.Vault
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: buildConfigWithArgs(testAccSakuraDataSourceSecretManager_byName, rand),
				Check: resource.ComposeTestCheckFunc(
					testCheckSakuraSecretManagerExists("sakura_secretmanager.foobar", &vault),
					testCheckSakuraDataSourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", rand),
					resource.TestCheckResourceAttr(resourceName, "description", "description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "tag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "tag2"),
					// 綺麗に動的にkms_key_idをテストで取得する方法があればコメントアウト
					// resource.TestCheckResourceAttr(resourceName, "kms_key_id", vault.KmsKeyID),
				),
			},
			{
				Config: buildConfigWithArgs(testAccSakuraDataSourceSecretManager_byResourceId, rand),
				Check: resource.ComposeTestCheckFunc(
					testCheckSakuraSecretManagerExists("sakura_secretmanager.foobar", &vault),
					testCheckSakuraDataSourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", rand),
					resource.TestCheckResourceAttr(resourceName, "description", "description"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.0", "tag1"),
					resource.TestCheckResourceAttr(resourceName, "tags.1", "tag2"),
					// resource.TestCheckResourceAttr(resourceName, "kms_key_id", vault.KmsKeyID),
				),
			},
		},
	})
}

//nolint:gosec
var testAccSakuraDataSourceSecretManager_byName = `
resource "sakura_kms" "foobar" {
  name        = "{{ .arg0 }}"
  description = "description"
  tags        = ["tag1", "tag2"]
}

resource "sakura_secretmanager" "foobar" {
  name        = "{{ .arg0 }}"
  description = "description"
  tags        = ["tag1", "tag2"]
  kms_key_id  = sakura_kms.foobar.id

  depends_on = [sakura_kms.foobar]
}

data "sakura_secretmanager" "foobar" {
  name = "{{ .arg0 }}"

  depends_on = [sakura_secretmanager.foobar]
}`

//nolint:gosec
var testAccSakuraDataSourceSecretManager_byResourceId = `
resource "sakura_kms" "foobar" {
  name        = "{{ .arg0 }}"
  description = "description"
  tags        = ["tag1", "tag2"]
}

resource "sakura_secretmanager" "foobar" {
  name        = "{{ .arg0 }}"
  description = "description"
  tags        = ["tag1", "tag2"]
  kms_key_id  = sakura_kms.foobar.id

  depends_on = [sakura_kms.foobar]
}

data "sakura_secretmanager" "foobar" {
  resource_id = sakura_secretmanager.foobar.id

  depends_on = [sakura_secretmanager.foobar]
}`

func TestFilterSecretManagerByName(t *testing.T) {
	t.Parallel()

	vaults := []v1.Vault{
		{
			Name: "test-key1",
			Tags: []string{"tag1"},
		},
		{
			Name: "test-key2",
			Tags: []string{"tag1", "tag2"},
		},
	}

	testCases := []struct {
		name    string
		keyName string
		want    *v1.Vault
		wantErr bool
	}{
		{
			name:    "found by name",
			keyName: "test-key1",
			want:    &vaults[0],
		},
		{
			name:    "not found",
			keyName: "not-exist",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := filterSecretManagerVaultByName(vaults, tc.keyName)
			if tc.wantErr && err == nil {
				t.Errorf("filterSecretManagerByName wants error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("filterSecretManagerByName error = %v", err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("filterSecretManagerByName got = %v, want %v", got, tc.want)
			}
		})
	}
}
