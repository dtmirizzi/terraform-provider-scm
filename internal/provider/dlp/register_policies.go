package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func getPoliciesResources() []func() resource.Resource {
	return []func() resource.Resource{
		NewEndpointPolicyResource,
		NewPeripheralResource,
		NewSyslogProfileResource,
		NewForwardProfileResource,
	}
}

func getPoliciesDataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewEndpointPolicyDataSource,
		NewEndpointPolicyListDataSource,
		NewPeripheralDataSource,
		NewPeripheralListDataSource,
		NewSyslogProfileDataSource,
		NewSyslogProfileListDataSource,
		NewForwardProfileDataSource,
		NewForwardProfileListDataSource,
		NewDryRunScanDataSource,
	}
}

func getPoliciesActions() []func() action.Action {
	return []func() action.Action{
		NewDryRunScanAction,
	}
}
