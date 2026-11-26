package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
	"github.com/paloaltonetworks/terraform-provider-scm/internal/provider/dlp/client"
)

// ===========================================================================
// Resource: scm_dlp_endpoint_policy
// ===========================================================================

var (
	_ resource.Resource                = &EndpointPolicyResource{}
	_ resource.ResourceWithConfigure   = &EndpointPolicyResource{}
	_ resource.ResourceWithImportState = &EndpointPolicyResource{}
)

func NewEndpointPolicyResource() resource.Resource {
	return &EndpointPolicyResource{}
}

type EndpointPolicyResource struct {
	client *dlpclient.Client
}

func (r *EndpointPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_endpoint_policy"
}

func (r *EndpointPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.EndpointPolicyResourceSchema
}

func (r *EndpointPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *EndpointPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.EndpointPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto, diags := endpointPolicyModelToDTO(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating DLP Endpoint Policy", map[string]any{"name": dto.Name})
	var created dlpclient.EndpointPolicyDTO
	if err := r.client.Post(ctx, "/v2/api/endpoint-policies", nil, dto, &created); err != nil {
		resp.Diagnostics.AddError("Error Creating DLP Endpoint Policy", err.Error())
		return
	}

	state, diags := endpointPolicyDTOToModel(&created, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EndpointPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.EndpointPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Endpoint Policy without an ID.")
		return
	}

	var dto dlpclient.EndpointPolicyDTO
	err := r.client.Get(ctx, fmt.Sprintf("/v2/api/endpoint-policies/%s", dlpclient.EscapeID(id)), nil, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading DLP Endpoint Policy", err.Error())
		return
	}

	updated, diags := endpointPolicyDTOToModel(&dto, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *EndpointPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.EndpointPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state models.EndpointPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = plan.Id.ValueString()
	}
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot update Endpoint Policy without an ID.")
		return
	}

	dto, diags := endpointPolicyModelToDTO(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto.ID = &id

	var updated dlpclient.EndpointPolicyDTO
	if err := r.client.Put(ctx, fmt.Sprintf("/v2/api/endpoint-policies/%s", dlpclient.EscapeID(id)), nil, dto, &updated); err != nil {
		resp.Diagnostics.AddError("Error Updating DLP Endpoint Policy", err.Error())
		return
	}

	newState, diags := endpointPolicyDTOToModel(&updated, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *EndpointPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.EndpointPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Endpoint Policy without an ID.")
		return
	}

	if err := r.client.Delete(ctx, fmt.Sprintf("/v2/api/endpoint-policies/%s", dlpclient.EscapeID(id)), nil); err != nil {
		if !dlpclient.IsNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting DLP Endpoint Policy", err.Error())
			return
		}
	}
}

