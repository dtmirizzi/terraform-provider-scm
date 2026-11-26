package provider

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/paloaltonetworks/terraform-provider-scm/internal/dlpclient"
	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
)

// ===========================================================================
// Resource: scm_dlp_document_type
// ===========================================================================

var (
	_ resource.Resource                = &DocumentTypeResource{}
	_ resource.ResourceWithConfigure   = &DocumentTypeResource{}
	_ resource.ResourceWithImportState = &DocumentTypeResource{}
)

func NewDocumentTypeResource() resource.Resource {
	return &DocumentTypeResource{}
}

type DocumentTypeResource struct {
	client *dlpclient.Client
}

func (r *DocumentTypeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "scm_dlp_document_type"
}

func (r *DocumentTypeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.DocumentTypeResourceSchema
}

func (r *DocumentTypeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *DocumentTypeResource) validateDocumentTypeDTO(dto *dlpclient.DocumentTypeDTO) []string {
	var errs []string
	if strings.TrimSpace(dto.Name) == "" {
		errs = append(errs, "'name' is required")
	}
	if strings.TrimSpace(dto.OriginalFileName) == "" {
		errs = append(errs, "'original_file_name' is required")
	}
	if !slices.Contains(dlpclient.ResourceCategories, dto.Category) {
		errs = append(errs, fmt.Sprintf("'category' must be one of: %s (got: %q)", strings.Join(dlpclient.ResourceCategories, ", "), dto.Category))
	}
	if !slices.Contains(dlpclient.DocTypeTechniques, dto.DetectionTechnique) {
		errs = append(errs, fmt.Sprintf("'detection_technique' must be one of: %s (got: %q)", strings.Join(dlpclient.DocTypeTechniques, ", "), dto.DetectionTechnique))
	}
	return errs
}

func (r *DocumentTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.DocumentTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto := documentTypeModelToDTO(&plan)
	if errs := r.validateDocumentTypeDTO(dto); len(errs) > 0 {
		for _, e := range errs {
			resp.Diagnostics.AddError("Document Type Validation Failed", e)
		}
		return
	}

	tflog.Debug(ctx, "Creating DLP Document Type", map[string]any{"name": dto.Name})
	var created dlpclient.DocumentTypeDTO
	if err := r.client.Post(ctx, "/v2/api/document-types", nil, dto, &created); err != nil {
		resp.Diagnostics.AddError("Error Creating DLP Document Type", err.Error())
		return
	}

	state := documentTypeDTOToModel(&created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DocumentTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.DocumentTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Document Type without an ID.")
		return
	}

	tflog.Debug(ctx, "Reading DLP Document Type", map[string]any{"id": id})
	var dto dlpclient.DocumentTypeDTO
	err := r.client.Get(ctx, fmt.Sprintf("/v2/api/document-types/%s", url.PathEscape(id)), nil, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading DLP Document Type", err.Error())
		return
	}

	newState := documentTypeDTOToModel(&dto)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *DocumentTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.DocumentTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.Id.ValueString()
	if id == "" {
		id = plan.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot update Document Type without an ID.")
		return
	}

	dto := documentTypeModelToDTO(&plan)
	dto.ID = &id
	if errs := r.validateDocumentTypeDTO(dto); len(errs) > 0 {
		for _, e := range errs {
			resp.Diagnostics.AddError("Document Type Validation Failed", e)
		}
		return
	}

	tflog.Debug(ctx, "Updating DLP Document Type", map[string]any{"id": id})
	var updated dlpclient.DocumentTypeDTO
	if err := r.client.Put(ctx, fmt.Sprintf("/v2/api/document-types/%s", url.PathEscape(id)), nil, dto, &updated); err != nil {
		resp.Diagnostics.AddError("Error Updating DLP Document Type", err.Error())
		return
	}

	state := documentTypeDTOToModel(&updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DocumentTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.DocumentTypeModel
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

	tflog.Debug(ctx, "Deleting DLP Document Type", map[string]any{"id": id})
	if err := r.client.Delete(ctx, fmt.Sprintf("/v2/api/document-types/%s", url.PathEscape(id)), nil); err != nil {
		if !dlpclient.IsNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting DLP Document Type", err.Error())
			return
		}
	}
}

