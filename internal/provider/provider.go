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
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apiprof "github.com/sacloud/api-client-go/profile"
)

type sakuraProviderModel struct {
	Profile             types.String `tfsdk:"profile"`
	AccessToken         types.String `tfsdk:"token"`
	AccessTokenSecret   types.String `tfsdk:"secret"`
	Zone                types.String `tfsdk:"zone"`
	Zones               types.List   `tfsdk:"zones"`
	DefaultZone         types.String `tfsdk:"default_zone"`
	APIRootURL          types.String `tfsdk:"api_root_url"`
	RetryMax            types.Int64  `tfsdk:"retry_max"`
	RetryWaitMax        types.Int64  `tfsdk:"retry_wait_max"`
	RetryWaitMin        types.Int64  `tfsdk:"retry_wait_min"`
	APIRequestTimeout   types.Int64  `tfsdk:"api_request_timeout"`
	APIRequestRateLimit types.Int64  `tfsdk:"api_request_rate_limit"`
	TraceMode           types.String `tfsdk:"trace"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &sakuraProvider{version: version}
	}
}

type sakuraProvider struct {
	version string
	client  *APIClient
}

func (p *sakuraProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sakura"
	resp.Version = p.version
}

func (p *sakuraProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"profile": schema.StringAttribute{Optional: true},
			"token":   schema.StringAttribute{Optional: true},
			"secret":  schema.StringAttribute{Optional: true},
			"zone":    schema.StringAttribute{Optional: true},
			"zones": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"default_zone":           schema.StringAttribute{Optional: true},
			"api_root_url":           schema.StringAttribute{Optional: true},
			"retry_max":              schema.Int64Attribute{Optional: true},
			"retry_wait_max":         schema.Int64Attribute{Optional: true},
			"retry_wait_min":         schema.Int64Attribute{Optional: true},
			"api_request_timeout":    schema.Int64Attribute{Optional: true},
			"api_request_rate_limit": schema.Int64Attribute{Optional: true},
			"trace":                  schema.StringAttribute{Optional: true},
		},
	}
}

func getIntValueFromEnv(resp *provider.ConfigureResponse, envVar string, defaultValue int) int {
	valueStr, ok := os.LookupEnv(envVar)
	if !ok {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error parsing environment variable %q", envVar), err.Error())
		return defaultValue
	}
	return value
}

func (p *sakuraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

	profile, ok := os.LookupEnv("SAKURACLOUD_PROFILE")
	if !ok {
		profile = apiprof.DefaultProfileName
	}
	token := os.Getenv("SAKURACLOUD_ACCESS_TOKEN")
	secret := os.Getenv("SAKURACLOUD_ACCESS_TOKEN_SECRET")
	zone, ok := os.LookupEnv("SAKURACLOUD_ZONE")
	if !ok {
		zone = Zone
	}
	defaultZone, _ := os.LookupEnv("SAKURACLOUD_DEFAULT_ZONE")
	apiRootUrl := os.Getenv("SAKURACLOUD_API_ROOT_URL")
	retryMax := getIntValueFromEnv(resp, "SAKURACLOUD_RETRY_MAX", RetryMax)
	retryWaitMax := getIntValueFromEnv(resp, "SAKURACLOUD_RETRY_WAIT_MAX", 0)
	retryWaitMin := getIntValueFromEnv(resp, "SAKURACLOUD_RETRY_WAIT_MIN", 0)
	apiRequestTimeout := getIntValueFromEnv(resp, "SAKURACLOUD_API_REQUEST_TIMEOUT", APIRequestTimeout)
	apiRequestRateLimit := getIntValueFromEnv(resp, "SAKURACLOUD_RATE_LIMIT", APIRequestRateLimit)

	var config sakuraProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Plugin Frameworkの設定値が最優先
	if config.Profile.ValueString() != "" {
		profile = config.Profile.ValueString()
	}
	if config.AccessToken.ValueString() != "" {
		token = config.AccessToken.ValueString()
	}
	if config.AccessTokenSecret.ValueString() != "" {
		secret = config.AccessTokenSecret.ValueString()
	}
	if config.Zone.ValueString() != "" {
		zone = config.Zone.ValueString()
	}
	if config.DefaultZone.ValueString() != "" {
		defaultZone = config.DefaultZone.ValueString()
	}
	if config.APIRootURL.ValueString() != "" {
		apiRootUrl = config.APIRootURL.ValueString()
	}
	if !config.RetryMax.IsNull() && !config.RetryMax.IsUnknown() {
		retryMax = int(config.RetryMax.ValueInt64())
	}
	if !config.RetryWaitMax.IsNull() && !config.RetryWaitMax.IsUnknown() {
		retryWaitMax = int(config.RetryWaitMax.ValueInt64())
	}
	if !config.RetryWaitMin.IsNull() && !config.RetryWaitMin.IsUnknown() {
		retryWaitMin = int(config.RetryWaitMin.ValueInt64())
	}
	if !config.APIRequestTimeout.IsNull() && !config.APIRequestTimeout.IsUnknown() {
		apiRequestTimeout = int(config.APIRequestTimeout.ValueInt64())
	}
	if !config.APIRequestRateLimit.IsNull() && !config.APIRequestRateLimit.IsUnknown() {
		apiRequestRateLimit = int(config.APIRequestRateLimit.ValueInt64())
	}
	zones := []string{}
	if !config.Zones.IsNull() && !config.Zones.IsUnknown() {
		for _, v := range config.Zones.Elements() {
			zones = append(zones, v.(types.String).ValueString())
		}
	}

	cfg := Config{
		Profile:             profile,
		AccessToken:         token,
		AccessTokenSecret:   secret,
		Zone:                zone,
		Zones:               zones,
		DefaultZone:         defaultZone,
		TraceMode:           config.TraceMode.ValueString(),
		APIRootURL:          apiRootUrl,
		RetryMax:            retryMax,
		RetryWaitMax:        retryWaitMax,
		RetryWaitMin:        retryWaitMin,
		APIRequestTimeout:   apiRequestTimeout,
		APIRequestRateLimit: apiRequestRateLimit,
		TerraformVersion:    req.TerraformVersion,
	}

	client, err := cfg.NewClient()
	if err != nil {
		resp.Diagnostics.AddError("Error creating Sakura client", err.Error())
		return
	}

	p.client = client
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *sakuraProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewKmsDataSource,
		NewSecretManagerDataSource,
		NewSecretManagerSecretDataSource,
		// ...他のデータソースも同様に追加...
	}
}

func (p *sakuraProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewKMSResource,
		NewSecretManagerResource,
		NewSecretManagerSecretResource,
		// ...他のリソースも同様に追加...
	}
}
