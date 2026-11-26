package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/dlpclient"
	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
)

type mockPoliciesBackend struct {
	mu               sync.Mutex
	endpointPolicies map[string]dlpclient.EndpointPolicyDTO
	peripherals      map[string]dlpclient.PeripheralDTO
	syslogProfiles   map[string]dlpclient.SyslogProfileDTO
	forwardProfiles  map[string]dlpclient.ForwardProfileDTO
	nextID           int
	configExportHit  bool
	dryRunScanHit    bool
}

func newMockPoliciesBackend() *mockPoliciesBackend {
	return &mockPoliciesBackend{
		endpointPolicies: make(map[string]dlpclient.EndpointPolicyDTO),
		peripherals:      make(map[string]dlpclient.PeripheralDTO),
		syslogProfiles:   make(map[string]dlpclient.SyslogProfileDTO),
		forwardProfiles:  make(map[string]dlpclient.ForwardProfileDTO),
		nextID:           100,
	}
}

func setupPoliciesMockServer(b *mockPoliciesBackend) *httptest.Server {
	mux := http.NewServeMux()

	// 1. Endpoint Policies (/v2/api/endpoint-policies)
	mux.HandleFunc("/v2/api/endpoint-policies", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			var list []dlpclient.EndpointPolicyDTO
			for _, v := range b.endpointPolicies {
				list = append(list, v)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":       list,
				"totalElements": len(list),
				"totalPages":    1,
				"last":          true,
			})
		case http.MethodPost:
			var dto dlpclient.EndpointPolicyDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			b.nextID++
			id := fmt.Sprintf("ep-%d", b.nextID)
			dto.ID = &id
			b.endpointPolicies[id] = dto
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(dto)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2/api/endpoint-policies/", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := strings.TrimPrefix(r.URL.Path, "/v2/api/endpoint-policies/")
		switch r.Method {
		case http.MethodGet:
			ep, exists := b.endpointPolicies[id]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ep)
		case http.MethodPut:
			var dto dlpclient.EndpointPolicyDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			dto.ID = &id
			b.endpointPolicies[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		case http.MethodDelete:
			delete(b.endpointPolicies, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 2. Peripherals (/v2/api/peripherals)
	mux.HandleFunc("/v2/api/peripherals", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			var list []dlpclient.PeripheralDTO
			for _, v := range b.peripherals {
				list = append(list, v)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":       list,
				"totalElements": len(list),
				"totalPages":    1,
				"last":          true,
			})
		case http.MethodPost:
			var dto dlpclient.PeripheralDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			b.nextID++
			id := fmt.Sprintf("periph-%d", b.nextID)
			dto.ID = &id
			b.peripherals[id] = dto
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(dto)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2/api/peripherals/", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := strings.TrimPrefix(r.URL.Path, "/v2/api/peripherals/")
		switch r.Method {
		case http.MethodGet:
			p, exists := b.peripherals[id]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(p)
		case http.MethodPut:
			var dto dlpclient.PeripheralDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			dto.ID = &id
			b.peripherals[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		case http.MethodDelete:
			delete(b.peripherals, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 3. Syslog Server Profile (/v1/api/syslog/server-profile)
	mux.HandleFunc("/v1/api/syslog/server-profile", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			var list []dlpclient.SyslogProfileDTO
			for _, v := range b.syslogProfiles {
				list = append(list, v)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items":       list,
				"total":       len(list),
				"total_pages": 1,
				"last":        true,
			})
		case http.MethodPost:
			var dto dlpclient.SyslogProfileDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			b.nextID++
			id := fmt.Sprintf("syslog-%d", b.nextID)
			dto.ID = &id
			dto.ProfileID = &id
			b.syslogProfiles[id] = dto
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(dto)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/api/syslog/server-profile/", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := strings.TrimPrefix(r.URL.Path, "/v1/api/syslog/server-profile/")
		switch r.Method {
		case http.MethodGet:
			sp, exists := b.syslogProfiles[id]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(sp)
		case http.MethodPut:
			var dto dlpclient.SyslogProfileDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			dto.ID = &id
			dto.ProfileID = &id
			b.syslogProfiles[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		case http.MethodDelete:
			delete(b.syslogProfiles, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 4. Forward Profile (/v1/api/forward-profile)
	mux.HandleFunc("/v1/api/forward-profile", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		switch r.Method {
		case http.MethodGet:
			var list []dlpclient.ForwardProfileDTO
			for _, v := range b.forwardProfiles {
				list = append(list, v)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items":       list,
				"total":       len(list),
				"total_pages": 1,
				"last":        true,
			})
		case http.MethodPost:
			var dto dlpclient.ForwardProfileDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			b.nextID++
			id := fmt.Sprintf("fp-%d", b.nextID)
			dto.ID = &id
			dto.ProfileID = &id
			b.forwardProfiles[id] = dto
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(dto)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/api/forward-profile/", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		id := strings.TrimPrefix(r.URL.Path, "/v1/api/forward-profile/")
		switch r.Method {
		case http.MethodGet:
			fp, exists := b.forwardProfiles[id]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(fp)
		case http.MethodPut:
			var dto dlpclient.ForwardProfileDTO
			if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			dto.ID = &id
			dto.ProfileID = &id
			b.forwardProfiles[id] = dto
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dto)
		case http.MethodDelete:
			delete(b.forwardProfiles, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 5. Action: Configuration Export & Import
	mux.HandleFunc("/v1/api/configuration/export", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		b.configExportHit = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(dlpclient.ConfigurationJobResponse{
			JobID:  "job-12345",
			Status: "COMPLETED",
		})
	})

	mux.HandleFunc("/v1/api/configuration/import", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file part", http.StatusBadRequest)
			return
		}
		defer file.Close()
		_, _ = io.ReadAll(file)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"job_id": "imp-67890",
			"status": "CONFLICT",
		})
	})

	mux.HandleFunc("/v1/api/configuration/import/job/", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"job_id": "imp-67890",
			"status": "COMPLETED",
			"action": r.URL.Query().Get("action"),
		})
	})

	// 6. Action: Dry Run Scan
	mux.HandleFunc("/v2/api/dry-run/scan", func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if r.FormValue("fileSha") == "" || r.FormValue("dataProfile") == "" {
			http.Error(w, "missing required multipart fields", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file part", http.StatusBadRequest)
			return
		}
		defer file.Close()
		_, _ = io.ReadAll(file)

		b.dryRunScanHit = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"report_id": "rep-9999",
			"verdict":   "NOT_MATCHED",
		})
	})

	return httptest.NewServer(mux)
}

