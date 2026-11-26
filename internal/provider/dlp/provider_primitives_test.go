package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

// mockServer encapsulates the test HTTP backend state
type mockPrimitivesBackend struct {
	mu           sync.Mutex
	patterns     map[string]dlpclient.DataPatternDTO
	dictionaries map[string]dlpclient.DictionaryDTO
	docTypes     map[string]dlpclient.DocumentTypeDTO
	edmDatasets  map[string]dlpclient.EdmDatasetDTO
	nextID       int
}

func newMockBackend() *mockPrimitivesBackend {
	return &mockPrimitivesBackend{
		patterns:     make(map[string]dlpclient.DataPatternDTO),
		dictionaries: make(map[string]dlpclient.DictionaryDTO),
		docTypes:     make(map[string]dlpclient.DocumentTypeDTO),
		edmDatasets:  make(map[string]dlpclient.EdmDatasetDTO),
		nextID:       100,
	}
}

func setupTestServer(b *mockPrimitivesBackend) *httptest.Server {
	mux := http.NewServeMux()

	// 1. Data Patterns
	mux.HandleFunc("/v2/api/data-patterns", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			nameFilter := r.URL.Query().Get("name")
			var list []dlpclient.DataPatternDTO
			for _, p := range b.patterns {
				if nameFilter == "" || strings.EqualFold(p.Name, nameFilter) {
					list = append(list, p)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":        list,
				"total_elements": len(list),
			})
		case http.MethodPost:
			var dto dlpclient.DataPatternDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			b.nextID++
			id := fmt.Sprintf("pattern-%d", b.nextID)
			dto.ID = &id
			v := int64(1)
			dto.Version = &v
			s := "ACTIVE"
			dto.Status = &s
			b.patterns[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2/api/data-patterns/", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := strings.TrimPrefix(r.URL.Path, "/v2/api/data-patterns/")
		switch r.Method {
		case http.MethodGet:
			p, exists := b.patterns[id]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(p)
		case http.MethodPut:
			var dto dlpclient.DataPatternDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			dto.ID = &id
			v := int64(2)
			dto.Version = &v
			b.patterns[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		case http.MethodDelete:
			delete(b.patterns, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 2. Dictionaries
	mux.HandleFunc("/v2/api/dictionaries", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			var list []dlpclient.DictionaryDTO
			for _, d := range b.dictionaries {
				list = append(list, d)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":        list,
				"total_elements": len(list),
			})
		case http.MethodPost:
			if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				mr, err := r.MultipartReader()
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var dto dlpclient.DictionaryDTO
				var keywords []string
				for {
					part, err := mr.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					data, _ := io.ReadAll(part)
					if part.FormName() == "data" || part.FormName() == "json" {
						_ = json.Unmarshal(data, &dto)
					} else if part.FormName() == "file" {
						lines := strings.Split(string(data), "\n")
						for _, l := range lines {
							l = strings.TrimSpace(l)
							if l != "" && !strings.EqualFold(l, "keywords") {
								keywords = append(keywords, l)
							}
						}
					}
				}
				b.nextID++
				id := fmt.Sprintf("dict-%d", b.nextID)
				dto.ID = &id
				dto.Keywords = keywords
				cnt := int64(len(keywords))
				dto.KeywordCount = &cnt
				b.dictionaries[id] = dto
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(dto)
				return
			}

			var dto dlpclient.DictionaryDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			b.nextID++
			id := fmt.Sprintf("dict-%d", b.nextID)
			dto.ID = &id
			cnt := int64(len(dto.Keywords))
			dto.KeywordCount = &cnt
			b.dictionaries[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2/api/dictionaries/", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := strings.TrimPrefix(r.URL.Path, "/v2/api/dictionaries/")
		switch r.Method {
		case http.MethodGet:
			d, exists := b.dictionaries[id]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(d)
		case http.MethodPut:
			// Handle multipart or json
			if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
				mr, err := r.MultipartReader()
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var dto dlpclient.DictionaryDTO
				var keywords []string
				for {
					part, err := mr.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					data, _ := io.ReadAll(part)
					if part.FormName() == "data" || part.FormName() == "json" {
						_ = json.Unmarshal(data, &dto)
					} else if part.FormName() == "file" {
						lines := strings.Split(string(data), "\n")
						for _, l := range lines {
							l = strings.TrimSpace(l)
							if l != "" && !strings.EqualFold(l, "keywords") {
								keywords = append(keywords, l)
							}
						}
					}
				}
				dto.ID = &id
				dto.Keywords = keywords
				cnt := int64(len(keywords))
				dto.KeywordCount = &cnt
				b.dictionaries[id] = dto
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(dto)
				return
			}

			var dto dlpclient.DictionaryDTO
			_ = json.NewDecoder(r.Body).Decode(&dto)
			dto.ID = &id
			b.dictionaries[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		case http.MethodDelete:
			delete(b.dictionaries, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 3. Document Types
	mux.HandleFunc("/v2/api/document-types", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			var list []dlpclient.DocumentTypeDTO
			for _, dt := range b.docTypes {
				list = append(list, dt)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":        list,
				"total_elements": len(list),
			})
		case http.MethodPost:
			var dto dlpclient.DocumentTypeDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			b.nextID++
			id := fmt.Sprintf("doc-%d", b.nextID)
			dto.ID = &id
			b.docTypes[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2/api/document-types/", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := strings.TrimPrefix(r.URL.Path, "/v2/api/document-types/")
		switch r.Method {
		case http.MethodGet:
			dt, exists := b.docTypes[id]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dt)
		case http.MethodPut:
			var dto dlpclient.DocumentTypeDTO
			_ = json.NewDecoder(r.Body).Decode(&dto)
			dto.ID = &id
			b.docTypes[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		case http.MethodDelete:
			delete(b.docTypes, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 4. EDM Datasets
	mux.HandleFunc("/v2/api/edm-datasets", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			var list []dlpclient.EdmDatasetDTO
			for _, ds := range b.edmDatasets {
				list = append(list, ds)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":        list,
				"total_elements": len(list),
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2/api/edm-datasets/", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := strings.TrimPrefix(r.URL.Path, "/v2/api/edm-datasets/")
		switch r.Method {
		case http.MethodGet:
			ds, exists := b.edmDatasets[id]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ds)
		case http.MethodDelete:
			delete(b.edmDatasets, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func createTestClient(ts *httptest.Server) *dlpclient.Client {
	u, _ := url.Parse(ts.URL)
	return dlpclient.NewClient(dlpclient.Config{
		Protocol:   u.Scheme,
		Host:       u.Host,
		HttpClient: ts.Client(),
	})
}

// ---------------------------------------------------------------------------
// Unit tests for scm_dlp_data_pattern
// ---------------------------------------------------------------------------

func TestPrimitivesDataPattern(t *testing.T) {
	backend := newMockBackend()
	ts := setupTestServer(backend)
	defer ts.Close()

	client := createTestClient(ts)
	providerData := map[string]any{"dlp": client}

	ctx := context.Background()
	res := NewDataPatternResource().(*DataPatternResource)
	res.client = client

	// 1. Create Data Pattern
	confLevels, _ := types.ListValueFrom(ctx, types.StringType, []string{"high"})
	createModel := models.DataPatternModel{
		Name: types.StringValue("Employee-ID-Pattern"),
		Type: types.StringValue("custom"),
		DetectionConfig: &models.DataPatternDetectionConfigModel{
			Technique:                 types.StringValue("weighted_regex"),
			SupportedConfidenceLevels: confLevels,
		},
		MatchingRules: &models.DataPatternMatchingRulesModel{
			ProximityDistance: types.Int64Value(50),
			ProximityKeywords: types.ListNull(types.StringType),
			Regexes: []models.DataPatternRegexRuleModel{
				{
					Regex:  types.StringValue("EMP-[0-9]{6}"),
					Weight: types.Float64Value(10),
				},
			},
		},
	}

	planState := tfsdk.State{
		Schema: models.DataPatternResourceSchema,
	}
	createPlan := tfsdk.Plan{
		Schema: models.DataPatternResourceSchema,
	}
	_ = createPlan.Set(ctx, &createModel)

	createReq := resource.CreateRequest{Plan: createPlan}
	createResp := resource.CreateResponse{State: planState}
	res.Create(ctx, createReq, &createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create failed with diagnostics: %v", createResp.Diagnostics.Errors())
	}

	var stateAfterCreate models.DataPatternModel
	_ = createResp.State.Get(ctx, &stateAfterCreate)
	patternID := stateAfterCreate.Id.ValueString()
	if patternID == "" {
		t.Fatalf("Expected pattern ID to be populated")
	}

	// 2. Read Data Pattern
	readReq := resource.ReadRequest{State: createResp.State}
	readResp := resource.ReadResponse{State: createResp.State}
	res.Read(ctx, readReq, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", readResp.Diagnostics.Errors())
	}

	// 3. Update Data Pattern
	updateModel := stateAfterCreate
	updateModel.Description = types.StringValue("Updated description")
	updatePlan := tfsdk.Plan{Schema: models.DataPatternResourceSchema}
	_ = updatePlan.Set(ctx, &updateModel)

	updateReq := resource.UpdateRequest{Plan: updatePlan, State: readResp.State}
	updateResp := resource.UpdateResponse{State: readResp.State}
	res.Update(ctx, updateReq, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", updateResp.Diagnostics.Errors())
	}

	var stateAfterUpdate models.DataPatternModel
	_ = updateResp.State.Get(ctx, &stateAfterUpdate)
	if stateAfterUpdate.Description.ValueString() != "Updated description" {
		t.Errorf("Expected description to be 'Updated description', got %q", stateAfterUpdate.Description.ValueString())
	}

	// 4. Test DataSource & List DataSource
	ds := NewDataPatternDataSource().(*DataPatternDataSource)
	ds.client = client
	var dsData models.DataPatternModel
	dsData.Id = types.StringValue(patternID)
	var dsConfigState tfsdk.State
	dsConfigState.Schema = models.DataPatternDataSourceSchema
	_ = dsConfigState.Set(ctx, &dsData)
	dsReq := datasource.ReadRequest{Config: tfsdk.Config{
		Raw:    dsConfigState.Raw,
		Schema: models.DataPatternDataSourceSchema,
	}}
	dsResp := datasource.ReadResponse{State: tfsdk.State{
		Schema: models.DataPatternDataSourceSchema,
	}}
	ds.Read(ctx, dsReq, &dsResp)
	if dsResp.Diagnostics.HasError() {
		t.Fatalf("DataSource Read failed: %v", dsResp.Diagnostics.Errors())
	}

	listDS := NewDataPatternListDataSource().(*DataPatternListDataSource)
	listDS.client = client
	var listData models.DataPatternListModel
	listData.Name = types.StringValue("Employee-ID-Pattern")
	var listConfigState tfsdk.State
	listConfigState.Schema = models.DataPatternListDataSourceSchema
	_ = listConfigState.Set(ctx, &listData)
	listReq := datasource.ReadRequest{Config: tfsdk.Config{
		Raw:    listConfigState.Raw,
		Schema: models.DataPatternListDataSourceSchema,
	}}
	listResp := datasource.ReadResponse{State: tfsdk.State{
		Schema: models.DataPatternListDataSourceSchema,
	}}
	listDS.Read(ctx, listReq, &listResp)
	if listResp.Diagnostics.HasError() {
		t.Fatalf("List DataSource Read failed: %v", listResp.Diagnostics.Errors())
	}
	var listResult models.DataPatternListModel
	_ = listResp.State.Get(ctx, &listResult)
	if listResult.Total.ValueInt64() != 1 {
		t.Errorf("Expected 1 result from list filter, got %d", listResult.Total.ValueInt64())
	}

	// 5. Delete Data Pattern
	delReq := resource.DeleteRequest{State: updateResp.State}
	delResp := resource.DeleteResponse{State: updateResp.State}
	res.Delete(ctx, delReq, &delResp)
	if delResp.Diagnostics.HasError() {
		t.Fatalf("Delete failed: %v", delResp.Diagnostics.Errors())
	}

	if _, ok := backend.patterns[patternID]; ok {
		t.Errorf("Pattern %s still exists in backend after Delete", patternID)
	}

	// Test Configure
	confReq := resource.ConfigureRequest{ProviderData: providerData}
	confResp := resource.ConfigureResponse{}
	res.Configure(ctx, confReq, &confResp)
	if confResp.Diagnostics.HasError() {
		t.Errorf("Configure failed: %v", confResp.Diagnostics.Errors())
	}
}

// ---------------------------------------------------------------------------
// Unit tests for scm_dlp_dictionary (including multipart keywords upload)
// ---------------------------------------------------------------------------

func TestPrimitivesDictionary(t *testing.T) {
	backend := newMockBackend()
	ts := setupTestServer(backend)
	defer ts.Close()

	client := createTestClient(ts)
	ctx := context.Background()
	res := NewDictionaryResource().(*DictionaryResource)
	res.client = client

	kwList, _ := types.ListValueFrom(ctx, types.StringType, []string{"topsecret", "classified"})

	createModel := models.DictionaryModel{
		Name:             types.StringValue("Project-Keywords"),
		Category:         types.StringValue("Confidential"),
		OriginalFileName: types.StringValue("keywords.txt"),
		RegionName:       types.StringValue("us"),
		IsCaseSensitive:  types.BoolValue(false),
		KeywordList:      kwList,
	}

	planState := tfsdk.State{Schema: models.DictionaryResourceSchema}
	createPlan := tfsdk.Plan{Schema: models.DictionaryResourceSchema}
	_ = createPlan.Set(ctx, &createModel)

	createReq := resource.CreateRequest{Plan: createPlan}
	createResp := resource.CreateResponse{State: planState}
	res.Create(ctx, createReq, &createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create dictionary failed: %v", createResp.Diagnostics.Errors())
	}

	var stateAfterCreate models.DictionaryModel
	_ = createResp.State.Get(ctx, &stateAfterCreate)
	dictID := stateAfterCreate.Id.ValueString()
	if dictID == "" {
		t.Fatalf("Expected dictionary ID to be set")
	}

	// Verify keywords uploaded via multipart in backend
	backend.mu.Lock()
	savedDict := backend.dictionaries[dictID]
	backend.mu.Unlock()
	if len(savedDict.Keywords) != 2 {
		t.Fatalf("Expected 2 keywords saved in backend via multipart, got %d", len(savedDict.Keywords))
	}

	// Read
	readReq := resource.ReadRequest{State: createResp.State}
	readResp := resource.ReadResponse{State: tfsdk.State{Schema: models.DictionaryResourceSchema}}
	res.Read(ctx, readReq, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", readResp.Diagnostics.Errors())
	}

	var readState models.DictionaryModel
	_ = readResp.State.Get(ctx, &readState)

	// Update with new keywords
	newKwList, _ := types.ListValueFrom(ctx, types.StringType, []string{"topsecret", "classified", "restricted"})
	updateModel := readState
	updateModel.KeywordList = newKwList
	updatePlan := tfsdk.Plan{Schema: models.DictionaryResourceSchema}
	_ = updatePlan.Set(ctx, &updateModel)

	updateReq := resource.UpdateRequest{Plan: updatePlan, State: readResp.State}
	updateResp := resource.UpdateResponse{State: readResp.State}
	res.Update(ctx, updateReq, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update dictionary failed: %v", updateResp.Diagnostics.Errors())
	}

	backend.mu.Lock()
	savedDict = backend.dictionaries[dictID]
	backend.mu.Unlock()
	if len(savedDict.Keywords) != 3 {
		t.Fatalf("Expected 3 keywords after update, got %d", len(savedDict.Keywords))
	}

	// Metadata-only update (keyword_list null): verify existing keywords are NOT wiped (C2)
	var stateAfterKeywordsUpdate models.DictionaryModel
	_ = updateResp.State.Get(ctx, &stateAfterKeywordsUpdate)
	metaOnlyModel := stateAfterKeywordsUpdate
	metaOnlyModel.Description = types.StringValue("Metadata only update without keywords")
	metaOnlyModel.KeywordList = types.ListNull(types.StringType)
	metaOnlyPlan := tfsdk.Plan{Schema: models.DictionaryResourceSchema}
	_ = metaOnlyPlan.Set(ctx, &metaOnlyModel)

	metaOnlyReq := resource.UpdateRequest{Plan: metaOnlyPlan, State: updateResp.State}
	metaOnlyResp := resource.UpdateResponse{State: updateResp.State}
	res.Update(ctx, metaOnlyReq, &metaOnlyResp)
	if metaOnlyResp.Diagnostics.HasError() {
		t.Fatalf("Metadata-only update failed: %v", metaOnlyResp.Diagnostics.Errors())
	}

	backend.mu.Lock()
	savedDict = backend.dictionaries[dictID]
	backend.mu.Unlock()
	if len(savedDict.Keywords) != 3 {
		t.Fatalf("Expected 3 keywords preserved after metadata-only update, got %d", len(savedDict.Keywords))
	}

	// DataSource and List
	ds := NewDictionaryDataSource().(*DictionaryDataSource)
	ds.client = client
	dsData := models.DictionaryModel{
		Id:               types.StringValue(dictID),
		Tfid:             types.StringNull(),
		Name:             types.StringNull(),
		Description:      types.StringNull(),
		Category:         types.StringNull(),
		OriginalFileName: types.StringNull(),
		RegionName:       types.StringNull(),
		IsCaseSensitive:  types.BoolNull(),
		KeywordList:      types.ListNull(types.StringType),
		KeywordCount:     types.Int64Null(),
	}
	var dsConfigState tfsdk.State
	dsConfigState.Schema = models.DictionaryDataSourceSchema
	_ = dsConfigState.Set(ctx, &dsData)
	dsResp := datasource.ReadResponse{State: tfsdk.State{
		Schema: models.DictionaryDataSourceSchema,
	}}
	ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Raw:    dsConfigState.Raw,
		Schema: models.DictionaryDataSourceSchema,
	}}, &dsResp)
	if dsResp.Diagnostics.HasError() {
		t.Fatalf("Dictionary DataSource Read failed: %v", dsResp.Diagnostics.Errors())
	}

	listDS := NewDictionaryListDataSource().(*DictionaryListDataSource)
	listDS.client = client
	var listData models.DictionaryListModel
	var listConfigState tfsdk.State
	listConfigState.Schema = models.DictionaryListDataSourceSchema
	_ = listConfigState.Set(ctx, &listData)
	listResp := datasource.ReadResponse{State: tfsdk.State{
		Schema: models.DictionaryListDataSourceSchema,
	}}
	listDS.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Raw:    listConfigState.Raw,
		Schema: models.DictionaryListDataSourceSchema,
	}}, &listResp)
	if listResp.Diagnostics.HasError() {
		t.Fatalf("Dictionary List DataSource Read failed: %v", listResp.Diagnostics.Errors())
	}

	// Delete
	delReq := resource.DeleteRequest{State: updateResp.State}
	delResp := resource.DeleteResponse{State: updateResp.State}
	res.Delete(ctx, delReq, &delResp)
	if delResp.Diagnostics.HasError() {
		t.Fatalf("Delete dictionary failed: %v", delResp.Diagnostics.Errors())
	}
}

// ---------------------------------------------------------------------------
// Unit tests for scm_dlp_document_type
// ---------------------------------------------------------------------------

func TestPrimitivesDocumentType(t *testing.T) {
	backend := newMockBackend()
	ts := setupTestServer(backend)
	defer ts.Close()

	client := createTestClient(ts)
	ctx := context.Background()
	res := NewDocumentTypeResource().(*DocumentTypeResource)
	res.client = client

	createModel := models.DocumentTypeModel{
		Name:               types.StringValue("Financial-Form"),
		Category:           types.StringValue("Financial"),
		DetectionTechnique: types.StringValue("document_fingerprint"),
		OriginalFileName:   types.StringValue("form1040.pdf"),
	}

	planState := tfsdk.State{Schema: models.DocumentTypeResourceSchema}
	createPlan := tfsdk.Plan{Schema: models.DocumentTypeResourceSchema}
	_ = createPlan.Set(ctx, &createModel)

	createReq := resource.CreateRequest{Plan: createPlan}
	createResp := resource.CreateResponse{State: planState}
	res.Create(ctx, createReq, &createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create document_type failed: %v", createResp.Diagnostics.Errors())
	}

	var state models.DocumentTypeModel
	_ = createResp.State.Get(ctx, &state)
	docID := state.Id.ValueString()
	if docID == "" {
		t.Fatalf("Expected docID to be set")
	}

	// Read
	readReq := resource.ReadRequest{State: createResp.State}
	readResp := resource.ReadResponse{State: createResp.State}
	res.Read(ctx, readReq, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read document_type failed: %v", readResp.Diagnostics.Errors())
	}

	// Update
	updateModel := state
	updateModel.Description = types.StringValue("IRS Tax Form")
	updatePlan := tfsdk.Plan{Schema: models.DocumentTypeResourceSchema}
	_ = updatePlan.Set(ctx, &updateModel)
	updateResp := resource.UpdateResponse{State: readResp.State}
	res.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: readResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update document_type failed: %v", updateResp.Diagnostics.Errors())
	}

	// DataSource and List
	ds := NewDocumentTypeDataSource().(*DocumentTypeDataSource)
	ds.client = client
	var dsData models.DocumentTypeModel
	dsData.Id = types.StringValue(docID)
	var dsConfigState tfsdk.State
	dsConfigState.Schema = models.DocumentTypeDataSourceSchema
	_ = dsConfigState.Set(ctx, &dsData)
	dsResp := datasource.ReadResponse{State: tfsdk.State{
		Schema: models.DocumentTypeDataSourceSchema,
	}}
	ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Raw:    dsConfigState.Raw,
		Schema: models.DocumentTypeDataSourceSchema,
	}}, &dsResp)
	if dsResp.Diagnostics.HasError() {
		t.Fatalf("DocumentType DataSource Read failed: %v", dsResp.Diagnostics.Errors())
	}

	listDS := NewDocumentTypeListDataSource().(*DocumentTypeListDataSource)
	listDS.client = client
	var listData models.DocumentTypeListModel
	var listConfigState tfsdk.State
	listConfigState.Schema = models.DocumentTypeListDataSourceSchema
	_ = listConfigState.Set(ctx, &listData)
	listResp := datasource.ReadResponse{State: tfsdk.State{
		Schema: models.DocumentTypeListDataSourceSchema,
	}}
	listDS.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Raw:    listConfigState.Raw,
		Schema: models.DocumentTypeListDataSourceSchema,
	}}, &listResp)
	if listResp.Diagnostics.HasError() {
		t.Fatalf("DocumentType List DataSource Read failed: %v", listResp.Diagnostics.Errors())
	}

	// Delete
	delResp := resource.DeleteResponse{State: updateResp.State}
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &delResp)
	if delResp.Diagnostics.HasError() {
		t.Fatalf("Delete document_type failed: %v", delResp.Diagnostics.Errors())
	}
}

