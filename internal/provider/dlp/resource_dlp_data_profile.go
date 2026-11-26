package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/dlpclient"
	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
)

// Ensure provider interfaces are implemented.
var (
	_ resource.Resource                  = &dataProfileResource{}
	_ resource.ResourceWithConfigure     = &dataProfileResource{}
	_ resource.ResourceWithImportState   = &dataProfileResource{}
	_ datasource.DataSource              = &dataProfileDataSource{}
	_ datasource.DataSourceWithConfigure = &dataProfileDataSource{}
	_ datasource.DataSource              = &dataProfileListDataSource{}
	_ datasource.DataSourceWithConfigure = &dataProfileListDataSource{}
)

func NewDataProfileResource() resource.Resource {
	return &dataProfileResource{}
}

func NewDataProfileDataSource() datasource.DataSource {
	return &dataProfileDataSource{}
}

func NewDataProfileListDataSource() datasource.DataSource {
	return &dataProfileListDataSource{}
}

// ===========================================================================
// Resource: scm_dlp_data_profile
// ===========================================================================

type dataProfileResource struct {
	client *dlpclient.Client
}

func (r *dataProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_data_profile"
}

func (r *dataProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.DataProfileResourceSchema
}

func (r *dataProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client := extractDLPClient(req.ProviderData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	r.client = client
}

func (r *dataProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DataProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto, err := dataProfileModelToDTO(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping data profile plan to DTO", err.Error())
		return
	}

	// Validate and normalize backend defaults
	if _, err := dlpclient.NormalizeAndValidateDataProfile(dto); err != nil {
		resp.Diagnostics.AddError("Data profile validation failed", err.Error())
		return
	}

	tflog.Debug(ctx, "Creating DLP Data Profile", map[string]interface{}{"name": dto.Name})

	var created dlpclient.DataProfileDTO
	if err := r.client.Post(ctx, "/v2/api/data-profiles", nil, dto, &created); err != nil {
		resp.Diagnostics.AddError("Failed to create DLP data profile", err.Error())
		return
	}

	if created.ID == nil || *created.ID == "" {
		resp.Diagnostics.AddError("Failed to create DLP data profile", "Backend returned empty ID")
		return
	}

	state, err := dataProfileDTOToModel(ctx, &created)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping created data profile to state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dataProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.DataProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read data profile without an ID")
		return
	}

	var dto dlpclient.DataProfileDTO
	err := r.client.Get(ctx, fmt.Sprintf("/v2/api/data-profiles/%s", id), nil, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read DLP data profile", err.Error())
		return
	}

	updatedState, err := dataProfileDTOToModel(ctx, &dto)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping data profile DTO to state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedState)...)
}

