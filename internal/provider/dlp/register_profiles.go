package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func getProfilesResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewDataProfileResource,
		NewDataFilteringProfileResource,
	}
}

func getProfilesDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataProfileDataSource,
		NewDataProfileListDataSource,
		NewDataFilteringProfileDataSource,
		NewDataFilteringProfileListDataSource,
	}
}