func setupPoliciesClient(ts *httptest.Server) *dlpclient.Client {
	return &dlpclient.Client{
		HttpClient:  ts.Client(),
		BaseURL:     ts.URL,
		ServiceName: "prisma-access",
		ClientName:  "terraform-provider-scm",
	}
}

// ===========================================================================
// Test: Endpoint Policy CRUD + List
// ===========================================================================

func TestPolicies_EndpointPolicyCRUDAndList(t *testing.T) {
	backend := newMockPoliciesBackend()
	server := setupPoliciesMockServer(backend)
	defer server.Close()

	client := setupPoliciesClient(server)
	provData := map[string]any{"dlp": client}
	ctx := context.Background()

	// 1. Create Resource
	res := NewEndpointPolicyResource().(*EndpointPolicyResource)
	var configResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: provData}, &configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("Configure failed: %v", configResp.Diagnostics)
	}

	scopeJSON := `{"users":["alice@example.com"]}`
	respJSON := `{"action":"block"}`
	plan := models.EndpointPolicyModel{
		Name:         types.StringValue("Block Peripheral"),
		Description:  types.StringValue("Block unauthorized devices"),
		Enabled:      types.BoolValue(true),
		Type:         types.StringValue("endpoint_peripheral_control"),
		Severity:     types.StringValue("high"),
		Priority:     types.Int64Value(1),
		ScopeJson:    types.StringValue(scopeJSON),
		ResponseJson: types.StringValue(respJSON),
	}

	planVal := tfsdk.Plan{Schema: models.EndpointPolicyResourceSchema}
	if diags := planVal.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("Set plan failed: %v", diags)
	}

	createReq := resource.CreateRequest{Plan: planVal}
	var createResp resource.CreateResponse
	createResp.State = tfsdk.State{Schema: models.EndpointPolicyResourceSchema}
	res.Create(ctx, createReq, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", createResp.Diagnostics)
	}

	var state models.EndpointPolicyModel
	d := createResp.State.Get(ctx, &state)
	if d.HasError() {
		t.Fatalf("Get state failed: %v", d)
	}
	if state.Id.IsNull() || state.Id.ValueString() == "" {
		t.Fatal("Expected created endpoint policy to have an ID")
	}
	id := state.Id.ValueString()

	// 2. Read Resource
	var readResp resource.ReadResponse
	readResp.State = createResp.State
	res.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", readResp.Diagnostics)
	}
	var readState models.EndpointPolicyModel
	_ = readResp.State.Get(ctx, &readState)
	if readState.Name.ValueString() != "Block Peripheral" {
		t.Fatalf("Expected name 'Block Peripheral', got %q", readState.Name.ValueString())
	}

	// 3. Update Resource
	plan.Description = types.StringValue("Updated description")
	plan.Id = state.Id
	plan.Tfid = state.Tfid
	_ = planVal.Set(ctx, &plan)
	updateReq := resource.UpdateRequest{
		Plan:  planVal,
		State: readResp.State,
	}
	var updateResp resource.UpdateResponse
	updateResp.State = tfsdk.State{Schema: models.EndpointPolicyResourceSchema}
	res.Update(ctx, updateReq, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", updateResp.Diagnostics)
	}
	var updatedState models.EndpointPolicyModel
	_ = updateResp.State.Get(ctx, &updatedState)
	if updatedState.Description.ValueString() != "Updated description" {
		t.Fatalf("Expected description 'Updated description', got %q", updatedState.Description.ValueString())
	}

	// 4. Read Data Source
	ds := NewEndpointPolicyDataSource().(*EndpointPolicyDataSource)
	var dsConfigResp datasource.ConfigureResponse
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &dsConfigResp)

	dsStateVal := tfsdk.State{Schema: models.EndpointPolicyDataSourceSchema}
	dsStateVal.Set(ctx, &models.EndpointPolicyModel{Id: types.StringValue(id)})
	var dsReadResp datasource.ReadResponse
	dsReadResp.State = dsStateVal
	ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config(dsStateVal)}, &dsReadResp)
	if dsReadResp.Diagnostics.HasError() {
		t.Fatalf("DataSource Read failed: %v", dsReadResp.Diagnostics)
	}
	var dsState models.EndpointPolicyModel
	_ = dsReadResp.State.Get(ctx, &dsState)
	if dsState.Id.ValueString() != id {
		t.Fatalf("Expected DS ID %q, got %q", id, dsState.Id.ValueString())
	}

	// 5. Read List Data Source
	listDs := NewEndpointPolicyListDataSource().(*EndpointPolicyListDataSource)
	var listDsConfigResp datasource.ConfigureResponse
	listDs.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &listDsConfigResp)
	listStateVal := tfsdk.State{Schema: models.EndpointPolicyListDataSourceSchema}
	var listReadResp datasource.ReadResponse
	listReadResp.State = listStateVal
	listDs.Read(ctx, datasource.ReadRequest{
		Config: tfsdk.Config(listStateVal),
	}, &listReadResp)
	if listReadResp.Diagnostics.HasError() {
		t.Fatalf("ListDataSource Read failed: %v", listReadResp.Diagnostics)
	}
	var listState models.EndpointPolicyListModel
	_ = listReadResp.State.Get(ctx, &listState)
	if listState.Total.ValueInt64() != 1 || len(listState.Data) != 1 {
		t.Fatalf("Expected 1 item in list, got %d", listState.Total.ValueInt64())
	}

	// 6. Delete Resource
	var delResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &delResp)
	if delResp.Diagnostics.HasError() {
		t.Fatalf("Delete failed: %v", delResp.Diagnostics)
	}

	// Verify deleted
	readResp = resource.ReadResponse{State: updateResp.State}
	res.Read(ctx, resource.ReadRequest{State: updateResp.State}, &readResp)
	if !readResp.State.Raw.IsNull() {
		t.Fatal("Expected state to be removed after delete")
	}
}

