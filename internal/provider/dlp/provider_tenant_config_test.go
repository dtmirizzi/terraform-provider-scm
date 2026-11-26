package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/dlpclient"
	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
)

// setupMockServer sets up an httptest server simulating DLP tenant config endpoints.
func setupMockServer(t *testing.T) (*httptest.Server, *sync.Mutex, map[string]any) {
	mu := &sync.Mutex{}
	store := map[string]any{
		"tenant_config": map[string]any{
			"id":                               "tenant-123",
			"enable_snippets":                  false,
			"mask_level":                       "NO_MASK",
			"incident_notification_enablement": false,
			"incident_notification_snippets_enablement":     false,
			"detection_exemption_period_minutes":            int64(30),
			"enable_structured_data_clustering":             false,
			"enforce_structured_data_clustering_uniformity": false,
			"structured_data_clustering_minimum_uniformity": int64(80),
			"edm_enablement":        "DISABLED",
			"edm_enablement_banner": false,
			"ocr_enablement":        "DISABLED",
			"ocr_enablement_banner": false,
		},
		"ocr_enablement": map[string]map[string]any{
			"prisma-access": {
				"target_service_name": "prisma-access",
				"ocr_enablement":      "DISABLED",
				"tenant_id":           "tenant-123",
			},
		},
		"alert_settings": map[string]any{
			"incident_notification_enablement":   false,
			"detection_exemption_period_minutes": int64(30),
		},
		"channels": map[string]any{
			"prisma-access": true,
			"ngfw":          true,
			"email-dlp":     false,
		},
		"edm_enable_called": false,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/v1/api/tenant-common-config/properties" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(store["tenant_config"])

		case r.URL.Path == "/v1/api/tenant-common-config/update" && r.Method == http.MethodPatch:
			var body map[string]any
			bodyBytes, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(bodyBytes, &body)
			tc := store["tenant_config"].(map[string]any)
			for k, v := range body {
				tc[k] = v
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "success"})

		case r.URL.Path == "/v1/api/edm-dataset/enable" && r.Method == http.MethodPost:
			store["edm_enable_called"] = true
			tc := store["tenant_config"].(map[string]any)
			tc["edm_enablement"] = "REQUESTED"
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "enabled"})

		case r.URL.Path == "/v2/api/ocr" && r.Method == http.MethodGet:
			svc := r.URL.Query().Get("target-service-name")
			ocrMap := store["ocr_enablement"].(map[string]map[string]any)
			if item, ok := ocrMap[svc]; ok {
				_ = json.NewEncoder(w).Encode(item)
			} else {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": "OCR enablement not found for target service",
				})
			}

		case r.URL.Path == "/v2/api/ocr" && r.Method == http.MethodPut:
			var body map[string]any
			bodyBytes, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(bodyBytes, &body)
			svc, _ := body["target_service_name"].(string)
			ocrState, _ := body["ocr_enablement"].(string)
			ocrMap := store["ocr_enablement"].(map[string]map[string]any)
			ocrMap[svc] = map[string]any{
				"target_service_name": svc,
				"ocr_enablement":      ocrState,
				"tenant_id":           "tenant-123",
			}
			_ = json.NewEncoder(w).Encode(ocrMap[svc])

		case r.URL.Path == "/v1/api/alerts/setting" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(store["alert_settings"])

		case r.URL.Path == "/v1/api/alerts/setting" && r.Method == http.MethodPut:
			var body map[string]any
			bodyBytes, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(bodyBytes, &body)
			as := store["alert_settings"].(map[string]any)
			for k, v := range body {
				as[k] = v
			}
			_ = json.NewEncoder(w).Encode(as)

		case r.URL.Path == "/v1/api/channels" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(store["channels"])

		default:
			http.NotFound(w, r)
		}
	})

	srv := httptest.NewServer(handler)
	return srv, mu, store
}

func testProviderData(srv *httptest.Server) map[string]any {
	client := dlpclient.NewClient(dlpclient.Config{
		HttpClient: srv.Client(),
		Host:       srv.URL,
	})
	return map[string]any{
		"dlp": client,
	}
}

