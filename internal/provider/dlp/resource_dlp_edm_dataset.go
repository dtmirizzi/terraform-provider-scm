package provider

import (
	"context"
	"fmt"
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
// Resource: scm_dlp_edm_dataset
// ===========================================================================

var (
	_ resource.Resource                = &EdmDatasetResource{}
	_ resource.ResourceWithConfigure   = &EdmDatasetResource{}
	_ resource.ResourceWithImportState = &EdmDatasetResource{}
)

func NewEdmDatasetResource() resource.Resource {
	return &EdmDatasetResource{}
}

type EdmDatasetResource struct {
	client *dlpclient.Client
}

func (r *EdmDatasetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_edm_dataset"
}

func (r *EdmDatasetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.EdmDatasetResourceSchema
}

func (r *EdmDatasetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *EdmDatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Only id and name are user-settable. The computed field lists are unknown in
	// the plan and cannot be decoded into the model's Go slices, so read the two
	// attributes individually instead of the whole plan.
	var plan models.EdmDatasetModel
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("id"), &plan.Id)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("name"), &plan.Name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	targetID := plan.Id.ValueString()

	// If ID is not explicitly provided, look up existing dataset by name
	if targetID == "" {
		allDatasets, err := dlpclient.GetAll[dlpclient.EdmDatasetDTO](
			ctx,
			r.client,
			"/v2/api/edm-datasets",
			nil,
			dlpclient.PaginationZeroBased,
			100,
			50,
		)
		if err != nil {
			resp.Diagnostics.AddError("Error Looking Up EDM Dataset by Name", err.Error())
			return
		}

		planName := plan.Name.ValueString()
		for _, ds := range allDatasets {
			if strings.EqualFold(ds.Name, planName) && ds.ID != nil {
				targetID = *ds.ID
				break
			}
		}

		if targetID == "" {
			resp.Diagnostics.AddError(
				"EDM Dataset Not Found",
				fmt.Sprintf("EDM datasets must be uploaded and indexed via the EDM CLI before adopting into Terraform. No dataset found with name %q or id %q.", planName, plan.Id.ValueString()),
			)
			return
		}
	}

	tflog.Debug(ctx, "Adopting EDM Dataset into Terraform State", map[string]any{"id": targetID})
	var dto dlpclient.EdmDatasetDTO
	if err := r.client.Get(ctx, fmt.Sprintf("/v2/api/edm-datasets/%s", dlpclient.EscapeID(targetID)), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Error Reading EDM Dataset", err.Error())
		return
	}

	state := edmDatasetDTOToModel(&dto)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EdmDatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.EdmDatasetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read EDM Dataset without an ID.")
		return
	}

	tflog.Debug(ctx, "Reading DLP EDM Dataset", map[string]any{"id": id})
	var dto dlpclient.EdmDatasetDTO
	err := r.client.Get(ctx, fmt.Sprintf("/v2/api/edm-datasets/%s", dlpclient.EscapeID(id)), nil, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading DLP EDM Dataset", err.Error())
		return
	}
	if dto.IndexStatus != nil && strings.EqualFold(*dto.IndexStatus, "DELETED") {
		tflog.Debug(ctx, "EDM Dataset has index_status DELETED, removing from state", map[string]any{"id": id})
		resp.State.RemoveResource(ctx)
		return
	}

	newState := edmDatasetDTOToModel(&dto)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *EdmDatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Schema attributes for EDM dataset other than name are computed.
	// We re-read dataset from backend to refresh state.
	var plan models.EdmDatasetModel
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &plan.Id)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("tfid"), &plan.Tfid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.Id.ValueString()
	if id == "" {
		id = plan.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot update EDM Dataset without an ID.")
		return
	}

	tflog.Debug(ctx, "Refreshing DLP EDM Dataset", map[string]any{"id": id})
	var dto dlpclient.EdmDatasetDTO
	if err := r.client.Get(ctx, fmt.Sprintf("/v2/api/edm-datasets/%s", dlpclient.EscapeID(id)), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Error Refreshing DLP EDM Dataset", err.Error())
		return
	}

	state := edmDatasetDTOToModel(&dto)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EdmDatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// EDM datasets are created and indexed out-of-band with the EDM CLI; this
	// resource only adopts an existing dataset. Destroying it therefore only
	// removes it from Terraform state and never deletes the uploaded data.
	var state models.EdmDatasetModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Removing DLP EDM Dataset from Terraform state; the dataset itself is not deleted", map[string]any{"id": state.Id.ValueString()})
}