// ===========================================================================
// Test: Peripheral CRUD + List
// ===========================================================================

func TestPolicies_PeripheralCRUDAndList(t *testing.T) {
	backend := newMockPoliciesBackend()
	server := setupPoliciesMockServer(backend)
	defer server.Close()

	client := setupPoliciesClient(server)
	provData := map[string]any{"dlp": client}
	ctx := context.Background()

	// 1. Create
	res := NewPeripheralResource().(*PeripheralResource)
	var configResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: provData}, &configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("Configure failed: %v", configResp.Diagnostics)
	}

	plan := models.PeripheralModel{
		Name:        types.StringValue("Corp USB"),
		Description: types.StringValue("SanDisk Cruzer"),
		Type:        types.StringValue("usb"),
		VendorId:    types.StringValue("0781"),
		ProductId:   types.StringValue("5567"),
	}

	planVal := tfsdk.Plan{Schema: models.PeripheralResourceSchema}
	if diags := planVal.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("Set plan failed: %v", diags)
	}

	createReq := resource.CreateRequest{Plan: planVal}
	var createResp resource.CreateResponse
	createResp.State = tfsdk.State{Schema: models.PeripheralResourceSchema}
	res.Create(ctx, createReq, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", createResp.Diagnostics)
	}

	var state models.PeripheralModel
	_ = createResp.State.Get(ctx, &state)
	id := state.Id.ValueString()
	if id == "" {
		t.Fatal("Expected created peripheral to have an ID")
	}

	// 2. Read
	var readResp resource.ReadResponse
	readResp.State = createResp.State
	res.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", readResp.Diagnostics)
	}

	// 3. Update
	plan.Description = types.StringValue("Updated Peripheral Description")
	plan.Id = state.Id
	plan.Tfid = state.Tfid
	_ = planVal.Set(ctx, &plan)
	var updateResp resource.UpdateResponse
	updateResp.State = tfsdk.State{Schema: models.PeripheralResourceSchema}
	res.Update(ctx, resource.UpdateRequest{
		Plan:  planVal,
		State: readResp.State,
	}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", updateResp.Diagnostics)
	}

	// 4. Data Source
	ds := NewPeripheralDataSource().(*PeripheralDataSource)
	var dsConfigResp datasource.ConfigureResponse
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &dsConfigResp)

	dsStateVal := tfsdk.State{Schema: models.PeripheralDataSourceSchema}
	dsStateVal.Set(ctx, &models.PeripheralModel{Id: types.StringValue(id)})
	var dsReadResp datasource.ReadResponse
	dsReadResp.State = dsStateVal
	ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config(dsStateVal)}, &dsReadResp)
	if dsReadResp.Diagnostics.HasError() {
		t.Fatalf("DataSource Read failed: %v", dsReadResp.Diagnostics)
	}

	// 5. List Data Source
	listDs := NewPeripheralListDataSource().(*PeripheralListDataSource)
	var listDsConfigResp datasource.ConfigureResponse
	listDs.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &listDsConfigResp)
	listStateVal := tfsdk.State{Schema: models.PeripheralListDataSourceSchema}
	var listReadResp datasource.ReadResponse
	listReadResp.State = listStateVal
	listDs.Read(ctx, datasource.ReadRequest{
		Config: tfsdk.Config(listStateVal),
	}, &listReadResp)
	if listReadResp.Diagnostics.HasError() {
		t.Fatalf("ListDataSource Read failed: %v", listReadResp.Diagnostics)
	}
	var listState models.PeripheralListModel
	_ = listReadResp.State.Get(ctx, &listState)
	if listState.Total.ValueInt64() != 1 {
		t.Fatalf("Expected 1 peripheral in list, got %d", listState.Total.ValueInt64())
	}

	// 6. Delete
	var delResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &delResp)
	if delResp.Diagnostics.HasError() {
		t.Fatalf("Delete failed: %v", delResp.Diagnostics)
	}
}