func (r *dataProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DataProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.Id.ValueString()
	if id == "" {
		id = plan.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot update data profile without an ID")
		return
	}

	dto, err := dataProfileModelToDTO(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping data profile plan to DTO", err.Error())
		return
	}

	dto.ID = &id
	if !plan.Version.IsNull() && !plan.Version.IsUnknown() {
		v := plan.Version.ValueInt64()
		dto.Version = &v
	}

	if _, err := dlpclient.NormalizeAndValidateDataProfile(dto); err != nil {
		resp.Diagnostics.AddError("Data profile validation failed", err.Error())
		return
	}

	tflog.Debug(ctx, "Updating DLP Data Profile", map[string]interface{}{"id": id, "name": dto.Name})

	var updated dlpclient.DataProfileDTO
	if err := r.client.Put(ctx, fmt.Sprintf("/v2/api/data-profiles/%s", id), nil, dto, &updated); err != nil {
		resp.Diagnostics.AddError("Failed to update DLP data profile", err.Error())
		return
	}

	state, err := dataProfileDTOToModel(ctx, &updated)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping updated data profile to state", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func strPtr(s string) *string { return &s }

func (r *dataProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.DataProfileResourceModel
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

	tflog.Debug(ctx, "Deleting DLP Data Profile", map[string]interface{}{"id": id})

	// Disable the profile before deletion so child data patterns/dictionaries can be cleanly deleted afterward.
	_ = r.client.Patch(ctx, "/v1/api/data-profile/update-data-profiles-status", nil, []map[string]any{
		{"id": id, "profile_status": "disabled"},
	}, nil)

	err := r.client.Delete(ctx, fmt.Sprintf("/v2/api/data-profiles/%s", id), nil)
	if err != nil && !dlpclient.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete DLP data profile", err.Error())
		return
	}

	// If the backend skipped deleting a schemaVersion=2 profile when service-name != dlp,
	// rename it and detach custom child patterns so both the profile name and child patterns can be reused/deleted.
	var check dlpclient.DataProfileDTO
	if getErr := r.client.Get(ctx, fmt.Sprintf("/v2/api/data-profiles/%s", id), nil, &check); getErr == nil {
		deletedName := fmt.Sprintf("deleted-%s-%d", id, time.Now().Unix())
		if len(deletedName) > 60 {
			deletedName = deletedName[:60]
		}
		check.Name = deletedName
		disabled := "disabled"
		check.ProfileStatus = &disabled
		check.DetectionRules = []dlpclient.DetectionRuleDTO{
			{
				RuleType: "expression_tree",
				ExpressionTree: &dlpclient.ExpressionTreeNodeDTO{
					OperatorType: strPtr("or"),
					SubExpressions: []dlpclient.ExpressionTreeNodeDTO{
						{
							RuleItem: &dlpclient.RuleItemDTO{
								ID:                     "5cd607d2f2441800083c542e",
								Name:                   strPtr("Credit Card Numbers"),
								DetectionTechnique:     "weighted_regex",
								MatchType:              strPtr("include"),
								OccurrenceOperatorType: "any",
								ConfidenceLevel:        strPtr("high"),
							},
						},
					},
				},
			},
		}
		_ = r.client.Put(ctx, fmt.Sprintf("/v2/api/data-profiles/%s", id), nil, check, nil)
	}
}