func (r *DocumentTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// DataSource: scm_dlp_document_type
// ===========================================================================

var (
	_ datasource.DataSource              = &DocumentTypeDataSource{}
	_ datasource.DataSourceWithConfigure = &DocumentTypeDataSource{}
)

func NewDocumentTypeDataSource() datasource.DataSource {
	return &DocumentTypeDataSource{}
}

type DocumentTypeDataSource struct {
	client *dlpclient.Client
}

func (d *DocumentTypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_document_type"
}

func (d *DocumentTypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DocumentTypeDataSourceSchema
}

func (d *DocumentTypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *DocumentTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.DocumentTypeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Attribute 'id' is required for scm_dlp_document_type data source.")
		return
	}

	var dto dlpclient.DocumentTypeDTO
	if err := d.client.Get(ctx, fmt.Sprintf("/v2/api/document-types/%s", url.PathEscape(id)), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Error Reading DLP Document Type", err.Error())
		return
	}

	state := documentTypeDTOToModel(&dto)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ===========================================================================
// DataSource: scm_dlp_document_type_list
// ===========================================================================

var (
	_ datasource.DataSource              = &DocumentTypeListDataSource{}
	_ datasource.DataSourceWithConfigure = &DocumentTypeListDataSource{}
)

func NewDocumentTypeListDataSource() datasource.DataSource {
	return &DocumentTypeListDataSource{}
}

type DocumentTypeListDataSource struct {
	client *dlpclient.Client
}

func (d *DocumentTypeListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_document_type_list"
}

func (d *DocumentTypeListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.DocumentTypeListDataSourceSchema
}

func (d *DocumentTypeListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *DocumentTypeListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	allDocs, err := dlpclient.GetAll[dlpclient.DocumentTypeDTO](
		ctx,
		d.client,
		"/v2/api/document-types",
		nil,
		dlpclient.PaginationZeroBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DLP Document Types", err.Error())
		return
	}

	var docModels []models.DocumentTypeModel
	for i := range allDocs {
		m := documentTypeDTOToModel(&allDocs[i])
		docModels = append(docModels, m)
	}

	data := models.DocumentTypeListModel{
		Total: types.Int64Value(int64(len(docModels))),
		Data:  docModels,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ===========================================================================
// Helper mappers for Document Type
// ===========================================================================

func documentTypeModelToDTO(m *models.DocumentTypeModel) *dlpclient.DocumentTypeDTO {
	dto := &dlpclient.DocumentTypeDTO{
		Name:               m.Name.ValueString(),
		Category:           m.Category.ValueString(),
		DetectionTechnique: m.DetectionTechnique.ValueString(),
		OriginalFileName:   m.OriginalFileName.ValueString(),
	}

	if !m.Id.IsNull() && !m.Id.IsUnknown() && m.Id.ValueString() != "" {
		v := m.Id.ValueString()
		dto.ID = &v
	}
	if !m.Description.IsNull() && !m.Description.IsUnknown() {
		v := m.Description.ValueString()
		dto.Description = &v
	}
	if !m.L1CategoryID.IsNull() && !m.L1CategoryID.IsUnknown() {
		v := m.L1CategoryID.ValueInt64()
		dto.L1CategoryID = &v
	}
	if !m.L2CategoryID.IsNull() && !m.L2CategoryID.IsUnknown() {
		v := m.L2CategoryID.ValueInt64()
		dto.L2CategoryID = &v
	}

	return dto
}

func documentTypeDTOToModel(dto *dlpclient.DocumentTypeDTO) models.DocumentTypeModel {
	var idStr string
	if dto.ID != nil {
		idStr = *dto.ID
	}

	m := models.DocumentTypeModel{
		Tfid:               types.StringValue(idStr),
		Id:                 types.StringValue(idStr),
		Name:               types.StringValue(dto.Name),
		Category:           types.StringValue(dto.Category),
		DetectionTechnique: types.StringValue(dto.DetectionTechnique),
		OriginalFileName:   types.StringValue(dto.OriginalFileName),
		Description:        types.StringNull(),
		L1CategoryID:       types.Int64Null(),
		L2CategoryID:       types.Int64Null(),
	}

	if dto.Description != nil {
		m.Description = types.StringValue(*dto.Description)
	}
	if dto.L1CategoryID != nil {
		m.L1CategoryID = types.Int64Value(*dto.L1CategoryID)
	}
	if dto.L2CategoryID != nil {
		m.L2CategoryID = types.Int64Value(*dto.L2CategoryID)
	}

	return m
}