func (r *EdmDatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// DataSource: scm_dlp_edm_dataset
// ===========================================================================

var (
	_ datasource.DataSource              = &EdmDatasetDataSource{}
	_ datasource.DataSourceWithConfigure = &EdmDatasetDataSource{}
)

func NewEdmDatasetDataSource() datasource.DataSource {
	return &EdmDatasetDataSource{}
}

type EdmDatasetDataSource struct {
	client *dlpclient.Client
}

func (d *EdmDatasetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_edm_dataset"
}

func (d *EdmDatasetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.EdmDatasetDataSourceSchema
}

func (d *EdmDatasetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *EdmDatasetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.EdmDatasetModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Attribute 'id' is required for scm_dlp_edm_dataset data source.")
		return
	}

	var dto dlpclient.EdmDatasetDTO
	if err := d.client.Get(ctx, fmt.Sprintf("/v2/api/edm-datasets/%s", dlpclient.EscapeID(id)), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Error Reading DLP EDM Dataset", err.Error())
		return
	}

	state := edmDatasetDTOToModel(&dto)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ===========================================================================
// DataSource: scm_dlp_edm_dataset_list
// ===========================================================================

var (
	_ datasource.DataSource              = &EdmDatasetListDataSource{}
	_ datasource.DataSourceWithConfigure = &EdmDatasetListDataSource{}
)

func NewEdmDatasetListDataSource() datasource.DataSource {
	return &EdmDatasetListDataSource{}
}

type EdmDatasetListDataSource struct {
	client *dlpclient.Client
}

func (d *EdmDatasetListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_edm_dataset_list"
}

func (d *EdmDatasetListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.EdmDatasetListDataSourceSchema
}

func (d *EdmDatasetListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *EdmDatasetListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.EdmDatasetListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	allDatasets, err := dlpclient.GetAll[dlpclient.EdmDatasetDTO](
		ctx,
		d.client,
		"/v2/api/edm-datasets",
		nil,
		dlpclient.PaginationZeroBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DLP EDM Datasets", err.Error())
		return
	}

	nameFilter := ""
	if !config.Name.IsNull() && config.Name.ValueString() != "" {
		nameFilter = config.Name.ValueString()
	}

	var datasetModels []models.EdmDatasetModel
	for i := range allDatasets {
		ds := &allDatasets[i]
		if nameFilter != "" && !strings.EqualFold(ds.Name, nameFilter) {
			continue
		}
		datasetModels = append(datasetModels, edmDatasetDTOToModel(ds))
	}

	data := models.EdmDatasetListModel{
		Name:  config.Name,
		Total: types.Int64Value(int64(len(datasetModels))),
		Data:  datasetModels,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ===========================================================================
// Helper mappers for EDM Dataset
// ===========================================================================

func edmFieldsToModel(fields []dlpclient.EdmFieldDTO) []models.EdmFieldModel {
	if fields == nil {
		return nil
	}
	out := make([]models.EdmFieldModel, len(fields))
	for i, f := range fields {
		out[i] = models.EdmFieldModel{
			Id:   types.StringValue(f.ID),
			Name: types.StringValue(f.Name),
			Type: types.StringValue(f.Type),
		}
	}
	return out
}

func edmDatasetDTOToModel(dto *dlpclient.EdmDatasetDTO) models.EdmDatasetModel {
	var idStr string
	if dto.ID != nil {
		idStr = *dto.ID
	}

	m := models.EdmDatasetModel{
		Tfid:           types.StringValue(idStr),
		Id:             types.StringValue(idStr),
		Name:           types.StringValue(dto.Name),
		RegionName:     types.StringNull(),
		TenantId:       types.StringNull(),
		IndexStatus:    types.StringNull(),
		IndexErrorCode: types.StringNull(),
		ActiveFields:   edmFieldsToModel(dto.ActiveFields),
		AllFields:      edmFieldsToModel(dto.AllFields),
	}

	if dto.RegionName != nil {
		m.RegionName = types.StringValue(*dto.RegionName)
	}
	if dto.TenantID != nil {
		m.TenantId = types.StringValue(*dto.TenantID)
	}
	if dto.IndexStatus != nil {
		m.IndexStatus = types.StringValue(*dto.IndexStatus)
	}
	if dto.IndexErrorCode != nil {
		m.IndexErrorCode = types.StringValue(*dto.IndexErrorCode)
	}

	return m
}