// ===========================================================================
// Test: Syslog Profile CRUD + List
// ===========================================================================

func TestPolicies_SyslogProfileCRUDAndList(t *testing.T) {
	backend := newMockPoliciesBackend()
	server := setupPoliciesMockServer(backend)
	defer server.Close()

	client := setupPoliciesClient(server)
	provData := map[string]any{"dlp": client}
	ctx := context.Background()

	res := NewSyslogProfileResource().(*SyslogProfileResource)
	var configResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: provData}, &configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("Configure failed: %v", configResp.Diagnostics)
	}

	plan := models.SyslogProfileModel{
		ProfileName:    types.StringValue("SIEM Syslog"),
		Address:        types.StringValue("192.168.1.100"),
		Port:           types.Int64Value(514),
		AlertEmail:     types.StringValue("secops@example.com"),
		Format:         types.StringValue("CEF"),
		ConnectionType: types.StringValue("TCP"),
	}

	planVal := tfsdk.Plan{Schema: models.SyslogProfileResourceSchema}
	if diags := planVal.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("Set plan failed: %v", diags)
	}

	var createResp resource.CreateResponse
	createResp.State = tfsdk.State{Schema: models.SyslogProfileResourceSchema}
	res.Create(ctx, resource.CreateRequest{
		Plan: planVal,
	}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", createResp.Diagnostics)
	}

	var state models.SyslogProfileModel
	_ = createResp.State.Get(ctx, &state)
	id := state.Id.ValueString()
	if id == "" {
		t.Fatal("Expected created syslog profile to have an ID")
	}

	// Read
	var readResp resource.ReadResponse
	readResp.State = createResp.State
	res.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", readResp.Diagnostics)
	}

	// Update
	plan.Port = types.Int64Value(6514)
	plan.Id = state.Id
	plan.Tfid = state.Tfid
	_ = planVal.Set(ctx, &plan)
	var updateResp resource.UpdateResponse
	updateResp.State = tfsdk.State{Schema: models.SyslogProfileResourceSchema}
	res.Update(ctx, resource.UpdateRequest{
		Plan:  planVal,
		State: readResp.State,
	}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", updateResp.Diagnostics)
	}

	// Data Source
	ds := NewSyslogProfileDataSource().(*SyslogProfileDataSource)
	var dsConfigResp datasource.ConfigureResponse
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &dsConfigResp)

	dsStateVal := tfsdk.State{Schema: models.SyslogProfileDataSourceSchema}
	dsStateVal.Set(ctx, &models.SyslogProfileModel{Id: types.StringValue(id)})
	var dsReadResp datasource.ReadResponse
	dsReadResp.State = dsStateVal
	ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config(dsStateVal)}, &dsReadResp)
	if dsReadResp.Diagnostics.HasError() {
		t.Fatalf("DataSource Read failed: %v", dsReadResp.Diagnostics)
	}

	// List Data Source
	listDs := NewSyslogProfileListDataSource().(*SyslogProfileListDataSource)
	var listDsConfigResp datasource.ConfigureResponse
	listDs.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &listDsConfigResp)
	listStateVal := tfsdk.State{Schema: models.SyslogProfileListDataSourceSchema}
	var listReadResp datasource.ReadResponse
	listReadResp.State = listStateVal
	listDs.Read(ctx, datasource.ReadRequest{
		Config: tfsdk.Config(listStateVal),
	}, &listReadResp)
	if listReadResp.Diagnostics.HasError() {
		t.Fatalf("ListDataSource Read failed: %v", listReadResp.Diagnostics)
	}
	var listState models.SyslogProfileListModel
	_ = listReadResp.State.Get(ctx, &listState)
	if listState.Total.ValueInt64() != 1 {
		t.Fatalf("Expected 1 syslog profile in list, got %d", listState.Total.ValueInt64())
	}

	// Delete
	var delResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &delResp)
	if delResp.Diagnostics.HasError() {
		t.Fatalf("Delete failed: %v", delResp.Diagnostics)
	}
}

