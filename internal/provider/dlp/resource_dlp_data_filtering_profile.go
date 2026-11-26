package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
	"github.com/paloaltonetworks/terraform-provider-scm/internal/provider/dlp/client"
)

// Ensure provider interfaces are implemented.
var (
	_ resource.Resource                  = &dataFilteringProfileResource{}
	_ resource.ResourceWithConfigure     = &dataFilteringProfileResource{}
	_ resource.ResourceWithImportState   = &dataFilteringProfileResource{}
	_ datasource.DataSource              = &dataFilteringProfileDataSource{}
	_ datasource.DataSourceWithConfigure = &dataFilteringProfileDataSource{}
	_ datasource.DataSource              = &dataFilteringProfileListDataSource{}
	_ datasource.DataSourceWithConfigure = &dataFilteringProfileListDataSource{}
)

func NewDataFilteringProfileResource() resource.Resource {
	return &dataFilteringProfileResource{}
}

func NewDataFilteringProfileDataSource() datasource.DataSource {
	return &dataFilteringProfileDataSource{}
}

func NewDataFilteringProfileListDataSource() datasource.DataSource {
	return &dataFilteringProfileListDataSource{}
}

// ===========================================================================
// Resource: scm_dlp_data_filtering_profile
// ===========================================================================

type dataFilteringProfileResource struct {
	client *dlpclient.Client
}

func (r *dataFilteringProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_data_filtering_profile"
}

func (r *dataFilteringProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.DataFilteringProfileResourceSchema
}

func (r *dataFilteringProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client := extractDLPClient(req.ProviderData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

func (r *dataFilteringProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DataFilteringProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := dataFilteringProfileModelToDTO(ctx, &plan)

	// Resolve ID: if omitted, resolve from list by data_profile_id.
	// Only allow fallback if id is explicitly set and non-empty/known.
	targetID := plan.Id.ValueString()
	if targetID == "" {
		dataProfileID := plan.DataProfileId.ValueInt64()
		existingItems, err := dlpclient.GetAll[dlpclient.DataFilteringProfileDTO](
			ctx,
			r.client,
			"/v2/api/data-filtering-profiles",
			nil,
			dlpclient.PaginationZeroBased,
			100,
			10,
		)
		if err != nil {
			resp.Diagnostics.AddError("Failed to list DLP data filtering profiles to resolve ID", err.Error())
			return
		}
		for _, item := range existingItems {
			if item.DataProfileID != nil && *item.DataProfileID == dataProfileID && item.ID != nil && *item.ID != "" {
				targetID = *item.ID
				break
			}
		}
		if targetID == "" {
			resp.Diagnostics.AddError(
				"Unable to resolve Data Filtering Profile ID",
				fmt.Sprintf("No existing data filtering profile found associated with data_profile_id %d. Ensure the parent data profile exists and has generated a filtering profile, or explicitly specify the 'id' attribute.", dataProfileID),
			)
			return
		}
	}

	dto.ID = &targetID
	tflog.Debug(ctx, "Creating/Configuring DLP Data Filtering Profile", map[string]interface{}{"id": targetID, "data_profile_id": plan.DataProfileId.ValueInt64()})

	var updated dlpclient.DataFilteringProfileDTO
	err := r.client.Put(ctx, fmt.Sprintf("/v2/api/data-filtering-profiles/%s", dlpclient.EscapeID(targetID)), nil, dto, &updated)
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure DLP data filtering profile", err.Error())
		return
	}

	state := dataFilteringProfileDTOToModel(ctx, &updated)
	if state.Id.IsNull() || state.Id.ValueString() == "" {
		state.Id = types.StringValue(targetID)
		state.Tfid = types.StringValue(targetID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dataFilteringProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.DataFilteringProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read data filtering profile without an ID")
		return
	}

	var dto dlpclient.DataFilteringProfileDTO
	err := r.client.Get(ctx, fmt.Sprintf("/v2/api/data-filtering-profiles/%s", dlpclient.EscapeID(id)), nil, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read DLP data filtering profile", err.Error())
		return
	}

	updatedState := dataFilteringProfileDTOToModel(ctx, &dto)
	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedState)...)
}

