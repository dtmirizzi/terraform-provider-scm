package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/action"
)

// GetActions returns all actions for the dlp package.
func GetActions() []func() action.Action {
	var out []func() action.Action
	out = append(out, getPoliciesActions()...)
	return out
}
