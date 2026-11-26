package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func getPrimitivesResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewDataPatternResource,
		NewDictionaryResource,
		NewDocumentTypeResource,
		NewEdmDatasetResource,
	}
}

func getPrimitivesDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDataPatternDataSource,
		NewDataPatternListDataSource,
		NewDictionaryDataSource,
		NewDictionaryListDataSource,
		NewDocumentTypeDataSource,
		NewDocumentTypeListDataSource,
		NewEdmDatasetDataSource,
		NewEdmDatasetListDataSource,
	}
}
