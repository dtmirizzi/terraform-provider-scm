package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/dlpclient"
	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
)

// setupProfilesMockServer mocks /v2/api/data-profiles and /v2/api/data-filtering-profiles
func setupProfilesMockServer(t *testing.T) (*httptest.Server, *sync.Mutex, map[string]any) {
	mu := &sync.Mutex{}
	store := map[string]any{
		"profiles":           map[string]dlpclient.DataProfileDTO{},
		"filtering_profiles": map[string]dlpclient.DataFilteringProfileDTO{},
		"next_id":            100,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")

		// -------------------------------------------------------------------
		// /v2/api/data-profiles
		// -------------------------------------------------------------------
		if r.URL.Path == "/v2/api/data-profiles" {
			switch r.Method {
			case http.MethodGet:
				nameFilter := r.URL.Query().Get("name")
				profilesMap := store["profiles"].(map[string]dlpclient.DataProfileDTO)
				var list []dlpclient.DataProfileDTO
				for _, p := range profilesMap {
					if nameFilter != "" && !strings.EqualFold(p.Name, nameFilter) {
						continue
					}
					list = append(list, p)
				}
				envelope := map[string]any{
					"content":       list,
					"totalElements": len(list),
					"totalPages":    1,
					"last":          true,
				}
				_ = json.NewEncoder(w).Encode(envelope)
				return

			case http.MethodPost:
				var dto dlpclient.DataProfileDTO
				if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				nextID := store["next_id"].(int)
				store["next_id"] = nextID + 1
				idStr := strconv.Itoa(nextID)
				dto.ID = &idStr
				v := int64(1)
				dto.Version = &v

				profilesMap := store["profiles"].(map[string]dlpclient.DataProfileDTO)
				profilesMap[idStr] = dto

				// Auto-create companion filtering profile as DLP backend does
				filteringMap := store["filtering_profiles"].(map[string]dlpclient.DataFilteringProfileDTO)
				dpid := int64(nextID)
				fpName := dto.Name + " Filtering"
				dir := "both"
				st := "include"
				ls := "HIGH"
				filteringMap[idStr] = dlpclient.DataFilteringProfileDTO{
					ID:            &idStr,
					DataProfileID: &dpid,
					Name:          &fpName,
					Direction:     &dir,
					ScanType:      &st,
					LogSeverity:   &ls,
					FileBased:     true,
					NonFileBased:  true,
				}

				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(dto)
				return
			}
		}

		if strings.HasPrefix(r.URL.Path, "/v2/api/data-profiles/") {
			id := strings.TrimPrefix(r.URL.Path, "/v2/api/data-profiles/")
			profilesMap := store["profiles"].(map[string]dlpclient.DataProfileDTO)

			switch r.Method {
			case http.MethodGet:
				p, ok := profilesMap[id]
				if !ok {
					http.Error(w, `{"error":"profile not found"}`, http.StatusNotFound)
					return
				}
				_ = json.NewEncoder(w).Encode(p)
				return

			case http.MethodPut:
				var dto dlpclient.DataProfileDTO
				if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				dto.ID = &id
				if dto.Version != nil {
					v := *dto.Version + 1
					dto.Version = &v
				} else {
					v := int64(2)
					dto.Version = &v
				}
				profilesMap[id] = dto
				_ = json.NewEncoder(w).Encode(dto)
				return

			case http.MethodDelete:
				delete(profilesMap, id)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		// -------------------------------------------------------------------
		// /v2/api/data-filtering-profiles
		// -------------------------------------------------------------------
		if r.URL.Path == "/v2/api/data-filtering-profiles" && r.Method == http.MethodGet {
			filteringMap := store["filtering_profiles"].(map[string]dlpclient.DataFilteringProfileDTO)
			nameFilter := r.URL.Query().Get("name")
			var list []dlpclient.DataFilteringProfileDTO
			for _, fp := range filteringMap {
				if nameFilter != "" && fp.Name != nil && !strings.EqualFold(*fp.Name, nameFilter) {
					continue
				}
				list = append(list, fp)
			}
			envelope := map[string]any{
				"content":       list,
				"totalElements": len(list),
				"totalPages":    1,
				"last":          true,
			}
			_ = json.NewEncoder(w).Encode(envelope)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/v2/api/data-filtering-profiles/") {
			id := strings.TrimPrefix(r.URL.Path, "/v2/api/data-filtering-profiles/")
			filteringMap := store["filtering_profiles"].(map[string]dlpclient.DataFilteringProfileDTO)

			switch r.Method {
			case http.MethodGet:
				fp, ok := filteringMap[id]
				if !ok {
					http.Error(w, `{"error":"filtering profile not found"}`, http.StatusNotFound)
					return
				}
				_ = json.NewEncoder(w).Encode(fp)
				return

			case http.MethodPut:
				var dto dlpclient.DataFilteringProfileDTO
				if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				dto.ID = &id
				filteringMap[id] = dto
				_ = json.NewEncoder(w).Encode(dto)
				return
			}
		}

		http.Error(w, fmt.Sprintf("unhandled path: %s %s", r.Method, r.URL.Path), http.StatusNotFound)
	})

	server := httptest.NewServer(handler)
	return server, mu, store
}

func newTestDLPClient(serverURL string) *dlpclient.Client {
	return &dlpclient.Client{
		HttpClient:  http.DefaultClient,
		BaseURL:     serverURL,
		ServiceName: "prisma-access",
		ClientName:  "terraform-provider-scm",
		UserAgent:   "test-agent",
		MaxRetries:  1,
	}
}

// TestProfiles_ExpressionTree_CRUD verifies CRUD on an expression_tree data profile.
func TestProfiles_ExpressionTree_CRUD(t *testing.T) {
	server, _, _ := setupProfilesMockServer(t)
	defer server.Close()

	client := newTestDLPClient(server.URL)
	provData := map[string]any{"dlp": client}
	ctx := context.Background()

	res := NewDataProfileResource().(resource.ResourceWithConfigure)
	var cfgResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: provData}, &cfgResp)
	if cfgResp.Diagnostics.HasError() {
		t.Fatalf("Configure failed: %v", cfgResp.Diagnostics)
	}

	plan := models.DataProfileResourceModel{
		Name:                  types.StringValue("Expression Tree Test Profile"),
		Description:           types.StringValue("Testing boolean expression tree"),
		ProfileType:           types.StringValue("advanced"),
		ProfileStatus:         types.StringValue("active"),
		IsGranularDataProfile: types.BoolValue(false),
		DetectionRules: []models.DataProfileDetectionRuleModel{
			{
				RuleType: types.StringValue("expression_tree"),
				ExpressionTree: &models.DataProfileExpressionTreeModel{
					OperatorType: types.StringValue("or"),
					SubExpressions: []models.DataProfileSubExpressionModel{
						{
							OperatorType: types.StringValue("and"),
							SubExpressions: []models.DataProfileSubExpressionLevel2Model{
								{
									RuleItem: &models.DataProfileRuleItemModel{
										Id:                     types.StringValue("pat-ssn-1"),
										Name:                   types.StringValue("US SSN Pattern"),
										DetectionTechnique:     types.StringValue("regex"),
										OccurrenceOperatorType: types.StringValue("more_than_equal_to"),
										OccurrenceCount:        types.Int64Value(1),
										ConfidenceLevel:        types.StringValue("high"),
										MatchType:              types.StringValue("include"),
										ByUniqueCount:          types.BoolValue(false),
									},
								},
								{
									RuleItem: &models.DataProfileRuleItemModel{
										Id:                     types.StringValue("pat-ccn-2"),
										Name:                   types.StringValue("Credit Card Pattern"),
										DetectionTechnique:     types.StringValue("regex"),
										OccurrenceOperatorType: types.StringValue("more_than_equal_to"),
										OccurrenceCount:        types.Int64Value(2),
										ConfidenceLevel:        types.StringValue("medium"),
										MatchType:              types.StringValue("include"),
										ByUniqueCount:          types.BoolValue(true),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	planVal := tfsdk.Plan{Schema: models.DataProfileResourceSchema}
	if diags := planVal.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("Failed to set test plan: %v", diags)
	}

	stateVal := tfsdk.State{Schema: models.DataProfileResourceSchema}

	createReq := resource.CreateRequest{Plan: planVal}
	createResp := resource.CreateResponse{State: stateVal}
	res.Create(ctx, createReq, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", createResp.Diagnostics)
	}

	var createdState models.DataProfileResourceModel
	createResp.State.Get(ctx, &createdState)
	if createdState.Id.IsNull() || createdState.Id.ValueString() == "" {
		t.Fatalf("Expected non-empty ID after create")
	}
	assignedID := createdState.Id.ValueString()
	if createdState.Version.ValueInt64() != 1 {
		t.Fatalf("Expected version 1, got %d", createdState.Version.ValueInt64())
	}

	// 2. Read
	readReq := resource.ReadRequest{State: createResp.State}
	readResp := resource.ReadResponse{State: createResp.State}
	res.Read(ctx, readReq, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", readResp.Diagnostics)
	}

	var readState models.DataProfileResourceModel
	readResp.State.Get(ctx, &readState)
	if readState.Name.ValueString() != "Expression Tree Test Profile" {
		t.Fatalf("Expected name 'Expression Tree Test Profile', got %s", readState.Name.ValueString())
	}

	// 3. Update
	readState.Description = types.StringValue("Updated description")
	updatePlan := tfsdk.Plan{Schema: models.DataProfileResourceSchema}
	updatePlan.Set(ctx, &readState)

	updateReq := resource.UpdateRequest{
		Plan:  updatePlan,
		State: readResp.State,
	}
	updateResp := resource.UpdateResponse{State: readResp.State}
	res.Update(ctx, updateReq, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", updateResp.Diagnostics)
	}

	var updatedState models.DataProfileResourceModel
	updateResp.State.Get(ctx, &updatedState)
	if updatedState.Description.ValueString() != "Updated description" {
		t.Fatalf("Expected description 'Updated description', got %s", updatedState.Description.ValueString())
	}

	// 4. Delete
	deleteReq := resource.DeleteRequest{
		State: updateResp.State,
	}
	deleteResp := resource.DeleteResponse{}
	res.Delete(ctx, deleteReq, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete failed: %v", deleteResp.Diagnostics)
	}

	// Read again -> expect resource removed
	readAfterDeleteReq := resource.ReadRequest{State: updateResp.State}
	readAfterDeleteResp := resource.ReadResponse{State: tfsdk.State{Schema: models.DataProfileResourceSchema}}
	res.Read(ctx, readAfterDeleteReq, &readAfterDeleteResp)
	if readAfterDeleteResp.Diagnostics.HasError() {
		t.Fatalf("Read after delete returned error: %v", readAfterDeleteResp.Diagnostics)
	}
	if !readAfterDeleteResp.State.Raw.IsNull() {
		t.Fatalf("Expected state to be removed after delete, got: %v", readAfterDeleteResp.State.Raw)
	}
	_ = assignedID
}

// TestProfiles_MultiProfile_Granular verifies multi_profile granular profile.
func TestProfiles_MultiProfile_Granular(t *testing.T) {
	server, _, _ := setupProfilesMockServer(t)
	defer server.Close()

	client := newTestDLPClient(server.URL)
	provData := map[string]any{"dlp": client}
	ctx := context.Background()

	res := NewDataProfileResource().(resource.ResourceWithConfigure)
	var cfgResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: provData}, &cfgResp)
	if cfgResp.Diagnostics.HasError() {
		t.Fatalf("Configure failed: %v", cfgResp.Diagnostics)
	}

	plan := models.DataProfileResourceModel{
		Name:                  types.StringValue("Granular Multi Profile"),
		Description:           types.StringValue("Multi profile combining child profiles"),
		ProfileType:           types.StringValue("advanced"),
		ProfileStatus:         types.StringValue("active"),
		IsGranularDataProfile: types.BoolValue(true),
		DetectionRules: []models.DataProfileDetectionRuleModel{
			{
				RuleType: types.StringValue("multi_profile"),
				MultiProfile: &models.DataProfileMultiProfileModel{
					OperatorType: types.StringValue("or"),
					DataProfileIds: []types.Int64{
						types.Int64Value(101),
						types.Int64Value(102),
					},
				},
			},
		},
	}

	planVal := tfsdk.Plan{Schema: models.DataProfileResourceSchema}
	planVal.Set(ctx, &plan)

	stateVal := tfsdk.State{Schema: models.DataProfileResourceSchema}

	createReq := resource.CreateRequest{Plan: planVal}
	createResp := resource.CreateResponse{State: stateVal}
	res.Create(ctx, createReq, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create multi_profile failed: %v", createResp.Diagnostics)
	}

	var createdState models.DataProfileResourceModel
	createResp.State.Get(ctx, &createdState)
	if createdState.Id.IsNull() || createdState.Id.ValueString() == "" {
		t.Fatalf("Expected assigned ID")
	}
	if !createdState.IsGranularDataProfile.ValueBool() {
		t.Fatalf("Expected is_granular_data_profile true")
	}
}

// TestProfiles_DataSources verifies single and list data sources.
func TestProfiles_DataSources(t *testing.T) {
	server, _, _ := setupProfilesMockServer(t)
	defer server.Close()

	client := newTestDLPClient(server.URL)
	provData := map[string]any{"dlp": client}
	ctx := context.Background()

	// 1. Seed two profiles via client
	p1 := dlpclient.DataProfileDTO{
		Name: "Financial PII Profile",
		DetectionRules: []dlpclient.DetectionRuleDTO{
			{
				RuleType: "expression_tree",
				ExpressionTree: &dlpclient.ExpressionTreeNodeDTO{
					RuleItem: &dlpclient.RuleItemDTO{
						ID:                     "pat-1",
						DetectionTechnique:     "regex",
						OccurrenceOperatorType: "any",
					},
				},
			},
		},
	}
	var created1 dlpclient.DataProfileDTO
	if err := client.Post(ctx, "/v2/api/data-profiles", nil, p1, &created1); err != nil {
		t.Fatalf("Seed profile 1 failed: %v", err)
	}

	p2 := dlpclient.DataProfileDTO{
		Name: "Healthcare HIPAA Profile",
		DetectionRules: []dlpclient.DetectionRuleDTO{
			{
				RuleType: "expression_tree",
				ExpressionTree: &dlpclient.ExpressionTreeNodeDTO{
					RuleItem: &dlpclient.RuleItemDTO{
						ID:                     "pat-2",
						DetectionTechnique:     "regex",
						OccurrenceOperatorType: "any",
					},
				},
			},
		},
	}
	var created2 dlpclient.DataProfileDTO
	if err := client.Post(ctx, "/v2/api/data-profiles", nil, p2, &created2); err != nil {
		t.Fatalf("Seed profile 2 failed: %v", err)
	}

	// 2. Test Single Data Source
	ds := NewDataProfileDataSource().(datasource.DataSourceWithConfigure)
	var dsCfgResp datasource.ConfigureResponse
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &dsCfgResp)
	if dsCfgResp.Diagnostics.HasError() {
		t.Fatalf("DS configure failed: %v", dsCfgResp.Diagnostics)
	}

	dsStateVal := tfsdk.State{Schema: models.DataProfileDataSourceSchema}
	dsStateVal.Set(ctx, &models.DataProfileResourceModel{Id: types.StringValue(*created1.ID)})

	dsReadReq := datasource.ReadRequest{Config: tfsdk.Config(dsStateVal)}
	dsReadResp := datasource.ReadResponse{State: dsStateVal}
	ds.Read(ctx, dsReadReq, &dsReadResp)
	if dsReadResp.Diagnostics.HasError() {
		t.Fatalf("DS read failed: %v", dsReadResp.Diagnostics)
	}

	var dsResult models.DataProfileResourceModel
	dsReadResp.State.Get(ctx, &dsResult)
	if dsResult.Name.ValueString() != "Financial PII Profile" {
		t.Fatalf("Expected name 'Financial PII Profile', got %s", dsResult.Name.ValueString())
	}

	// 3. Test List Data Source with name filter
	listDS := NewDataProfileListDataSource().(datasource.DataSourceWithConfigure)
	var listCfgResp datasource.ConfigureResponse
	listDS.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &listCfgResp)
	if listCfgResp.Diagnostics.HasError() {
		t.Fatalf("List DS configure failed: %v", listCfgResp.Diagnostics)
	}

	listStateVal := tfsdk.State{Schema: models.DataProfileListDataSourceSchema}
	listStateVal.Set(ctx, &models.DataProfileListDataSourceModel{
		Name: types.StringValue("Healthcare HIPAA Profile"),
	})

	listReadReq := datasource.ReadRequest{Config: tfsdk.Config(listStateVal)}
	listReadResp := datasource.ReadResponse{State: listStateVal}
	listDS.Read(ctx, listReadReq, &listReadResp)
	if listReadResp.Diagnostics.HasError() {
		t.Fatalf("List DS read failed: %v", listReadResp.Diagnostics)
	}

	var listResult models.DataProfileListDataSourceModel
	listReadResp.State.Get(ctx, &listResult)
	if listResult.Total.ValueInt64() != 1 {
		t.Fatalf("Expected 1 filtered profile, got %d", listResult.Total.ValueInt64())
	}
	if len(listResult.Data) != 1 || listResult.Data[0].Name.ValueString() != "Healthcare HIPAA Profile" {
		t.Fatalf("Unexpected filtered profile name: %v", listResult.Data)
	}
}

// TestProfiles_DataFilteringProfile_CRUD verifies CRUD on data filtering profile linked to data profile.
func TestProfiles_DataFilteringProfile_CRUD(t *testing.T) {
	server, _, _ := setupProfilesMockServer(t)
	defer server.Close()

	client := newTestDLPClient(server.URL)
	provData := map[string]any{"dlp": client}
	ctx := context.Background()

	// 1. First seed parent data profile
	p := dlpclient.DataProfileDTO{
		Name: "Parent Profile for Filtering",
		DetectionRules: []dlpclient.DetectionRuleDTO{
			{
				RuleType: "expression_tree",
				ExpressionTree: &dlpclient.ExpressionTreeNodeDTO{
					RuleItem: &dlpclient.RuleItemDTO{
						ID:                     "pat-1",
						DetectionTechnique:     "regex",
						OccurrenceOperatorType: "any",
					},
				},
			},
		},
	}
	var createdProfile dlpclient.DataProfileDTO
	if err := client.Post(ctx, "/v2/api/data-profiles", nil, p, &createdProfile); err != nil {
		t.Fatalf("Seed parent profile failed: %v", err)
	}
	parentIDInt, _ := strconv.ParseInt(*createdProfile.ID, 10, 64)

	// 2. Configure Filtering Profile
	res := NewDataFilteringProfileResource().(resource.ResourceWithConfigure)
	var cfgResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: provData}, &cfgResp)
	if cfgResp.Diagnostics.HasError() {
		t.Fatalf("Configure failed: %v", cfgResp.Diagnostics)
	}

	fpPlan := models.DataFilteringProfileResourceModel{
		DataProfileId:            types.Int64Value(parentIDInt),
		Name:                     types.StringValue("Custom Filtering Profile"),
		Direction:                types.StringValue("c2s"),
		ScanType:                 types.StringValue("include"),
		LogSeverity:              types.StringValue("CRITICAL"),
		FileBased:                types.BoolValue(true),
		NonFileBased:             types.BoolValue(false),
		FileType:                 []types.String{types.StringValue("pdf"), types.StringValue("docx")},
		IsEndUserCoachingEnabled: types.BoolValue(true),
		EucTemplateId:            types.StringValue("euc-template-01"),
		Rule1Action:              types.StringValue("block"),
		Rule2Action:              types.StringValue("alert"),
		AppExclusionList: []models.AppExclusionModel{
			{
				AppId:   types.StringValue("app-box-1"),
				AppName: types.StringValue("Box"),
				Type:    types.StringValue("application"),
			},
		},
		UrlExclusionList: []models.URLExclusionModel{
			{
				UrlId:   types.StringValue("url-safe-internal"),
				UrlName: types.StringValue("*.internal.domain"),
				Type:    types.StringValue("url"),
			},
		},
		ExceptionRules: []models.ExceptionRuleModel{
			{
				Name:        types.StringValue("Allow CEO uploads"),
				Action:      types.StringValue("allow"),
				LogSeverity: types.StringValue("INFORMATIONAL"),
			},
		},
	}

	planVal := tfsdk.Plan{Schema: models.DataFilteringProfileResourceSchema}
	planVal.Set(ctx, &fpPlan)

	stateVal := tfsdk.State{Schema: models.DataFilteringProfileResourceSchema}

	createReq := resource.CreateRequest{Plan: planVal}
	createResp := resource.CreateResponse{State: stateVal}
	res.Create(ctx, createReq, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Filtering profile Create failed: %v", createResp.Diagnostics)
	}

	var createdState models.DataFilteringProfileResourceModel
	createResp.State.Get(ctx, &createdState)
	if createdState.Id.IsNull() || createdState.Id.ValueString() == "" {
		t.Fatalf("Expected non-empty ID")
	}
	if createdState.Direction.ValueString() != "c2s" {
		t.Fatalf("Expected direction 'c2s', got %s", createdState.Direction.ValueString())
	}
	if len(createdState.FileType) != 2 {
		t.Fatalf("Expected 2 file types, got %d", len(createdState.FileType))
	}
	if len(createdState.AppExclusionList) != 1 {
		t.Fatalf("Expected 1 app exclusion, got %d", len(createdState.AppExclusionList))
	}

	// 3. Update Filtering Profile
	createdState.Direction = types.StringValue("both")
	createdState.LogSeverity = types.StringValue("HIGH")
	updatePlan := tfsdk.Plan{Schema: models.DataFilteringProfileResourceSchema}
	updatePlan.Set(ctx, &createdState)

	updateReq := resource.UpdateRequest{
		Plan:  updatePlan,
		State: createResp.State,
	}
	updateResp := resource.UpdateResponse{State: createResp.State}
	res.Update(ctx, updateReq, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Filtering profile Update failed: %v", updateResp.Diagnostics)
	}

	var updatedState models.DataFilteringProfileResourceModel
	updateResp.State.Get(ctx, &updatedState)
	if updatedState.Direction.ValueString() != "both" {
		t.Fatalf("Expected direction 'both', got %s", updatedState.Direction.ValueString())
	}

	// 4. Test Filtering Profile Data Source
	fpDS := NewDataFilteringProfileDataSource().(datasource.DataSourceWithConfigure)
	var fpDsCfgResp datasource.ConfigureResponse
	fpDS.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &fpDsCfgResp)
	if fpDsCfgResp.Diagnostics.HasError() {
		t.Fatalf("FP DS configure failed: %v", fpDsCfgResp.Diagnostics)
	}

	fpDsStateVal := tfsdk.State{Schema: models.DataFilteringProfileDataSourceSchema}
	fpDsStateVal.Set(ctx, &models.DataFilteringProfileResourceModel{Id: updatedState.Id})

	fpDsReadReq := datasource.ReadRequest{Config: tfsdk.Config(fpDsStateVal)}
	fpDsReadResp := datasource.ReadResponse{State: fpDsStateVal}
	fpDS.Read(ctx, fpDsReadReq, &fpDsReadResp)
	if fpDsReadResp.Diagnostics.HasError() {
		t.Fatalf("FP DS read failed: %v", fpDsReadResp.Diagnostics)
	}

	var fpDsResult models.DataFilteringProfileResourceModel
	fpDsReadResp.State.Get(ctx, &fpDsResult)
	if fpDsResult.LogSeverity.ValueString() != "HIGH" {
		t.Fatalf("Expected log_severity 'HIGH', got %s", fpDsResult.LogSeverity.ValueString())
	}

	// 5. Test Filtering Profile List Data Source
	fpListDS := NewDataFilteringProfileListDataSource().(datasource.DataSourceWithConfigure)
	var fpListCfgResp datasource.ConfigureResponse
	fpListDS.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &fpListCfgResp)
	if fpListCfgResp.Diagnostics.HasError() {
		t.Fatalf("FP List DS configure failed: %v", fpListCfgResp.Diagnostics)
	}

	fpListStateVal := tfsdk.State{Schema: models.DataFilteringProfileListDataSourceSchema}
	fpListStateVal.Set(ctx, &models.DataFilteringProfileListDataSourceModel{})

	fpListReadReq := datasource.ReadRequest{Config: tfsdk.Config(fpListStateVal)}
	fpListReadResp := datasource.ReadResponse{State: fpListStateVal}
	fpListDS.Read(ctx, fpListReadReq, &fpListReadResp)
	if fpListReadResp.Diagnostics.HasError() {
		t.Fatalf("FP List DS read failed: %v", fpListReadResp.Diagnostics)
	}

	var fpListResult models.DataFilteringProfileListDataSourceModel
	fpListReadResp.State.Get(ctx, &fpListResult)
	if fpListResult.Total.ValueInt64() < 1 {
		t.Fatalf("Expected at least 1 filtering profile in list, got %d", fpListResult.Total.ValueInt64())
	}
}
