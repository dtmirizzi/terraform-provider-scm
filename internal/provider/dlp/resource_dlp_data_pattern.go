package provider

import (
	"context"
	"fmt"
	"strings"

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
// Resource: scm_dlp_data_pattern
// ===========================================================================

var (
	_ resource.Resource                = &DataPatternResource{}
	_ resource.ResourceWithConfigure   = &DataPatternResource{}
	_ resource.ResourceWithImportState = &DataPatternResource{}
)

func NewDataPatternResource() resource.Resource {
	return &DataPatternResource{}
}

type DataPatternResource struct {
	client *dlpclient.Client
}

func (r *DataPatternResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "scm_dlp_data_pattern"
}

func (r *DataPatternResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.DataPatternResourceSchema
}

func (r *DataPatternResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *DataPatternResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DataPatternModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto, diags := dataPatternModelToDTO(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if valErrs, err := dlpclient.ValidateDataPattern(dto); err != nil {
		for _, e := range valErrs {
			resp.Diagnostics.AddError("Data Pattern Validation Failed", e)
		}
		if len(valErrs) == 0 {
			resp.Diagnostics.AddError("Data Pattern Validation Failed", err.Error())
		}
		return
	}

	tflog.Debug(ctx, "Creating DLP Data Pattern", map[string]any{"name": dto.Name})
	var created dlpclient.DataPatternDTO
	if err := r.client.Post(ctx, "/v2/api/data-patterns", nil, dto, &created); err != nil {
		resp.Diagnostics.AddError("Error Creating DLP Data Pattern", err.Error())
		return
	}

	state, diags := dataPatternDTOToModel(ctx, &created, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DataPatternResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.DataPatternModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Data Pattern without an ID.")
		return
	}

	tflog.Debug(ctx, "Reading DLP Data Pattern", map[string]any{"id": id})
	var dto dlpclient.DataPatternDTO
	err := r.client.Get(ctx, fmt.Sprintf("/v2/api/data-patterns/%s", id), nil, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading DLP Data Pattern", err.Error())
		return
	}

	newState, diags := dataPatternDTOToModel(ctx, &dto, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *DataPatternResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DataPatternModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.Id.ValueString()
	if id == "" {
		id = plan.Tfid.ValueString()
	}

	dto, diags := dataPatternModelToDTO(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	dto.ID = &id

	if valErrs, err := dlpclient.ValidateDataPattern(dto); err != nil {
		for _, e := range valErrs {
			resp.Diagnostics.AddError("Data Pattern Validation Failed", e)
		}
		if len(valErrs) == 0 {
			resp.Diagnostics.AddError("Data Pattern Validation Failed", err.Error())
		}
		return
	}

	tflog.Debug(ctx, "Updating DLP Data Pattern", map[string]any{"id": id})
	var updated dlpclient.DataPatternDTO
	if err := r.client.Put(ctx, fmt.Sprintf("/v2/api/data-patterns/%s", id), nil, dto, &updated); err != nil {
		resp.Diagnostics.AddError("Error Updating DLP Data Pattern", err.Error())
		return
	}

	state, diags := dataPatternDTOToModel(ctx, &updated, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DataPatternResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.DataPatternModel
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

	tflog.Debug(ctx, "Deleting DLP Data Pattern", map[string]any{"id": id})
	if err := r.client.Delete(ctx, fmt.Sprintf("/v2/api/data-patterns/%s", id), nil); err != nil {
		if !dlpclient.IsNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting DLP Data Pattern", err.Error())
			return
		}
	}
}

func (r *DataPatternResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// DataSource: scm_dlp_data_pattern
// ===========================================================================

var (
	_ datasource.DataSource              = &DataPatternDataSource{}
	_ datasource.DataSourceWithConfigure = &DataPatternDataSource{}
)

func NewDataPatternDataSource() datasource.DataSource {
	return &DataPatternDataSource{}
}

type DataPatternDataSource struct {
	client *dlpclient.Client
}

func (d *DataPatternDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_data_pattern"
}

func (d *DataPatternDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DataPatternDataSourceSchema
}

func (d *DataPatternDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *DataPatternDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.DataPatternModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Attribute 'id' is required for scm_dlp_data_pattern data source.")
		return
	}

	var dto dlpclient.DataPatternDTO
	if err := d.client.Get(ctx, fmt.Sprintf("/v2/api/data-patterns/%s", id), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Error Reading DLP Data Pattern", err.Error())
		return
	}

	state, diags := dataPatternDTOToModel(ctx, &dto, nil)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ===========================================================================
// DataSource: scm_dlp_data_pattern_list
// ===========================================================================

var (
	_ datasource.DataSource              = &DataPatternListDataSource{}
	_ datasource.DataSourceWithConfigure = &DataPatternListDataSource{}
)

func NewDataPatternListDataSource() datasource.DataSource {
	return &DataPatternListDataSource{}
}

type DataPatternListDataSource struct {
	client *dlpclient.Client
}

func (d *DataPatternListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_data_pattern_list"
}

func (d *DataPatternListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DataPatternListDataSourceSchema
}

func (d *DataPatternListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *DataPatternListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.DataPatternListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := make(map[string]string)
	nameFilter := ""
	if !config.Name.IsNull() && config.Name.ValueString() != "" {
		nameFilter = config.Name.ValueString()
		params["name"] = nameFilter
	}

	allPatterns, err := dlpclient.GetAll[dlpclient.DataPatternDTO](
		ctx,
		d.client,
		"/v2/api/data-patterns",
		params,
		dlpclient.PaginationZeroBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DLP Data Patterns", err.Error())
		return
	}

	var patternModels []models.DataPatternModel
	for i := range allPatterns {
		p := &allPatterns[i]
		if nameFilter != "" && !strings.EqualFold(p.Name, nameFilter) {
			continue
		}
		m, diags := dataPatternDTOToModel(ctx, p, nil)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		patternModels = append(patternModels, m)
	}

	data := models.DataPatternListModel{
		Name:  config.Name,
		Total: types.Int64Value(int64(len(patternModels))),
		Data:  patternModels,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ===========================================================================
// Helper mappers for Data Pattern
// ===========================================================================

func dataPatternModelToDTO(ctx context.Context, m *models.DataPatternModel) (*dlpclient.DataPatternDTO, diag.Diagnostics) {
	var diags diag.Diagnostics
	dto := &dlpclient.DataPatternDTO{
		Name: m.Name.ValueString(),
		Type: m.Type.ValueString(),
	}

	if !m.Id.IsNull() && !m.Id.IsUnknown() && m.Id.ValueString() != "" {
		v := m.Id.ValueString()
		dto.ID = &v
	}
	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		v := m.Description.ValueString()
		dto.Description = &v
	}

	if m.DetectionConfig != nil {
		var levels []string
		if !m.DetectionConfig.SupportedConfidenceLevels.IsNull() && !m.DetectionConfig.SupportedConfidenceLevels.IsUnknown() {
			diags.Append(m.DetectionConfig.SupportedConfidenceLevels.ElementsAs(ctx, &levels, false)...)
		}
		dto.DetectionConfig = &dlpclient.DataPatternDetectionConfig{
			Technique:                 m.DetectionConfig.Technique.ValueString(),
			SupportedConfidenceLevels: levels,
		}
	}

	if m.MatchingRules != nil {
		mr := &dlpclient.DataPatternMatchingRules{}
		if !m.MatchingRules.ProximityDistance.IsNull() && !m.MatchingRules.ProximityDistance.IsUnknown() {
			v := m.MatchingRules.ProximityDistance.ValueInt64()
			mr.ProximityDistance = &v
		}
		if !m.MatchingRules.ProximityKeywords.IsNull() && !m.MatchingRules.ProximityKeywords.IsUnknown() {
			var kws []string
			diags.Append(m.MatchingRules.ProximityKeywords.ElementsAs(ctx, &kws, false)...)
			mr.ProximityKeywords = kws
		}
		if len(m.MatchingRules.Regexes) > 0 {
			var rxRules []dlpclient.DataPatternRegexRule
			for _, r := range m.MatchingRules.Regexes {
				rxRules = append(rxRules, dlpclient.DataPatternRegexRule{
					Regex:  r.Regex.ValueString(),
					Weight: r.Weight.ValueFloat64(),
				})
			}
			mr.Regexes = rxRules
		}
		dto.MatchingRules = mr
	}

	return dto, diags
}

func dataPatternDTOToModel(ctx context.Context, dto *dlpclient.DataPatternDTO, prior *models.DataPatternModel) (models.DataPatternModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var idStr string
	if dto.ID != nil {
		idStr = *dto.ID
	}

	m := models.DataPatternModel{
		Tfid:        types.StringValue(idStr),
		Id:          types.StringValue(idStr),
		Name:        types.StringValue(dto.Name),
		Type:        types.StringValue(dto.Type),
		Description: types.StringNull(),
		Status:      types.StringNull(),
		Version:     types.Int64Null(),
	}

	if dto.Description != nil {
		m.Description = types.StringValue(*dto.Description)
	}
	if dto.Status != nil {
		m.Status = types.StringValue(*dto.Status)
	}
	if dto.Version != nil {
		m.Version = types.Int64Value(*dto.Version)
	}

	if dto.DetectionConfig != nil {
		levelsVal, d := types.ListValueFrom(ctx, types.StringType, dto.DetectionConfig.SupportedConfidenceLevels)
		diags.Append(d...)
		if prior != nil && prior.DetectionConfig != nil && !prior.DetectionConfig.SupportedConfidenceLevels.IsNull() && !prior.DetectionConfig.SupportedConfidenceLevels.IsUnknown() {
			levelsVal = prior.DetectionConfig.SupportedConfidenceLevels
		}
		m.DetectionConfig = &models.DataPatternDetectionConfigModel{
			Technique:                 types.StringValue(dto.DetectionConfig.Technique),
			SupportedConfidenceLevels: levelsVal,
		}
	}

	if dto.MatchingRules != nil {
		mr := &models.DataPatternMatchingRulesModel{
			ProximityDistance: types.Int64Null(),
			ProximityKeywords: types.ListNull(types.StringType),
		}
		if dto.MatchingRules.ProximityDistance != nil {
			mr.ProximityDistance = types.Int64Value(*dto.MatchingRules.ProximityDistance)
		}
		if dto.MatchingRules.ProximityKeywords != nil {
			kwVal, d := types.ListValueFrom(ctx, types.StringType, dto.MatchingRules.ProximityKeywords)
			diags.Append(d...)
			mr.ProximityKeywords = kwVal
		}
		if len(dto.MatchingRules.Regexes) > 0 {
			var regexModels []models.DataPatternRegexRuleModel
			for _, r := range dto.MatchingRules.Regexes {
				regexModels = append(regexModels, models.DataPatternRegexRuleModel{
					Regex:  types.StringValue(r.Regex),
					Weight: types.Float64Value(r.Weight),
				})
			}
			mr.Regexes = regexModels
		}
		m.MatchingRules = mr
	}

	return m, diags
}