// TestTenantConfigResourceCRUD tests Create, Read, Update, and DataSource for scm_dlp_tenant_config.
func TestTenantConfigResourceCRUD(t *testing.T) {
	srv, mu, store := setupMockServer(t)
	defer srv.Close()

	ctx := context.Background()
	providerData := testProviderData(srv)

	res := NewDlpTenantConfigResource().(resource.ResourceWithConfigure)
	var configResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: providerData}, &configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("configure failed: %v", configResp.Diagnostics)
	}

	// 1. CREATE test with enable_snippets=true, mask_level="PARTIAL_MASK", edm_enablement="REQUESTED"
	planModel := models.DlpTenantConfigModel{
		EnableSnippets:                         types.BoolValue(true),
		MaskLevel:                              types.StringValue("PARTIAL_MASK"),
		IncidentNotificationEnablement:         types.BoolValue(true),
		IncidentNotificationSnippetsEnablement: types.BoolValue(true),
		DetectionExemptionPeriodMinutes:        types.Int64Value(60),
		EnableStructuredDataClustering:         types.BoolValue(true),
		EnforceStructuredDataClusteringUniform: types.BoolValue(false),
		StructuredDataClusteringMinUniformity:  types.Int64Value(75),
		EdmEnablement:                          types.StringValue("REQUESTED"),
		EdmEnablementBanner:                    types.BoolValue(false),
		OcrEnablement:                          types.StringValue("ENABLED"),
		OcrEnablementBanner:                    types.BoolValue(false),
	}

	planVal := tfsdk.Plan{Schema: models.DlpTenantConfigResourceSchema}
	if diags := planVal.Set(ctx, &planModel); diags.HasError() {
		t.Fatalf("planVal.Set failed: %v", diags)
	}

	stateVal := tfsdk.State{Schema: models.DlpTenantConfigResourceSchema}

	createReq := resource.CreateRequest{Plan: planVal}
	createResp := resource.CreateResponse{State: stateVal}
	res.Create(ctx, createReq, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", createResp.Diagnostics)
	}

	var stateAfterCreate models.DlpTenantConfigModel
	diags := createResp.State.Get(ctx, &stateAfterCreate)
	if diags.HasError() {
		t.Fatalf("failed to get state after create: %v", diags)
	}

	if !stateAfterCreate.EnableSnippets.ValueBool() {
		t.Errorf("expected enable_snippets=true, got false")
	}
	if stateAfterCreate.MaskLevel.ValueString() != "PARTIAL_MASK" {
		t.Errorf("expected mask_level=PARTIAL_MASK, got %s", stateAfterCreate.MaskLevel.ValueString())
	}
	if stateAfterCreate.EdmEnablement.ValueString() != "REQUESTED" {
		t.Errorf("expected edm_enablement=REQUESTED, got %s", stateAfterCreate.EdmEnablement.ValueString())
	}
	if stateAfterCreate.Id.ValueString() != "tenant-123" {
		t.Errorf("expected id=tenant-123, got %s", stateAfterCreate.Id.ValueString())
	}

	mu.Lock()
	edmCalled := store["edm_enable_called"].(bool)
	mu.Unlock()
	if !edmCalled {
		t.Errorf("expected edm enable endpoint to be called when edm_enablement=REQUESTED")
	}

	// 2. READ test
	readReq := resource.ReadRequest{State: createResp.State}
	readResp := resource.ReadResponse{State: createResp.State}
	res.Read(ctx, readReq, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", readResp.Diagnostics)
	}

	var stateAfterRead models.DlpTenantConfigModel
	_ = readResp.State.Get(ctx, &stateAfterRead)
	if stateAfterRead.MaskLevel.ValueString() != "PARTIAL_MASK" {
		t.Errorf("expected mask_level=PARTIAL_MASK on read, got %s", stateAfterRead.MaskLevel.ValueString())
	}

	// 3. UPDATE test
	planModel.MaskLevel = types.StringValue("FULL_MASK")
	if diags := planVal.Set(ctx, &planModel); diags.HasError() {
		t.Fatalf("planVal.Set update failed: %v", diags)
	}

	updateReq := resource.UpdateRequest{
		Plan:  planVal,
		State: readResp.State,
	}
	updateResp := resource.UpdateResponse{State: readResp.State}
	res.Update(ctx, updateReq, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", updateResp.Diagnostics)
	}

	var stateAfterUpdate models.DlpTenantConfigModel
	_ = updateResp.State.Get(ctx, &stateAfterUpdate)
	if stateAfterUpdate.MaskLevel.ValueString() != "FULL_MASK" {
		t.Errorf("expected mask_level=FULL_MASK after update, got %s", stateAfterUpdate.MaskLevel.ValueString())
	}

	// 4. DATA SOURCE test
	ds := NewDlpTenantConfigDataSource().(datasource.DataSourceWithConfigure)
	var dsConfigResp datasource.ConfigureResponse
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: providerData}, &dsConfigResp)
	if dsConfigResp.Diagnostics.HasError() {
		t.Fatalf("data source configure failed: %v", dsConfigResp.Diagnostics)
	}

	dsStateVal := tfsdk.State{Schema: models.DlpTenantConfigDataSourceSchema}
	dsReadReq := datasource.ReadRequest{}
	dsReadResp := datasource.ReadResponse{State: dsStateVal}
	ds.Read(ctx, dsReadReq, &dsReadResp)
	if dsReadResp.Diagnostics.HasError() {
		t.Fatalf("data source Read failed: %v", dsReadResp.Diagnostics)
	}

	var dsResult models.DlpTenantConfigModel
	_ = dsReadResp.State.Get(ctx, &dsResult)
	if dsResult.MaskLevel.ValueString() != "FULL_MASK" {
		t.Errorf("expected data source mask_level=FULL_MASK, got %s", dsResult.MaskLevel.ValueString())
	}
	if !dsResult.EnableSnippets.ValueBool() {
		t.Errorf("expected data source enable_snippets=true")
	}
}

