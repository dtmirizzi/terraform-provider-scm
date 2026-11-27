package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func assertBetaWarning(t *testing.T, kind string, diags diag.Diagnostics) {
	t.Helper()
	if diags.HasError() || diags.WarningsCount() != 1 || diags.Warnings()[0].Summary() != BetaWarningSummary {
		t.Errorf("%s: expected exactly one %q warning, got %v", kind, BetaWarningSummary, diags)
	}
}

// Every registered scm_dlp_* type must surface the Beta / Experimental warning.
func TestBetaWarningOnAllDLPTypes(t *testing.T) {
	ctx := context.Background()

	for _, f := range GetResources() {
		r := f()
		var md resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "scm"}, &md)
		v, ok := r.(resource.ResourceWithValidateConfig)
		if !ok {
			t.Errorf("resource %s does not implement ValidateConfig (beta warning)", md.TypeName)
			continue
		}
		var resp resource.ValidateConfigResponse
		v.ValidateConfig(ctx, resource.ValidateConfigRequest{}, &resp)
		assertBetaWarning(t, md.TypeName, resp.Diagnostics)
	}

	for _, f := range GetDataSources() {
		d := f()
		var md datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "scm"}, &md)
		v, ok := d.(datasource.DataSourceWithValidateConfig)
		if !ok {
			t.Errorf("data source %s does not implement ValidateConfig (beta warning)", md.TypeName)
			continue
		}
		var resp datasource.ValidateConfigResponse
		v.ValidateConfig(ctx, datasource.ValidateConfigRequest{}, &resp)
		assertBetaWarning(t, md.TypeName, resp.Diagnostics)
	}

	for _, f := range GetActions() {
		a := f()
		var md action.MetadataResponse
		a.Metadata(ctx, action.MetadataRequest{ProviderTypeName: "scm"}, &md)
		v, ok := a.(action.ActionWithValidateConfig)
		if !ok {
			t.Errorf("action %s does not implement ValidateConfig (beta warning)", md.TypeName)
			continue
		}
		var resp action.ValidateConfigResponse
		v.ValidateConfig(ctx, action.ValidateConfigRequest{}, &resp)
		assertBetaWarning(t, md.TypeName, resp.Diagnostics)
	}
}
