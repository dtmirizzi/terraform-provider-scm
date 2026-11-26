package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// GetResources returns all resources for the dlp package.
func GetResources() []func() resource.Resource {
	var out []func() resource.Resource
	out = append(out, getTenantConfigResources()...)
	out = append(out, getPrimitivesResources()...)
	out = append(out, getProfilesResources()...)
	out = append(out, getPoliciesResources()...)
	return out
}