// ---------------------------------------------------------------------------
// Unit tests for scm_dlp_edm_dataset
// ---------------------------------------------------------------------------

func TestPrimitivesEdmDataset(t *testing.T) {
	backend := newMockBackend()
	ts := setupTestServer(backend)
	defer ts.Close()

	// Seed existing EDM dataset in backend (as uploaded by EDM CLI)
	datasetID := "edm-seed-001"
	rName := "us"
	tID := "tenant-xyz"
	status := "COMPLETED"
	backend.edmDatasets[datasetID] = dlpclient.EdmDatasetDTO{
		ID:          &datasetID,
		Name:        "Customer_PII_Dataset",
		RegionName:  &rName,
		TenantID:    &tID,
		IndexStatus: &status,
		ActiveFields: []dlpclient.EdmFieldDTO{
			{ID: "f1", Name: "SSN", Type: "ssn"},
			{ID: "f2", Name: "FullName", Type: "name"},
		},
		AllFields: []dlpclient.EdmFieldDTO{
			{ID: "f1", Name: "SSN", Type: "ssn"},
			{ID: "f2", Name: "FullName", Type: "name"},
			{ID: "f3", Name: "Email", Type: "email"},
		},
	}

	client := createTestClient(ts)
	ctx := context.Background()
	res := NewEdmDatasetResource().(*EdmDatasetResource)
	res.client = client

	// 1. Adopt dataset by Name
	createModel := models.EdmDatasetModel{
		Name: types.StringValue("Customer_PII_Dataset"),
	}
	planState := tfsdk.State{Schema: models.EdmDatasetResourceSchema}
	createPlan := tfsdk.Plan{Schema: models.EdmDatasetResourceSchema}
	_ = createPlan.Set(ctx, &createModel)

	createReq := resource.CreateRequest{Plan: createPlan}
	createResp := resource.CreateResponse{State: planState}
	res.Create(ctx, createReq, &createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("Adopting EDM dataset failed: %v", createResp.Diagnostics.Errors())
	}

	var state models.EdmDatasetModel
	_ = createResp.State.Get(ctx, &state)
	if state.Id.ValueString() != datasetID {
		t.Errorf("Expected adopted ID %s, got %s", datasetID, state.Id.ValueString())
	}
	if len(state.ActiveFields) != 2 {
		t.Errorf("Expected 2 active fields, got %d", len(state.ActiveFields))
	}

	// 2. Read EDM dataset
	readReq := resource.ReadRequest{State: createResp.State}
	readResp := resource.ReadResponse{State: createResp.State}
	res.Read(ctx, readReq, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read EDM dataset failed: %v", readResp.Diagnostics.Errors())
	}

	// 3. DataSource & List DataSource
	ds := NewEdmDatasetDataSource().(*EdmDatasetDataSource)
	ds.client = client
	var dsData models.EdmDatasetModel
	dsData.Id = types.StringValue(datasetID)
	var dsConfigState tfsdk.State
	dsConfigState.Schema = models.EdmDatasetDataSourceSchema
	_ = dsConfigState.Set(ctx, &dsData)
	dsResp := datasource.ReadResponse{State: tfsdk.State{
		Schema: models.EdmDatasetDataSourceSchema,
	}}
	ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Raw:    dsConfigState.Raw,
		Schema: models.EdmDatasetDataSourceSchema,
	}}, &dsResp)
	if dsResp.Diagnostics.HasError() {
		t.Fatalf("EDM Dataset DataSource Read failed: %v", dsResp.Diagnostics.Errors())
	}

	listDS := NewEdmDatasetListDataSource().(*EdmDatasetListDataSource)
	listDS.client = client
	var listData models.EdmDatasetListModel
	listData.Name = types.StringValue("Customer_PII_Dataset")
	var listConfigState tfsdk.State
	listConfigState.Schema = models.EdmDatasetListDataSourceSchema
	_ = listConfigState.Set(ctx, &listData)
	listResp := datasource.ReadResponse{State: tfsdk.State{
		Schema: models.EdmDatasetListDataSourceSchema,
	}}
	listDS.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config{
		Raw:    listConfigState.Raw,
		Schema: models.EdmDatasetListDataSourceSchema,
	}}, &listResp)
	if listResp.Diagnostics.HasError() {
		t.Fatalf("EDM Dataset List DataSource Read failed: %v", listResp.Diagnostics.Errors())
	}
	var listResult models.EdmDatasetListModel
	_ = listResp.State.Get(ctx, &listResult)
	if listResult.Total.ValueInt64() != 1 {
		t.Errorf("Expected 1 dataset from list query, got %d", listResult.Total.ValueInt64())
	}

	// 4. Delete EDM dataset
	delResp := resource.DeleteResponse{State: createResp.State}
	res.Delete(ctx, resource.DeleteRequest{State: createResp.State}, &delResp)
	if delResp.Diagnostics.HasError() {
		t.Fatalf("Delete EDM dataset failed: %v", delResp.Diagnostics.Errors())
	}

	if _, ok := backend.edmDatasets[datasetID]; ok {
		t.Errorf("EDM dataset still exists in backend after Delete")
	}

	// 5. Test M13: index_status == "DELETED" treats resource as deleted in Read
	delDatasetID := "edm-seed-deleted"
	delStatus := "DELETED"
	backend.edmDatasets[delDatasetID] = dlpclient.EdmDatasetDTO{
		ID:          &delDatasetID,
		Name:        "Deleted_Dataset",
		IndexStatus: &delStatus,
	}
	var delStateModel models.EdmDatasetModel
	delStateModel.Id = types.StringValue(delDatasetID)
	delState := tfsdk.State{Schema: models.EdmDatasetResourceSchema}
	_ = delState.Set(ctx, &delStateModel)
	delReadResp := resource.ReadResponse{State: delState}
	res.Read(ctx, resource.ReadRequest{State: delState}, &delReadResp)
	if delReadResp.Diagnostics.HasError() {
		t.Fatalf("Read on DELETED status failed: %v", delReadResp.Diagnostics.Errors())
	}
	var resultAfterRead models.EdmDatasetModel
	_ = delReadResp.State.Get(ctx, &resultAfterRead)
	if !resultAfterRead.Id.IsNull() && resultAfterRead.Id.ValueString() != "" {
		t.Errorf("Expected resource state to be removed when index_status is DELETED, got id=%s", resultAfterRead.Id.ValueString())
	}
}

