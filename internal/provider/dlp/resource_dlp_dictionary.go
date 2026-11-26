package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/dlpclient"
	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
)

// ===========================================================================
// Resource: scm_dlp_dictionary
// ===========================================================================

var (
	_ resource.Resource                = &DictionaryResource{}
	_ resource.ResourceWithConfigure   = &DictionaryResource{}
	_ resource.ResourceWithImportState = &DictionaryResource{}
)

func NewDictionaryResource() resource.Resource {
	return &DictionaryResource{}
}

type DictionaryResource struct {
	client *dlpclient.Client
}

func (r *DictionaryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "scm_dlp_dictionary"
}

func (r *DictionaryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.DictionaryResourceSchema
}

func (r *DictionaryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *DictionaryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DictionaryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto, keywords, diags := dictionaryModelToDTO(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating DLP Dictionary metadata", map[string]any{"name": dto.Name})
	var created dlpclient.DictionaryDTO
	if err := r.client.Post(ctx, "/v2/api/dictionaries", nil, dto, &created); err != nil {
		resp.Diagnostics.AddError("Error Creating DLP Dictionary", err.Error())
		return
	}

	createdID := ""
	if created.ID != nil {
		createdID = *created.ID
	}
	if createdID == "" {
		resp.Diagnostics.AddError("Error Creating DLP Dictionary", "Backend returned dictionary without an ID.")
		return
	}

	// If keyword_list is provided on Create, after POST returns created ID,
	// also call multipart PUT /v2/api/dictionaries/{id}?keywords=true with BuildDictionaryUpdateParts.
	if len(keywords) > 0 {
		created.ID = &createdID
		parts, valErrs, err := dlpclient.BuildDictionaryUpdateParts(&created, keywords)
		if err != nil {
			for _, e := range valErrs {
				resp.Diagnostics.AddError("Dictionary Keywords Validation Failed", e)
			}
			if len(valErrs) == 0 {
				resp.Diagnostics.AddError("Dictionary Keywords Validation Failed", err.Error())
			}
			return
		}

		tflog.Debug(ctx, "Uploading DLP Dictionary keywords via multipart PUT", map[string]any{"id": createdID, "count": len(keywords)})
		var updated dlpclient.DictionaryDTO
		params := map[string]string{"keywords": "true"}
		putErr := r.client.DoMultipart(ctx, "PUT", fmt.Sprintf("/v2/api/dictionaries/%s", createdID), params, parts, &updated)
		if putErr != nil {
			resp.Diagnostics.AddError("Error Uploading DLP Dictionary Keywords", putErr.Error())
			return
		}
		created = updated
	}

	// Fetch full dictionary details with keywords=true
	var finalDTO dlpclient.DictionaryDTO
	readParams := map[string]string{"keywords": "true"}
	if err := r.client.Get(ctx, fmt.Sprintf("/v2/api/dictionaries/%s", createdID), readParams, &finalDTO); err != nil {
		// Fallback to created if GET fails
		finalDTO = created
	}

	state, diags := dictionaryDTOToModel(ctx, &finalDTO, keywords)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DictionaryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.DictionaryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Dictionary without an ID.")
		return
	}

	tflog.Debug(ctx, "Reading DLP Dictionary", map[string]any{"id": id})
	var dto dlpclient.DictionaryDTO
	params := map[string]string{"keywords": "true"}
	err := r.client.Get(ctx, fmt.Sprintf("/v2/api/dictionaries/%s", id), params, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading DLP Dictionary", err.Error())
		return
	}

	var planKeywords []string
	if !state.KeywordList.IsNull() {
		_ = state.KeywordList.ElementsAs(ctx, &planKeywords, false)
	}

	newState, diags := dictionaryDTOToModel(ctx, &dto, planKeywords)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *DictionaryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DictionaryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.Id.ValueString()
	if id == "" {
		id = plan.Tfid.ValueString()
	}

	dto, keywords, diags := dictionaryModelToDTO(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In case required fields are not set in the plan, fallback to state
	var state models.DictionaryModel
	_ = req.State.Get(ctx, &state)
	if strings.TrimSpace(dto.Name) == "" && !state.Name.IsNull() {
		dto.Name = state.Name.ValueString()
	}
	if strings.TrimSpace(dto.Category) == "" && !state.Category.IsNull() {
		dto.Category = state.Category.ValueString()
	}
	if strings.TrimSpace(dto.OriginalFileName) == "" && !state.OriginalFileName.IsNull() {
		dto.OriginalFileName = state.OriginalFileName.ValueString()
	}
	if strings.TrimSpace(dto.RegionName) == "" && !state.RegionName.IsNull() {
		dto.RegionName = state.RegionName.ValueString()
	}
	if dto.Description == nil && !state.Description.IsNull() {
		desc := state.Description.ValueString()
		dto.Description = &desc
	}
	if dto.IsCaseSensitive == nil && !state.IsCaseSensitive.IsNull() {
		cs := state.IsCaseSensitive.ValueBool()
		dto.IsCaseSensitive = &cs
	}
	if id == "" && !state.Id.IsNull() {
		id = state.Id.ValueString()
	}
	dto.ID = &id

	parts, valErrs, err := dlpclient.BuildDictionaryUpdateParts(dto, keywords)
	if err != nil {
		for _, e := range valErrs {
			resp.Diagnostics.AddError("Dictionary Validation Failed", e)
		}
		if len(valErrs) == 0 {
			resp.Diagnostics.AddError("Dictionary Validation Failed", err.Error())
		}
		return
	}

	tflog.Debug(ctx, "Updating DLP Dictionary via multipart PUT", map[string]any{"id": id})
	var updated dlpclient.DictionaryDTO
	params := map[string]string{"keywords": "true"}
	if err := r.client.DoMultipart(ctx, "PUT", fmt.Sprintf("/v2/api/dictionaries/%s", id), params, parts, &updated); err != nil {
		resp.Diagnostics.AddError("Error Updating DLP Dictionary", err.Error())
		return
	}

	// Fetch refreshed dictionary with keywords
	var finalDTO dlpclient.DictionaryDTO
	if err := r.client.Get(ctx, fmt.Sprintf("/v2/api/dictionaries/%s", id), params, &finalDTO); err != nil {
		finalDTO = updated
	}

	resState, diags := dictionaryDTOToModel(ctx, &finalDTO, keywords)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &resState)...)
}