func (r *dataProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// Data Source: scm_dlp_data_profile
// ===========================================================================

type dataProfileDataSource struct {
	client *dlpclient.Client
}

func (d *dataProfileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_data_profile"
}

func (d *dataProfileDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DataProfileDataSourceSchema
}

func (d *dataProfileDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client := extractDLPClient(req.ProviderData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

func (d *dataProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state models.DataProfileResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "ID is required for scm_dlp_data_profile data source")
		return
	}

	var dto dlpclient.DataProfileDTO
	if err := d.client.Get(ctx, fmt.Sprintf("/v2/api/data-profiles/%s", id), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Failed to fetch DLP data profile", err.Error())
		return
	}

	model, err := dataProfileDTOToModel(ctx, &dto)
	if err != nil {
		resp.Diagnostics.AddError("Error mapping data profile DTO to model", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

// ===========================================================================
// List Data Source: scm_dlp_data_profile_list
// ===========================================================================

type dataProfileListDataSource struct {
	client *dlpclient.Client
}

func (d *dataProfileListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_data_profile_list"
}

func (d *dataProfileListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DataProfileListDataSourceSchema
}

func (d *dataProfileListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client := extractDLPClient(req.ProviderData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = client
}

func (d *dataProfileListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.DataProfileListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := make(map[string]string)
	if !config.Name.IsNull() && config.Name.ValueString() != "" {
		params["name"] = config.Name.ValueString()
	}

	items, err := dlpclient.GetAll[dlpclient.DataProfileDTO](
		ctx,
		d.client,
		"/v2/api/data-profiles",
		params,
		dlpclient.PaginationZeroBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list DLP data profiles", err.Error())
		return
	}

	var dataList []models.DataProfileResourceModel
	for _, item := range items {
		// If name filter requested and API didn't filter server-side, filter client-side too
		if !config.Name.IsNull() && config.Name.ValueString() != "" {
			if !strings.EqualFold(item.Name, config.Name.ValueString()) {
				continue
			}
		}
		m, err := dataProfileDTOToModel(ctx, &item)
		if err != nil {
			resp.Diagnostics.AddError("Error mapping data profile item to model", err.Error())
			return
		}
		dataList = append(dataList, m)
	}

	config.Tfid = types.StringValue("dlp_data_profiles")
	config.Total = types.Int64Value(int64(len(dataList)))
	config.Data = dataList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// ===========================================================================
// Mappers and Helpers: DataProfile
// ===========================================================================

func dataProfileModelToDTO(ctx context.Context, m *models.DataProfileResourceModel) (*dlpclient.DataProfileDTO, error) {
	dto := &dlpclient.DataProfileDTO{
		Name: m.Name.ValueString(),
	}

	if !m.Id.IsNull() && !m.Id.IsUnknown() && m.Id.ValueString() != "" {
		id := m.Id.ValueString()
		dto.ID = &id
	}
	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		desc := m.Description.ValueString()
		dto.Description = &desc
	}
	if !m.ProfileType.IsNull() && !m.ProfileType.IsUnknown() {
		pt := m.ProfileType.ValueString()
		dto.ProfileType = &pt
	}
	if !m.ProfileStatus.IsNull() && !m.ProfileStatus.IsUnknown() {
		ps := m.ProfileStatus.ValueString()
		dto.ProfileStatus = &ps
	}
	if !m.IsGranularDataProfile.IsNull() && !m.IsGranularDataProfile.IsUnknown() {
		ig := m.IsGranularDataProfile.ValueBool()
		dto.IsGranularDataProfile = &ig
	}
	if !m.Version.IsNull() && !m.Version.IsUnknown() {
		v := m.Version.ValueInt64()
		dto.Version = &v
	}

	for _, ruleModel := range m.DetectionRules {
		ruleDTO := dlpclient.DetectionRuleDTO{
			RuleType: ruleModel.RuleType.ValueString(),
		}

		switch ruleDTO.RuleType {
		case "multi_profile":
			if ruleModel.MultiProfile != nil {
				mpDTO := &dlpclient.MultiProfileNodeDTO{
					OperatorType: ruleModel.MultiProfile.OperatorType.ValueString(),
				}
				for _, idVal := range ruleModel.MultiProfile.DataProfileIds {
					mpDTO.DataProfileIDs = append(mpDTO.DataProfileIDs, idVal.ValueInt64())
				}
				ruleDTO.MultiProfile = mpDTO
			}

		case "expression_tree":
			if ruleModel.ExpressionTree != nil {
				treeDTO, err := expressionTreeModelToDTO(ruleModel.ExpressionTree)
				if err != nil {
					return nil, err
				}
				ruleDTO.ExpressionTree = treeDTO
			}
		}

		dto.DetectionRules = append(dto.DetectionRules, ruleDTO)
	}

	return dto, nil
}

func expressionTreeModelToDTO(m *models.DataProfileExpressionTreeModel) (*dlpclient.ExpressionTreeNodeDTO, error) {
	// If raw expression_tree_json is provided, deserialize it directly
	if !m.ExpressionTreeJson.IsNull() && !m.ExpressionTreeJson.IsUnknown() && m.ExpressionTreeJson.ValueString() != "" {
		var node dlpclient.ExpressionTreeNodeDTO
		if err := json.Unmarshal([]byte(m.ExpressionTreeJson.ValueString()), &node); err != nil {
			return nil, fmt.Errorf("invalid expression_tree_json: %w", err)
		}
		return &node, nil
	}

	node := &dlpclient.ExpressionTreeNodeDTO{}

	if !m.OperatorType.IsNull() && !m.OperatorType.IsUnknown() && m.OperatorType.ValueString() != "" {
		op := m.OperatorType.ValueString()
		node.OperatorType = &op
	}

	if m.RuleItem != nil && !m.RuleItem.Id.IsNull() && m.RuleItem.Id.ValueString() != "" {
		node.RuleItem = ruleItemModelToDTO(m.RuleItem)
	}

	for _, sub := range m.SubExpressions {
		subNode := dlpclient.ExpressionTreeNodeDTO{}
		if !sub.OperatorType.IsNull() && !sub.OperatorType.IsUnknown() && sub.OperatorType.ValueString() != "" {
			op := sub.OperatorType.ValueString()
			subNode.OperatorType = &op
		}
		if sub.RuleItem != nil && !sub.RuleItem.Id.IsNull() && sub.RuleItem.Id.ValueString() != "" {
			subNode.RuleItem = ruleItemModelToDTO(sub.RuleItem)
		}
		for _, sub2 := range sub.SubExpressions {
			sub2Node := dlpclient.ExpressionTreeNodeDTO{}
			if !sub2.OperatorType.IsNull() && !sub2.OperatorType.IsUnknown() && sub2.OperatorType.ValueString() != "" {
				op := sub2.OperatorType.ValueString()
				sub2Node.OperatorType = &op
			}
			if sub2.RuleItem != nil && !sub2.RuleItem.Id.IsNull() && sub2.RuleItem.Id.ValueString() != "" {
				sub2Node.RuleItem = ruleItemModelToDTO(sub2.RuleItem)
			}
			subNode.SubExpressions = append(subNode.SubExpressions, sub2Node)
		}
		node.SubExpressions = append(node.SubExpressions, subNode)
	}

	return node, nil
}

func ruleItemModelToDTO(m *models.DataProfileRuleItemModel) *dlpclient.RuleItemDTO {
	item := &dlpclient.RuleItemDTO{
		ID:                     m.Id.ValueString(),
		DetectionTechnique:     m.DetectionTechnique.ValueString(),
		OccurrenceOperatorType: m.OccurrenceOperatorType.ValueString(),
	}
	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		name := m.Name.ValueString()
		item.Name = &name
	}
	if !m.OccurrenceCount.IsNull() && !m.OccurrenceCount.IsUnknown() {
		count := m.OccurrenceCount.ValueInt64()
		item.OccurrenceCount = &count
	}
	if !m.OccurrenceLow.IsNull() && !m.OccurrenceLow.IsUnknown() {
		low := m.OccurrenceLow.ValueInt64()
		item.OccurrenceLow = &low
	}
	if !m.OccurrenceHigh.IsNull() && !m.OccurrenceHigh.IsUnknown() {
		high := m.OccurrenceHigh.ValueInt64()
		item.OccurrenceHigh = &high
	}
	if !m.ConfidenceLevel.IsNull() && !m.ConfidenceLevel.IsUnknown() {
		cl := m.ConfidenceLevel.ValueString()
		item.ConfidenceLevel = &cl
	} else {
		cl := "high"
		item.ConfidenceLevel = &cl
	}
	if !m.MatchType.IsNull() && !m.MatchType.IsUnknown() {
		mt := m.MatchType.ValueString()
		item.MatchType = &mt
	} else {
		mt := "include"
		item.MatchType = &mt
	}
	if !m.ByUniqueCount.IsNull() && !m.ByUniqueCount.IsUnknown() {
		buc := m.ByUniqueCount.ValueBool()
		item.ByUniqueCount = &buc
	} else {
		buc := false
		item.ByUniqueCount = &buc
	}

	return item
}

func dataProfileDTOToModel(ctx context.Context, dto *dlpclient.DataProfileDTO) (models.DataProfileResourceModel, error) {
	var m models.DataProfileResourceModel

	idStr := ""
	if dto.ID != nil {
		idStr = *dto.ID
	}
	m.Id = types.StringValue(idStr)
	m.Tfid = types.StringValue(idStr)
	m.Name = types.StringValue(dto.Name)

	if dto.Description != nil {
		m.Description = types.StringValue(*dto.Description)
	} else {
		m.Description = types.StringNull()
	}

	if dto.ProfileType != nil {
		m.ProfileType = types.StringValue(*dto.ProfileType)
	} else {
		m.ProfileType = types.StringValue("advanced")
	}

	if dto.ProfileStatus != nil {
		m.ProfileStatus = types.StringValue(*dto.ProfileStatus)
	} else {
		m.ProfileStatus = types.StringValue("active")
	}

	if dto.IsGranularDataProfile != nil {
		m.IsGranularDataProfile = types.BoolValue(*dto.IsGranularDataProfile)
	} else {
		m.IsGranularDataProfile = types.BoolValue(false)
	}

	if dto.Version != nil {
		m.Version = types.Int64Value(*dto.Version)
	} else {
		m.Version = types.Int64Value(1)
	}

	for _, ruleDTO := range dto.DetectionRules {
		ruleModel := models.DataProfileDetectionRuleModel{
			RuleType: types.StringValue(ruleDTO.RuleType),
		}

		if ruleDTO.MultiProfile != nil {
			mpModel := &models.DataProfileMultiProfileModel{
				OperatorType: types.StringValue(ruleDTO.MultiProfile.OperatorType),
			}
			for _, pid := range ruleDTO.MultiProfile.DataProfileIDs {
				mpModel.DataProfileIds = append(mpModel.DataProfileIds, types.Int64Value(pid))
			}
			ruleModel.MultiProfile = mpModel
		}

		if ruleDTO.ExpressionTree != nil {
			treeModel, err := expressionTreeDTOToModel(ruleDTO.ExpressionTree)
			if err != nil {
				return m, err
			}
			ruleModel.ExpressionTree = treeModel
		}

		m.DetectionRules = append(m.DetectionRules, ruleModel)
	}

	return m, nil
}

func expressionTreeDTOToModel(node *dlpclient.ExpressionTreeNodeDTO) (*models.DataProfileExpressionTreeModel, error) {
	tree := &models.DataProfileExpressionTreeModel{
		ExpressionTreeJson: types.StringNull(),
	}

	if node.OperatorType != nil {
		tree.OperatorType = types.StringValue(*node.OperatorType)
	} else {
		tree.OperatorType = types.StringNull()
	}

	if node.RuleItem != nil {
		tree.RuleItem = ruleItemDTOToModel(node.RuleItem)
	}

	for _, subDTO := range node.SubExpressions {
		subModel := models.DataProfileSubExpressionModel{}
		if subDTO.OperatorType != nil {
			subModel.OperatorType = types.StringValue(*subDTO.OperatorType)
		} else {
			subModel.OperatorType = types.StringNull()
		}
		if subDTO.RuleItem != nil {
			subModel.RuleItem = ruleItemDTOToModel(subDTO.RuleItem)
		}

		for _, sub2DTO := range subDTO.SubExpressions {
			sub2Model := models.DataProfileSubExpressionLevel2Model{}
			if sub2DTO.OperatorType != nil {
				sub2Model.OperatorType = types.StringValue(*sub2DTO.OperatorType)
			} else {
				sub2Model.OperatorType = types.StringNull()
			}
			if sub2DTO.RuleItem != nil {
				sub2Model.RuleItem = ruleItemDTOToModel(sub2DTO.RuleItem)
			}
			subModel.SubExpressions = append(subModel.SubExpressions, sub2Model)
		}

		tree.SubExpressions = append(tree.SubExpressions, subModel)
	}

	// Also serialize raw tree JSON for complete fidelity if needed
	rawJSON, err := json.Marshal(node)
	if err == nil {
		tree.ExpressionTreeJson = types.StringValue(string(rawJSON))
	}

	return tree, nil
}

func ruleItemDTOToModel(dto *dlpclient.RuleItemDTO) *models.DataProfileRuleItemModel {
	item := &models.DataProfileRuleItemModel{
		Id:                     types.StringValue(dto.ID),
		DetectionTechnique:     types.StringValue(dto.DetectionTechnique),
		OccurrenceOperatorType: types.StringValue(dto.OccurrenceOperatorType),
	}

	if dto.Name != nil {
		item.Name = types.StringValue(*dto.Name)
	} else {
		item.Name = types.StringNull()
	}

	if dto.OccurrenceCount != nil {
		item.OccurrenceCount = types.Int64Value(*dto.OccurrenceCount)
	} else {
		item.OccurrenceCount = types.Int64Null()
	}

	if dto.OccurrenceLow != nil {
		item.OccurrenceLow = types.Int64Value(*dto.OccurrenceLow)
	} else {
		item.OccurrenceLow = types.Int64Null()
	}

	if dto.OccurrenceHigh != nil {
		item.OccurrenceHigh = types.Int64Value(*dto.OccurrenceHigh)
	} else {
		item.OccurrenceHigh = types.Int64Null()
	}

	if dto.ConfidenceLevel != nil {
		item.ConfidenceLevel = types.StringValue(*dto.ConfidenceLevel)
	} else {
		item.ConfidenceLevel = types.StringValue("high")
	}

	if dto.MatchType != nil {
		item.MatchType = types.StringValue(*dto.MatchType)
	} else {
		item.MatchType = types.StringValue("include")
	}

	if dto.ByUniqueCount != nil {
		item.ByUniqueCount = types.BoolValue(*dto.ByUniqueCount)
	} else {
		item.ByUniqueCount = types.BoolValue(false)
	}

	return item
}