// Ensure schema and metadata calls don't crash
func TestPrimitivesSchemaAndMetadata(t *testing.T) {
	ctx := context.Background()

	pRes := NewDataPatternResource()
	var pMeta resource.MetadataResponse
	pRes.Metadata(ctx, resource.MetadataRequest{}, &pMeta)
	if pMeta.TypeName != "scm_dlp_data_pattern" {
		t.Errorf("unexpected type name: %s", pMeta.TypeName)
	}

	var pSchema resource.SchemaResponse
	pRes.Schema(ctx, resource.SchemaRequest{}, &pSchema)
	if pSchema.Schema.Attributes == nil {
		t.Errorf("expected non-nil attributes for pattern schema")
	}

	dRes := NewDictionaryResource()
	var dMeta resource.MetadataResponse
	dRes.Metadata(ctx, resource.MetadataRequest{}, &dMeta)
	if dMeta.TypeName != "scm_dlp_dictionary" {
		t.Errorf("unexpected type name: %s", dMeta.TypeName)
	}

	dtRes := NewDocumentTypeResource()
	var dtMeta resource.MetadataResponse
	dtRes.Metadata(ctx, resource.MetadataRequest{}, &dtMeta)
	if dtMeta.TypeName != "scm_dlp_document_type" {
		t.Errorf("unexpected type name: %s", dtMeta.TypeName)
	}

	edmRes := NewEdmDatasetResource()
	var edmMeta resource.MetadataResponse
	edmRes.Metadata(ctx, resource.MetadataRequest{}, &edmMeta)
	if edmMeta.TypeName != "scm_dlp_edm_dataset" {
		t.Errorf("unexpected type name: %s", edmMeta.TypeName)
	}

	// Register functions
	resFuncs := getPrimitivesResources()
	if len(resFuncs) != 4 {
		t.Errorf("expected 4 primitive resources, got %d", len(resFuncs))
	}
	dsFuncs := getPrimitivesDataSources()
	if len(dsFuncs) != 8 {
		t.Errorf("expected 8 primitive datasources, got %d", len(dsFuncs))
	}
}
