package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func getTenantConfigResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewDlpTenantConfigResource,
		NewDlpOcrEnablementResource,
		NewDlpAlertSettingsResource,
	}
}

func getTenantConfigDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDlpTenantConfigDataSource,
		NewDlpOcrEnablementDataSource,
		NewDlpAlertSettingsDataSource,
		NewDlpTenantChannelsDataSource,
	}
}
