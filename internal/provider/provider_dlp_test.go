package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestDlpProviderRegistration(t *testing.T) {
	ctx := context.Background()
	p := New("1.0.0")

	var schemaResp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected provider schema diagnostics: %v", schemaResp.Diagnostics)
	}
	if _, ok := schemaResp.Schema.Attributes["dlp_host"]; !ok {
		t.Errorf("expected provider schema to contain dlp_host attribute")
	}
	if _, ok := schemaResp.Schema.Attributes["dlp_service_name"]; !ok {
		t.Errorf("expected provider schema to contain dlp_service_name attribute")
	}

	expectedResources := []string{
		"scm_dlp_tenant_config",
		"scm_dlp_ocr_enablement",
		"scm_dlp_alert_settings",
		"scm_dlp_data_pattern",
		"scm_dlp_dictionary",
		"scm_dlp_document_type",
		"scm_dlp_edm_dataset",
		"scm_dlp_data_profile",
		"scm_dlp_data_filtering_profile",
		"scm_dlp_endpoint_policy",
		"scm_dlp_peripheral",
		"scm_dlp_syslog_profile",
		"scm_dlp_forward_profile",
	}

	registeredResources := make(map[string]bool)
	for _, rf := range p.Resources(ctx) {
		r := rf()
		var metaResp resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "scm"}, &metaResp)
		if strings.HasPrefix(metaResp.TypeName, "scm_dlp_") {
			if registeredResources[metaResp.TypeName] {
				t.Errorf("duplicate DLP resource type name: %s", metaResp.TypeName)
			}
			registeredResources[metaResp.TypeName] = true

			var rSchemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &rSchemaResp)
			if rSchemaResp.Diagnostics.HasError() {
				t.Errorf("resource %s schema error: %v", metaResp.TypeName, rSchemaResp.Diagnostics)
			}
		}
	}

	for _, exp := range expectedResources {
		if !registeredResources[exp] {
			t.Errorf("expected DLP resource %q to be registered on ScmProvider", exp)
		}
	}

	expectedDataSources := []string{
		"scm_dlp_tenant_config",
		"scm_dlp_ocr_enablement",
		"scm_dlp_alert_settings",
		"scm_dlp_tenant_channels",
		"scm_dlp_data_pattern",
		"scm_dlp_data_pattern_list",
		"scm_dlp_dictionary",
		"scm_dlp_dictionary_list",
		"scm_dlp_document_type",
		"scm_dlp_document_type_list",
		"scm_dlp_edm_dataset",
		"scm_dlp_edm_dataset_list",
		"scm_dlp_data_profile",
		"scm_dlp_data_profile_list",
		"scm_dlp_data_filtering_profile",
		"scm_dlp_data_filtering_profile_list",
		"scm_dlp_endpoint_policy",
		"scm_dlp_endpoint_policy_list",
		"scm_dlp_peripheral",
		"scm_dlp_peripheral_list",
		"scm_dlp_syslog_profile",
		"scm_dlp_syslog_profile_list",
		"scm_dlp_forward_profile",
		"scm_dlp_forward_profile_list",
		"scm_dlp_config_export_job",
		"scm_dlp_config_import_job",
		"scm_dlp_dry_run_scan",
	}

	registeredDataSources := make(map[string]bool)
	for _, dsf := range p.DataSources(ctx) {
		ds := dsf()
		var metaResp datasource.MetadataResponse
		ds.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "scm"}, &metaResp)
		if strings.HasPrefix(metaResp.TypeName, "scm_dlp_") {
			if registeredDataSources[metaResp.TypeName] {
				t.Errorf("duplicate DLP data source type name: %s", metaResp.TypeName)
			}
			registeredDataSources[metaResp.TypeName] = true

			var dsSchemaResp datasource.SchemaResponse
			ds.Schema(ctx, datasource.SchemaRequest{}, &dsSchemaResp)
			if dsSchemaResp.Diagnostics.HasError() {
				t.Errorf("data source %s schema error: %v", metaResp.TypeName, dsSchemaResp.Diagnostics)
			}
		}
	}

	for _, exp := range expectedDataSources {
		if !registeredDataSources[exp] {
			t.Errorf("expected DLP data source %q to be registered on ScmProvider", exp)
		}
	}

	pa, ok := p.(provider.ProviderWithActions)
	if !ok {
		t.Fatalf("expected ScmProvider to implement ProviderWithActions")
	}

	expectedActions := []string{
		"scm_dlp_config_export",
		"scm_dlp_config_import",
		"scm_dlp_dry_run_scan",
	}
	registeredActions := make(map[string]bool)
	for _, af := range pa.Actions(ctx) {
		a := af()
		var metaResp action.MetadataResponse
		a.Metadata(ctx, action.MetadataRequest{ProviderTypeName: "scm"}, &metaResp)
		if strings.HasPrefix(metaResp.TypeName, "scm_dlp_") {
			registeredActions[metaResp.TypeName] = true
			var aSchemaResp action.SchemaResponse
			a.Schema(ctx, action.SchemaRequest{}, &aSchemaResp)
			if aSchemaResp.Diagnostics.HasError() {
				t.Errorf("action %s schema error: %v", metaResp.TypeName, aSchemaResp.Diagnostics)
			}
		}
	}
	for _, exp := range expectedActions {
		if !registeredActions[exp] {
			t.Errorf("expected DLP action %q to be registered on ScmProvider", exp)
		}
	}
}