// ===========================================================================
// Test: Forward Profile CRUD + List
// ===========================================================================

func TestPolicies_ForwardProfileCRUDAndList(t *testing.T) {
	backend := newMockPoliciesBackend()
	server := setupPoliciesMockServer(backend)
	defer server.Close()

	client := setupPoliciesClient(server)
	provData := map[string]any{"dlp": client}
	ctx := context.Background()

	res := NewForwardProfileResource().(*ForwardProfileResource)
	var configResp resource.ConfigureResponse
	res.Configure(ctx, resource.ConfigureRequest{ProviderData: provData}, &configResp)
	if configResp.Diagnostics.HasError() {
		t.Fatalf("Configure failed: %v", configResp.Diagnostics)
	}

	regList, _ := types.ListValueFrom(ctx, types.StringType, []string{"us", "eu"})
	plan := models.ForwardProfileModel{
		Name:    types.StringValue("Forward US EU"),
		Enabled: types.BoolValue(true),
		Regions: regList,
	}

	planVal := tfsdk.Plan{Schema: models.ForwardProfileResourceSchema}
	if diags := planVal.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("Set plan failed: %v", diags)
	}

	var createResp resource.CreateResponse
	createResp.State = tfsdk.State{Schema: models.ForwardProfileResourceSchema}
	res.Create(ctx, resource.CreateRequest{
		Plan: planVal,
	}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create failed: %v", createResp.Diagnostics)
	}

	var state models.ForwardProfileModel
	_ = createResp.State.Get(ctx, &state)
	id := state.Id.ValueString()
	if id == "" {
		t.Fatal("Expected created forward profile to have an ID")
	}

	// Read
	var readResp resource.ReadResponse
	readResp.State = createResp.State
	res.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read failed: %v", readResp.Diagnostics)
	}

	// Update
	regListUpdated, _ := types.ListValueFrom(ctx, types.StringType, []string{"us", "eu", "ap"})
	plan.Regions = regListUpdated
	plan.Id = state.Id
	plan.Tfid = state.Tfid
	_ = planVal.Set(ctx, &plan)
	var updateResp resource.UpdateResponse
	updateResp.State = tfsdk.State{Schema: models.ForwardProfileResourceSchema}
	res.Update(ctx, resource.UpdateRequest{
		Plan:  planVal,
		State: readResp.State,
	}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update failed: %v", updateResp.Diagnostics)
	}

	// Data Source
	ds := NewForwardProfileDataSource().(*ForwardProfileDataSource)
	var dsConfigResp datasource.ConfigureResponse
	ds.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &dsConfigResp)

	dsStateVal := tfsdk.State{Schema: models.ForwardProfileDataSourceSchema}
	dsStateVal.Set(ctx, &models.ForwardProfileModel{
		Id:      types.StringValue(id),
		Tfid:    types.StringNull(),
		Name:    types.StringNull(),
		Enabled: types.BoolNull(),
		Regions: types.ListNull(types.StringType),
	})
	var dsReadResp datasource.ReadResponse
	dsReadResp.State = dsStateVal
	ds.Read(ctx, datasource.ReadRequest{Config: tfsdk.Config(dsStateVal)}, &dsReadResp)
	if dsReadResp.Diagnostics.HasError() {
		t.Fatalf("DataSource Read failed: %v", dsReadResp.Diagnostics)
	}

	// List Data Source
	listDs := NewForwardProfileListDataSource().(*ForwardProfileListDataSource)
	var listDsConfigResp datasource.ConfigureResponse
	listDs.Configure(ctx, datasource.ConfigureRequest{ProviderData: provData}, &listDsConfigResp)
	listStateVal := tfsdk.State{Schema: models.ForwardProfileListDataSourceSchema}
	var listReadResp datasource.ReadResponse
	listReadResp.State = listStateVal
	listDs.Read(ctx, datasource.ReadRequest{
		Config: tfsdk.Config(listStateVal),
	}, &listReadResp)
	if listReadResp.Diagnostics.HasError() {
		t.Fatalf("ListDataSource Read failed: %v", listReadResp.Diagnostics)
	}
	var listState models.ForwardProfileListModel
	_ = listReadResp.State.Get(ctx, &listState)
	if listState.Total.ValueInt64() != 1 {
		t.Fatalf("Expected 1 forward profile in list, got %d", listState.Total.ValueInt64())
	}

	// Delete
	var delResp resource.DeleteResponse
	res.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &delResp)
	if delResp.Diagnostics.HasError() {
		t.Fatalf("Delete failed: %v", delResp.Diagnostics)
	}
}