func (r *DictionaryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.DictionaryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		return
	}

	tflog.Debug(ctx, "Deleting DLP Dictionary", map[string]any{"id": id})
	if err := r.client.Delete(ctx, fmt.Sprintf("/v2/api/dictionaries/%s", id), nil); err != nil {
		if !dlpclient.IsNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting DLP Dictionary", err.Error())
			return
		}
	}
}

func (r *DictionaryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// DataSource: scm_dlp_dictionary
// ===========================================================================

var (
	_ datasource.DataSource              = &DictionaryDataSource{}
	_ datasource.DataSourceWithConfigure = &DictionaryDataSource{}
)

func NewDictionaryDataSource() datasource.DataSource {
	return &DictionaryDataSource{}
}

type DictionaryDataSource struct {
	client *dlpclient.Client
}

func (d *DictionaryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_dictionary"
}

func (d *DictionaryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DictionaryDataSourceSchema
}

func (d *DictionaryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *DictionaryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.DictionaryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Attribute 'id' is required for scm_dlp_dictionary data source.")
		return
	}

	var dto dlpclient.DictionaryDTO
	params := map[string]string{"keywords": "true"}
	if err := d.client.Get(ctx, fmt.Sprintf("/v2/api/dictionaries/%s", id), params, &dto); err != nil {
		resp.Diagnostics.AddError("Error Reading DLP Dictionary", err.Error())
		return
	}

	state, diags := dictionaryDTOToModel(ctx, &dto, nil)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ===========================================================================
// DataSource: scm_dlp_dictionary_list
// ===========================================================================

var (
	_ datasource.DataSource              = &DictionaryListDataSource{}
	_ datasource.DataSourceWithConfigure = &DictionaryListDataSource{}
)

func NewDictionaryListDataSource() datasource.DataSource {
	return &DictionaryListDataSource{}
}

type DictionaryListDataSource struct {
	client *dlpclient.Client
}

func (d *DictionaryListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_dictionary_list"
}

func (d *DictionaryListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DictionaryListDataSourceSchema
}

func (d *DictionaryListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *DictionaryListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	allDicts, err := dlpclient.GetAll[dlpclient.DictionaryDTO](
		ctx,
		d.client,
		"/v2/api/dictionaries",
		nil,
		dlpclient.PaginationZeroBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DLP Dictionaries", err.Error())
		return
	}

	var dictModels []models.DictionaryModel
	for i := range allDicts {
		m, diags := dictionaryDTOToModel(ctx, &allDicts[i], nil)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		dictModels = append(dictModels, m)
	}

	data := models.DictionaryListModel{
		Total: types.Int64Value(int64(len(dictModels))),
		Data:  dictModels,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ===========================================================================
// Helper mappers for Dictionary
// ===========================================================================

func dictionaryModelToDTO(ctx context.Context, m *models.DictionaryModel) (*dlpclient.DictionaryDTO, []string, diag.Diagnostics) {
	var diags diag.Diagnostics
	dto := &dlpclient.DictionaryDTO{
		Name:             m.Name.ValueString(),
		Category:         m.Category.ValueString(),
		OriginalFileName: m.OriginalFileName.ValueString(),
		RegionName:       m.RegionName.ValueString(),
	}

	if !m.Id.IsNull() && !m.Id.IsUnknown() && m.Id.ValueString() != "" {
		v := m.Id.ValueString()
		dto.ID = &v
	}
	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		v := m.Description.ValueString()
		dto.Description = &v
	}
	if !m.IsCaseSensitive.IsNull() && !m.IsCaseSensitive.IsUnknown() {
		v := m.IsCaseSensitive.ValueBool()
		dto.IsCaseSensitive = &v
	}

	var keywords []string
	if !m.KeywordList.IsNull() && !m.KeywordList.IsUnknown() {
		diags.Append(m.KeywordList.ElementsAs(ctx, &keywords, false)...)
		dto.Keywords = keywords
	}

	return dto, keywords, diags
}

func dictionaryDTOToModel(ctx context.Context, dto *dlpclient.DictionaryDTO, preferKeywords []string) (models.DictionaryModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var idStr string
	if dto.ID != nil {
		idStr = *dto.ID
	}

	m := models.DictionaryModel{
		Tfid:             types.StringValue(idStr),
		Id:               types.StringValue(idStr),
		Name:             types.StringValue(dto.Name),
		Category:         types.StringValue(dto.Category),
		OriginalFileName: types.StringValue(dto.OriginalFileName),
		RegionName:       types.StringValue(dto.RegionName),
		Description:      types.StringNull(),
		IsCaseSensitive:  types.BoolValue(false),
		KeywordList:      types.ListNull(types.StringType),
		KeywordCount:     types.Int64Null(),
	}

	if dto.Description != nil {
		m.Description = types.StringValue(*dto.Description)
	}
	if dto.IsCaseSensitive != nil {
		m.IsCaseSensitive = types.BoolValue(*dto.IsCaseSensitive)
	}
	if dto.KeywordCount != nil {
		m.KeywordCount = types.Int64Value(*dto.KeywordCount)
	}

	var kws []string
	if len(dto.Keywords) > 0 {
		kws = dto.Keywords
	} else if len(preferKeywords) > 0 {
		kws = preferKeywords
	}

	if len(kws) > 0 {
		listVal, d := types.ListValueFrom(ctx, types.StringType, kws)
		diags.Append(d...)
		m.KeywordList = listVal
		if m.KeywordCount.IsNull() {
			m.KeywordCount = types.Int64Value(int64(len(kws)))
		}
	} else if dto.Keywords != nil {
		m.KeywordList = types.ListValueMust(types.StringType, []attr.Value{})
	}

	return m, diags
}