func (r *EndpointPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// DataSource: scm_dlp_endpoint_policy
// ===========================================================================

var (
	_ datasource.DataSource              = &EndpointPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &EndpointPolicyDataSource{}
)

func NewEndpointPolicyDataSource() datasource.DataSource {
	return &EndpointPolicyDataSource{}
}

type EndpointPolicyDataSource struct {
	client *dlpclient.Client
}

func (d *EndpointPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_endpoint_policy"
}

func (d *EndpointPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.EndpointPolicyDataSourceSchema
}

func (d *EndpointPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *EndpointPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.EndpointPolicyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Attribute 'id' is required for scm_dlp_endpoint_policy data source.")
		return
	}

	var dto dlpclient.EndpointPolicyDTO
	if err := d.client.Get(ctx, fmt.Sprintf("/v2/api/endpoint-policies/%s", dlpclient.EscapeID(id)), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Error Reading DLP Endpoint Policy", err.Error())
		return
	}

	state, diags := endpointPolicyDTOToModel(&dto, nil)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ===========================================================================
// DataSource: scm_dlp_endpoint_policy_list
// ===========================================================================

var (
	_ datasource.DataSource              = &EndpointPolicyListDataSource{}
	_ datasource.DataSourceWithConfigure = &EndpointPolicyListDataSource{}
)

func NewEndpointPolicyListDataSource() datasource.DataSource {
	return &EndpointPolicyListDataSource{}
}

type EndpointPolicyListDataSource struct {
	client *dlpclient.Client
}

func (d *EndpointPolicyListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_endpoint_policy_list"
}

func (d *EndpointPolicyListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.EndpointPolicyListDataSourceSchema
}

func (d *EndpointPolicyListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *EndpointPolicyListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	allPolicies, err := dlpclient.GetAll[dlpclient.EndpointPolicyDTO](
		ctx,
		d.client,
		"/v2/api/endpoint-policies",
		nil,
		dlpclient.PaginationZeroBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DLP Endpoint Policies", err.Error())
		return
	}

	var policyModels []models.EndpointPolicyModel
	for i := range allPolicies {
		p := &allPolicies[i]
		m, diags := endpointPolicyDTOToModel(p, nil)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		policyModels = append(policyModels, m)
	}

	data := models.EndpointPolicyListModel{
		Total: types.Int64Value(int64(len(policyModels))),
		Data:  policyModels,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ===========================================================================
// Helpers: Endpoint Policy Model <-> DTO
// ===========================================================================

func endpointPolicyModelToDTO(m *models.EndpointPolicyModel) (*dlpclient.EndpointPolicyDTO, diag.Diagnostics) {
	var diags diag.Diagnostics

	dto := &dlpclient.EndpointPolicyDTO{
		Name:     m.Name.ValueString(),
		Enabled:  m.Enabled.ValueBool(),
		Type:     m.Type.ValueString(),
		Severity: m.Severity.ValueString(),
	}

	if !m.Id.IsNull() && !m.Id.IsUnknown() && m.Id.ValueString() != "" {
		v := m.Id.ValueString()
		dto.ID = &v
	}
	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		v := m.Description.ValueString()
		dto.Description = &v
	}
	if !m.Priority.IsNull() && !m.Priority.IsUnknown() {
		v := m.Priority.ValueInt64()
		dto.Priority = &v
	}

	scopeStr := strings.TrimSpace(m.ScopeJson.ValueString())
	if scopeStr != "" {
		if !json.Valid([]byte(scopeStr)) {
			diags.AddError("Invalid scope_json", "scope_json must be a valid JSON string.")
		} else {
			dto.Scope = json.RawMessage(scopeStr)
		}
	} else {
		dto.Scope = json.RawMessage("{}")
	}

	respStr := strings.TrimSpace(m.ResponseJson.ValueString())
	if respStr != "" {
		if !json.Valid([]byte(respStr)) {
			diags.AddError("Invalid response_json", "response_json must be a valid JSON string.")
		} else {
			dto.Response = json.RawMessage(respStr)
		}
	} else {
		dto.Response = json.RawMessage("{}")
	}

	return dto, diags
}

func jsonStringsSemanticallyEqual(a, b string) bool {
	var va, vb any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}
	return reflect.DeepEqual(va, vb)
}

func endpointPolicyDTOToModel(dto *dlpclient.EndpointPolicyDTO, prior *models.EndpointPolicyModel) (models.EndpointPolicyModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var idStr string
	if dto.ID != nil {
		idStr = *dto.ID
	}

	m := models.EndpointPolicyModel{
		Tfid:         types.StringValue(idStr),
		Id:           types.StringValue(idStr),
		Name:         types.StringValue(dto.Name),
		Description:  types.StringNull(),
		Enabled:      types.BoolValue(dto.Enabled),
		Type:         types.StringValue(dto.Type),
		Severity:     types.StringValue(dto.Severity),
		Priority:     types.Int64Null(),
		ScopeJson:    jsontypes.NewNormalizedNull(),
		ResponseJson: jsontypes.NewNormalizedNull(),
	}

	if dto.Description != nil {
		m.Description = types.StringValue(*dto.Description)
	}
	if dto.Priority != nil {
		m.Priority = types.Int64Value(*dto.Priority)
	}

	if len(dto.Scope) > 0 {
		serverScope := string(dto.Scope)
		if prior != nil && !prior.ScopeJson.IsNull() && !prior.ScopeJson.IsUnknown() && jsonStringsSemanticallyEqual(prior.ScopeJson.ValueString(), serverScope) {
			m.ScopeJson = prior.ScopeJson
		} else {
			m.ScopeJson = jsontypes.NewNormalizedValue(serverScope)
		}
	} else {
		m.ScopeJson = jsontypes.NewNormalizedValue("{}")
	}

	if len(dto.Response) > 0 {
		serverResp := string(dto.Response)
		if prior != nil && !prior.ResponseJson.IsNull() && !prior.ResponseJson.IsUnknown() && jsonStringsSemanticallyEqual(prior.ResponseJson.ValueString(), serverResp) {
			m.ResponseJson = prior.ResponseJson
		} else {
			m.ResponseJson = jsontypes.NewNormalizedValue(serverResp)
		}
	} else {
		m.ResponseJson = jsontypes.NewNormalizedValue("{}")
	}

	return m, diags
}
