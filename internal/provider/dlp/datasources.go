package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// GetDataSources returns all data sources for the dlp package.
func GetDataSources() []func() datasource.DataSource {
	var out []func() datasource.DataSource
	out = append(out, getTenantConfigDataSources()...)
	out = append(out, getPrimitivesDataSources()...)
	out = append(out, getProfilesDataSources()...)
	out = append(out, getPoliciesDataSources()...)
	return out
}