// ===========================================================================
// Test: Terraform Actions (scm_dlp_config_export & scm_dlp_dry_run_scan)
// ===========================================================================

func TestPolicies_Actions(t *testing.T) {
	backend := newMockPoliciesBackend()
	server := setupPoliciesMockServer(backend)
	defer server.Close()

	client := setupPoliciesClient(server)
	provData := map[string]any{"dlp": client}
	ctx := context.Background()

	// 1. Config Export Action
	exportAction := NewConfigExportAction().(*ConfigExportAction)
	var exportConfigResp action.ConfigureResponse
	exportAction.Configure(ctx, action.ConfigureRequest{ProviderData: (any)(provData)}, &exportConfigResp)
	if exportConfigResp.Diagnostics.HasError() {
		t.Fatalf("ConfigExport Configure failed: %v", exportConfigResp.Diagnostics)
	}
	if exportAction.client == nil {
		t.Fatal("exportAction.client is nil after Configure")
	}

	exportData := ConfigExportActionModel{
		ExportType: types.StringValue("JSON"),
	}
	var exportSchemaResp action.SchemaResponse
	exportAction.Schema(ctx, action.SchemaRequest{}, &exportSchemaResp)
	exportStateVal := tfsdk.State{Schema: exportSchemaResp.Schema}
	exportStateVal.Set(ctx, &exportData)

	var exportInvokeResp action.InvokeResponse
	exportAction.Invoke(ctx, action.InvokeRequest{Config: tfsdk.Config(exportStateVal)}, &exportInvokeResp)
	if exportInvokeResp.Diagnostics.HasError() {
		t.Fatalf("ConfigExport Invoke failed: %v", exportInvokeResp.Diagnostics)
	}
	if !backend.configExportHit {
		t.Fatal("Expected configuration export endpoint to be called")
	}

	// 2. Dry Run Scan Action
	scanAction := NewDryRunScanAction().(*DryRunScanAction)
	var scanConfigResp action.ConfigureResponse
	scanAction.Configure(ctx, action.ConfigureRequest{ProviderData: provData}, &scanConfigResp)
	if scanConfigResp.Diagnostics.HasError() {
		t.Fatalf("DryRunScan Configure failed: %v", scanConfigResp.Diagnostics)
	}

	scanData := DryRunScanActionModel{
		DataProfile: types.StringValue(`{"name":"test-profile"}`),
		FileContent: types.StringValue("credit card number: 4111111111111111"),
		FileName:    types.StringValue("card.txt"),
	}
	var scanSchemaResp action.SchemaResponse
	scanAction.Schema(ctx, action.SchemaRequest{}, &scanSchemaResp)
	scanStateVal := tfsdk.State{Schema: scanSchemaResp.Schema}
	scanStateVal.Set(ctx, &scanData)

	var scanInvokeResp action.InvokeResponse
	scanAction.Invoke(ctx, action.InvokeRequest{Config: tfsdk.Config(scanStateVal)}, &scanInvokeResp)
	if scanInvokeResp.Diagnostics.HasError() {
		t.Fatalf("DryRunScan Invoke failed: %v", scanInvokeResp.Diagnostics)
	}
	if !backend.dryRunScanHit {
		t.Fatal("Expected dry run scan endpoint to be called")
	}

	// 3. Config Import Action (upload + conflict resolution)
	importAction := NewConfigImportAction().(*ConfigImportAction)
	var importConfigResp action.ConfigureResponse
	importAction.Configure(ctx, action.ConfigureRequest{ProviderData: provData}, &importConfigResp)
	if importConfigResp.Diagnostics.HasError() {
		t.Fatalf("ConfigImport Configure failed: %v", importConfigResp.Diagnostics)
	}

	importData := ConfigImportActionModel{
		FileContent:    types.StringValue(`{"profiles":[]}`),
		FileName:       types.StringValue("export.json"),
		ConflictAction: types.StringValue("CONTINUE"),
	}
	var importSchemaResp action.SchemaResponse
	importAction.Schema(ctx, action.SchemaRequest{}, &importSchemaResp)
	importStateVal := tfsdk.State{Schema: importSchemaResp.Schema}
	importStateVal.Set(ctx, &importData)

	var importInvokeResp action.InvokeResponse
	importAction.Invoke(ctx, action.InvokeRequest{Config: tfsdk.Config(importStateVal)}, &importInvokeResp)
	if importInvokeResp.Diagnostics.HasError() {
		t.Fatalf("ConfigImport Invoke failed: %v", importInvokeResp.Diagnostics)
	}

	// 4. File Upload Path Validation (M9: reject /etc/passwd and ~/.ssh/*)
	if _, _, err := readValidatedUploadFile("/etc/passwd"); err == nil {
		t.Fatal("Expected readValidatedUploadFile to reject /etc/passwd")
	}
	if _, _, err := readValidatedUploadFile(""); err == nil {
		t.Fatal("Expected readValidatedUploadFile to reject empty path")
	}
}

// Suppress unused imports check
var _ = strconv.Itoa