// TestOcrEnablementCRUD tests Create, Read, Update, Delete, and DataSource for scm_dlp_ocr_enablement.
func TestOcrEnablementCRUD(t *testing.T) {
	srv, _, _ := setupMockServer(t)
	defer srv.Close()

	ctx := context.Background()
	providerData := testProviderData(srv)

	res := NewDlpOcrEnablementResource().(resource.ResourceWithConfigure)
	var configResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: providerData}, &configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("ocr resource configure failed: %v", configResp.Diagnostics)
	}

	planModel := models.DlpOcrEnablementModel{
		TargetServiceName: types.StringValue("prisma-access"),
		OcrEnablement:     types.StringValue("ENABLED"),
	}
	planVal := tfsdk.Plan{Schema: models.DlpOcrEnablementResourceSchema}
	if diags := planVal.Set(ctx, &planModel); diags.HasError() {
		t.Fatalf("planVal.Set failed: %v", diags)
	}

	stateVal := tfsdk.State{Schema: models.DlpOcrEnablementResourceSchema}

	// 1. Create
	createReq := resource.CreateRequest{Plan: planVal}
	createResp := resource.CreateResponse{State: stateVal}
	res.Create(ctx, createReq, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("ocr Create failed: %v", createResp.Diagnostics)
	}

	var stateAfterCreate models.DlpOcrEnablementModel
	_ = createResp.State.Get(ctx, &stateAfterCreate)
	if stateAfterCreate.OcrEnablement.ValueString() != "ENABLED" {
		t.Errorf("expected ocr_enablement=ENABLED, got %s", stateAfterCreate.OcrEnablement.ValueString())
	}
	if stateAfterCreate.TargetServiceName.ValueString() != "prisma-access" {
		t.Errorf("expected target_service_name=prisma-access, got %s", stateAfterCreate.TargetServiceName.ValueString())
	}

	// 2. Read
	readReq := resource.ReadRequest{State: createResp.State}
	readResp := resource.ReadResponse{State: createResp.State}
	res.Read(ctx, readReq, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("ocr Read failed: %v", readResp.Diagnostics)
	}

	// 3. Update to DISABLED
	planModel.OcrEnablement = types.StringValue("DISABLED")
	if diags := planVal.Set(ctx, &planModel); diags.HasError() {
		t.Fatalf("planVal.Set update failed: %v", diags)
	}
	updateReq := resource.UpdateRequest{Plan: planVal, State: readResp.State}
	updateResp := resource.UpdateResponse{State: readResp.State}
	res.Update(ctx, updateReq, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("ocr Update failed: %v", updateResp.Diagnostics)
	}
	var stateAfterUpdate models.DlpOcrEnablementModel
	_ = updateResp.State.Get(ctx, &stateAfterUpdate)
	if stateAfterUpdate.OcrEnablement.ValueString() != "DISABLED" {
		t.Errorf("expected ocr_enablement=DISABLED after update, got %s", stateAfterUpdate.OcrEnablement.ValueString())
	}

	// 4. Data Source
	ds := NewDlpOcrEnablementDataSource().(datasource.DataSourceWithConfigure)
	var dsConfigResp datasource.ConfigureResponse
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: providerData}, &dsConfigResp)
	if dsConfigResp.Diagnostics.HasError() {
		t.Fatalf("ocr ds configure failed: %v", dsConfigResp.Diagnostics)
	}

	cfgModel := models.DlpOcrEnablementModel{
		TargetServiceName: types.StringValue("prisma-access"),
	}
	var cfgState tfsdk.State
	cfgState.Schema = models.DlpOcrEnablementDataSourceSchema
	_ = cfgState.Set(ctx, &cfgModel)
	dsCfgVal := tfsdk.Config{
		Raw:    cfgState.Raw,
		Schema: models.DlpOcrEnablementDataSourceSchema,
	}

	dsStateVal := tfsdk.State{Schema: models.DlpOcrEnablementDataSourceSchema}
	dsReadReq := datasource.ReadRequest{Config: dsCfgVal}
	dsReadResp := datasource.ReadResponse{State: dsStateVal}
	ds.Read(ctx, dsReadReq, &dsReadResp)
	if dsReadResp.Diagnostics.HasError() {
		t.Fatalf("ocr ds Read failed: %v", dsReadResp.Diagnostics)
	}

	var dsResult models.DlpOcrEnablementModel
	_ = dsReadResp.State.Get(ctx, &dsResult)
	if dsResult.OcrEnablement.ValueString() != "DISABLED" {
		t.Errorf("expected ds ocr_enablement=DISABLED, got %s", dsResult.OcrEnablement.ValueString())
	}
}