func (r *dataFilteringProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DataFilteringProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.Id.ValueString()
	if id == "" {
		id = plan.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot update data filtering profile without an ID")
		return
	}

	dto := dataFilteringProfileModelToDTO(ctx, &plan)
	dto.ID = &id

	tflog.Debug(ctx, "Updating DLP Data Filtering Profile", map[string]interface{}{"id": id})

	var updated dlpclient.DataFilteringProfileDTO
	err := r.client.Put(ctx, fmt.Sprintf("/v2/api/data-filtering-profiles/%s", dlpclient.EscapeID(id)), nil, dto, &updated)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update DLP data filtering profile", err.Error())
		return
	}

	state := dataFilteringProfileDTOToModel(ctx, &updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dataFilteringProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Data filtering profiles are automatically generated and destroyed with parent data profiles.
	// In DLP API, DELETE /v2/api/data-filtering-profiles/{id} is not supported or is a no-op reset.
	// Removing from Terraform state is sufficient.
	tflog.Debug(ctx, "Removing DLP Data Filtering Profile from Terraform management")
}

func (r *dataFilteringProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// Data Source: scm_dlp_data_filtering_profile
// ===========================================================================

type dataFilteringProfileDataSource struct {
	client *dlpclient.Client
}

func (d *dataFilteringProfileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_data_filtering_profile"
}

func (d *dataFilteringProfileDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DataFilteringProfileDataSourceSchema
}

func (d *dataFilteringProfileDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client := extractDLPClient(req.ProviderData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

func (d *dataFilteringProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state models.DataFilteringProfileResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "ID is required for scm_dlp_data_filtering_profile data source")
		return
	}

	var dto dlpclient.DataFilteringProfileDTO
	if err := d.client.Get(ctx, fmt.Sprintf("/v2/api/data-filtering-profiles/%s", dlpclient.EscapeID(id)), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Failed to fetch DLP data filtering profile", err.Error())
		return
	}

	model := dataFilteringProfileDTOToModel(ctx, &dto)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// ===========================================================================
// List Data Source: scm_dlp_data_filtering_profile_list
// ===========================================================================

type dataFilteringProfileListDataSource struct {
	client *dlpclient.Client
}

func (d *dataFilteringProfileListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_data_filtering_profile_list"
}

func (d *dataFilteringProfileListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DataFilteringProfileListDataSourceSchema
}

func (d *dataFilteringProfileListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client := extractDLPClient(req.ProviderData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

func (d *dataFilteringProfileListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.DataFilteringProfileListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := make(map[string]string)
	if !config.Name.IsNull() && config.Name.ValueString() != "" {
		params["name"] = config.Name.ValueString()
	}
	if !config.Status.IsNull() && config.Status.ValueString() != "" {
		params["status"] = config.Status.ValueString()
	}

	items, err := dlpclient.GetAll[dlpclient.DataFilteringProfileDTO](
		ctx,
		d.client,
		"/v2/api/data-filtering-profiles",
		params,
		dlpclient.PaginationZeroBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list DLP data filtering profiles", err.Error())
		return
	}

	var dataList []models.DataFilteringProfileResourceModel
	for _, item := range items {
		if !config.Name.IsNull() && config.Name.ValueString() != "" {
			if item.Name == nil || !strings.EqualFold(*item.Name, config.Name.ValueString()) {
				continue
			}
		}
		m := dataFilteringProfileDTOToModel(ctx, &item)
		dataList = append(dataList, m)
	}

	config.Tfid = types.StringValue("dlp_data_filtering_profiles")
	config.Total = types.Int64Value(int64(len(dataList)))
	config.Data = dataList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// ===========================================================================
// Mappers and Helpers: DataFilteringProfile
// ===========================================================================

func dataFilteringProfileModelToDTO(ctx context.Context, m *models.DataFilteringProfileResourceModel) *dlpclient.DataFilteringProfileDTO {
	dto := &dlpclient.DataFilteringProfileDTO{
		FileBased:    m.FileBased.ValueBool(),
		NonFileBased: m.NonFileBased.ValueBool(),
	}

	if !m.Id.IsNull() && !m.Id.IsUnknown() && m.Id.ValueString() != "" {
		id := m.Id.ValueString()
		dto.ID = &id
	}

	if !m.DataProfileId.IsNull() && !m.DataProfileId.IsUnknown() {
		dpid := m.DataProfileId.ValueInt64()
		dto.DataProfileID = &dpid
	}

	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		name := m.Name.ValueString()
		dto.Name = &name
	}

	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		desc := m.Description.ValueString()
		dto.Description = &desc
	}

	if !m.Direction.IsNull() && !m.Direction.IsUnknown() {
		dir := m.Direction.ValueString()
		dto.Direction = &dir
	}

	if !m.ScanType.IsNull() && !m.ScanType.IsUnknown() {
		st := m.ScanType.ValueString()
		dto.ScanType = &st
	}

	if !m.LogSeverity.IsNull() && !m.LogSeverity.IsUnknown() {
		ls := m.LogSeverity.ValueString()
		dto.LogSeverity = &ls
	}

	if len(m.FileType) > 0 {
		var ftList []string
		for _, ft := range m.FileType {
			if !ft.IsNull() && !ft.IsUnknown() {
				ftList = append(ftList, ft.ValueString())
			}
		}
		dto.FileType = ftList
	}

	if !m.IsEndUserCoachingEnabled.IsNull() && !m.IsEndUserCoachingEnabled.IsUnknown() {
		euc := m.IsEndUserCoachingEnabled.ValueBool()
		dto.IsEndUserCoachingEnabled = &euc
	}

	if !m.EucTemplateId.IsNull() && !m.EucTemplateId.IsUnknown() {
		tmpl := m.EucTemplateId.ValueString()
		dto.EucTemplateID = &tmpl
	}

	if !m.Rule1Action.IsNull() && !m.Rule1Action.IsUnknown() {
		a := m.Rule1Action.ValueString()
		dto.Rule1 = &dlpclient.DataFilteringRuleDTO{Action: &a}
	}

	if !m.Rule2Action.IsNull() && !m.Rule2Action.IsUnknown() {
		a := m.Rule2Action.ValueString()
		dto.Rule2 = &dlpclient.DataFilteringRuleDTO{Action: &a}
	}

	if len(m.AppExclusionList) > 0 || len(m.UrlExclusionList) > 0 {
		exclusions := &dlpclient.ExclusionsDTO{}
		for _, app := range m.AppExclusionList {
			item := dlpclient.AppExclusionItemDTO{AppID: app.AppId.ValueString()}
			if !app.AppName.IsNull() && !app.AppName.IsUnknown() {
				name := app.AppName.ValueString()
				item.AppName = &name
			}
			if !app.Type.IsNull() && !app.Type.IsUnknown() {
				t := app.Type.ValueString()
				item.Type = &t
			}
			exclusions.AppExclusionList = append(exclusions.AppExclusionList, item)
		}
		for _, u := range m.UrlExclusionList {
			item := dlpclient.URLExclusionItemDTO{URLID: u.UrlId.ValueString()}
			if !u.UrlName.IsNull() && !u.UrlName.IsUnknown() {
				name := u.UrlName.ValueString()
				item.URLName = &name
			}
			if !u.Type.IsNull() && !u.Type.IsUnknown() {
				t := u.Type.ValueString()
				item.Type = &t
			}
			exclusions.URLExclusionList = append(exclusions.URLExclusionList, item)
		}
		dto.Exclusions = exclusions
	}

	for _, er := range m.ExceptionRules {
		item := dlpclient.ExceptionRuleDTO{}
		if !er.Name.IsNull() && !er.Name.IsUnknown() {
			n := er.Name.ValueString()
			item.Name = &n
		}
		if !er.Action.IsNull() && !er.Action.IsUnknown() {
			a := er.Action.ValueString()
			item.Action = &a
		}
		if !er.LogSeverity.IsNull() && !er.LogSeverity.IsUnknown() {
			ls := er.LogSeverity.ValueString()
			item.LogSeverity = &ls
		}
		dto.ExceptionRules = append(dto.ExceptionRules, item)
	}

	return dto
}

func dataFilteringProfileDTOToModel(ctx context.Context, dto *dlpclient.DataFilteringProfileDTO) models.DataFilteringProfileResourceModel {
	var m models.DataFilteringProfileResourceModel

	idStr := ""
	if dto.ID != nil {
		idStr = *dto.ID
	}
	m.Id = types.StringValue(idStr)
	m.Tfid = types.StringValue(idStr)

	if dto.DataProfileID != nil {
		m.DataProfileId = types.Int64Value(*dto.DataProfileID)
	} else if idStr != "" {
		if pid, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			m.DataProfileId = types.Int64Value(pid)
		}
	}

	if dto.Name != nil {
		m.Name = types.StringValue(*dto.Name)
	} else {
		m.Name = types.StringNull()
	}

	if dto.Description != nil {
		m.Description = types.StringValue(*dto.Description)
	} else {
		m.Description = types.StringNull()
	}

	if dto.Direction != nil {
		m.Direction = types.StringValue(*dto.Direction)
	} else {
		m.Direction = types.StringValue("both")
	}

	if dto.ScanType != nil {
		m.ScanType = types.StringValue(*dto.ScanType)
	} else {
		m.ScanType = types.StringValue("include")
	}

	if dto.LogSeverity != nil {
		m.LogSeverity = types.StringValue(*dto.LogSeverity)
	} else {
		m.LogSeverity = types.StringValue("HIGH")
	}

	m.FileBased = types.BoolValue(dto.FileBased)
	m.NonFileBased = types.BoolValue(dto.NonFileBased)

	for _, ft := range dto.FileType {
		m.FileType = append(m.FileType, types.StringValue(ft))
	}

	if dto.IsEndUserCoachingEnabled != nil {
		m.IsEndUserCoachingEnabled = types.BoolValue(*dto.IsEndUserCoachingEnabled)
	} else {
		m.IsEndUserCoachingEnabled = types.BoolValue(false)
	}

	if dto.EucTemplateID != nil {
		m.EucTemplateId = types.StringValue(*dto.EucTemplateID)
	} else {
		m.EucTemplateId = types.StringNull()
	}

	if dto.Rule1 != nil && dto.Rule1.Action != nil {
		m.Rule1Action = types.StringValue(*dto.Rule1.Action)
	} else {
		m.Rule1Action = types.StringNull()
	}

	if dto.Rule2 != nil && dto.Rule2.Action != nil {
		m.Rule2Action = types.StringValue(*dto.Rule2.Action)
	} else {
		m.Rule2Action = types.StringNull()
	}

	if dto.Exclusions != nil {
		for _, app := range dto.Exclusions.AppExclusionList {
			appMod := models.AppExclusionModel{
				AppId: types.StringValue(app.AppID),
			}
			if app.AppName != nil {
				appMod.AppName = types.StringValue(*app.AppName)
			} else {
				appMod.AppName = types.StringNull()
			}
			if app.Type != nil {
				appMod.Type = types.StringValue(*app.Type)
			} else {
				appMod.Type = types.StringNull()
			}
			m.AppExclusionList = append(m.AppExclusionList, appMod)
		}

		for _, u := range dto.Exclusions.URLExclusionList {
			urlMod := models.URLExclusionModel{
				UrlId: types.StringValue(u.URLID),
			}
			if u.URLName != nil {
				urlMod.UrlName = types.StringValue(*u.URLName)
			} else {
				urlMod.UrlName = types.StringNull()
			}
			if u.Type != nil {
				urlMod.Type = types.StringValue(*u.Type)
			} else {
				urlMod.Type = types.StringNull()
			}
			m.UrlExclusionList = append(m.UrlExclusionList, urlMod)
		}
	}

	for _, er := range dto.ExceptionRules {
		erMod := models.ExceptionRuleModel{}
		if er.Name != nil {
			erMod.Name = types.StringValue(*er.Name)
		} else {
			erMod.Name = types.StringNull()
		}
		if er.Action != nil {
			erMod.Action = types.StringValue(*er.Action)
		} else {
			erMod.Action = types.StringNull()
		}
		if er.LogSeverity != nil {
			erMod.LogSeverity = types.StringValue(*er.LogSeverity)
		} else {
			erMod.LogSeverity = types.StringNull()
		}
		m.ExceptionRules = append(m.ExceptionRules, erMod)
	}

	return m
}