// TestAlertSettingsAndChannelsCRUD tests scm_dlp_alert_settings and scm_dlp_tenant_channels.
func TestAlertSettingsAndChannelsCRUD(t *testing.T) {
	srv, _, _ := setupMockServer(t)
	defer srv.Close()

	ctx := context.Background()
	providerData := testProviderData(srv)

	// 1. Alert settings resource
	res := NewDlpAlertSettingsResource().(resource.ResourceWithConfigure)
	var configResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: providerData}, &configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("alert settings configure failed: %v", configResp.Diagnostics)
	}

	planModel := models.DlpAlertSettingsModel{
		IncidentNotificationEnablement:  types.BoolValue(true),
		DetectionExemptionPeriodMinutes: types.Int64Value(120),
	}
	planVal := tfsdk.Plan{Schema: models.DlpAlertSettingsResourceSchema}
	if diags := planVal.Set(ctx, &planModel); diags.HasError() {
		t.Fatalf("planVal.Set failed: %v", diags)
	}

	stateVal := tfsdk.State{Schema: models.DlpAlertSettingsResourceSchema}

	createReq := resource.CreateRequest{Plan: planVal}
	createResp := resource.CreateResponse{State: stateVal}
	res.Create(ctx, createReq, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("alert settings Create failed: %v", createResp.Diagnostics)
	}

	var stateAfterCreate models.DlpAlertSettingsModel
	_ = createResp.State.Get(ctx, &stateAfterCreate)
	if !stateAfterCreate.IncidentNotificationEnablement.ValueBool() {
		t.Errorf("expected incident_notification_enablement=true, got false")
	}
	if stateAfterCreate.DetectionExemptionPeriodMinutes.ValueInt64() != 120 {
		t.Errorf("expected detection_exemption_period_minutes=120, got %d", stateAfterCreate.DetectionExemptionPeriodMinutes.ValueInt64())
	}

	// 2. Alert settings data source
	dsAlert := NewDlpAlertSettingsDataSource().(datasource.DataSourceWithConfigure)
	var dsAlertCfgResp datasource.ConfigureResponse
	dsAlert.Configure(ctx, datasource.ConfigureRequest{ProviderData: providerData}, &dsAlertCfgResp)
	if dsAlertCfgResp.Diagnostics.HasError() {
		t.Fatalf("alert ds configure failed: %v", dsAlertCfgResp.Diagnostics)
	}

	dsAlertStateVal := tfsdk.State{Schema: models.DlpAlertSettingsDataSourceSchema}
	dsAlertReadResp := datasource.ReadResponse{State: dsAlertStateVal}
	dsAlert.Read(ctx, datasource.ReadRequest{}, &dsAlertReadResp)
	if dsAlertReadResp.Diagnostics.HasError() {
		t.Fatalf("alert ds Read failed: %v", dsAlertReadResp.Diagnostics)
	}

	var dsAlertResult models.DlpAlertSettingsModel
	_ = dsAlertReadResp.State.Get(ctx, &dsAlertResult)
	if !dsAlertResult.IncidentNotificationEnablement.ValueBool() {
		t.Errorf("expected ds incident_notification_enablement=true")
	}

	// 3. Tenant channels data source
	dsChan := NewDlpTenantChannelsDataSource().(datasource.DataSourceWithConfigure)
	var dsChanCfgResp datasource.ConfigureResponse
	dsChan.Configure(ctx, datasource.ConfigureRequest{ProviderData: providerData}, &dsChanCfgResp)
	if dsChanCfgResp.Diagnostics.HasError() {
		t.Fatalf("channels ds configure failed: %v", dsChanCfgResp.Diagnostics)
	}

	dsChanStateVal := tfsdk.State{Schema: models.DlpTenantChannelsDataSourceSchema}
	dsChanReadResp := datasource.ReadResponse{State: dsChanStateVal}
	dsChan.Read(ctx, datasource.ReadRequest{}, &dsChanReadResp)
	if dsChanReadResp.Diagnostics.HasError() {
		t.Fatalf("channels ds Read failed: %v", dsChanReadResp.Diagnostics)
	}

	var chanResult models.DlpTenantChannelsModel
	_ = dsChanReadResp.State.Get(ctx, &chanResult)
	elems := chanResult.Channels.Elements()
	if len(elems) != 3 {
		t.Errorf("expected 3 channels, got %d", len(elems))
	}
	if b, ok := elems["prisma-access"].(types.Bool); !ok || !b.ValueBool() {
		t.Errorf("expected prisma-access channel=true")
	}
	if b, ok := elems["email-dlp"].(types.Bool); !ok || b.ValueBool() {
		t.Errorf("expected email-dlp channel=false")
	}

	// 4. Test OCR enablement not found handling
	dsOcr := NewDlpOcrEnablementDataSource().(datasource.DataSourceWithConfigure)
	var dsOcrCfgResp datasource.ConfigureResponse
	dsOcr.Configure(ctx, datasource.ConfigureRequest{ProviderData: providerData}, &dsOcrCfgResp)
	if dsOcrCfgResp.Diagnostics.HasError() {
		t.Fatalf("ocr ds configure failed: %v", dsOcrCfgResp.Diagnostics)
	}

	dsNotFoundCfg := models.DlpOcrEnablementModel{
		TargetServiceName: types.StringValue("ngfw"),
	}
	var notFoundState tfsdk.State
	notFoundState.Schema = models.DlpOcrEnablementDataSourceSchema
	_ = notFoundState.Set(ctx, &dsNotFoundCfg)
	notFoundCfgVal := tfsdk.Config{
		Raw:    notFoundState.Raw,
		Schema: models.DlpOcrEnablementDataSourceSchema,
	}
	notFoundDsResp := datasource.ReadResponse{State: tfsdk.State{Schema: models.DlpOcrEnablementDataSourceSchema}}
	dsOcr.Read(ctx, datasource.ReadRequest{Config: notFoundCfgVal}, &notFoundDsResp)
	if !notFoundDsResp.Diagnostics.HasError() {
		t.Errorf("expected error diagnostic when target_service_name not found in OCR list, got none")
	}
}
