package provider

import (
	"context"
	"fmt"

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
// Resource: scm_dlp_peripheral
// ===========================================================================

var (
	_ resource.Resource                = &PeripheralResource{}
	_ resource.ResourceWithConfigure   = &PeripheralResource{}
	_ resource.ResourceWithImportState = &PeripheralResource{}
)

func NewPeripheralResource() resource.Resource {
	return &PeripheralResource{}
}

type PeripheralResource struct {
	client *dlpclient.Client
}

func (r *PeripheralResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "scm_dlp_peripheral"
}

func (r *PeripheralResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.PeripheralResourceSchema
}

func (r *PeripheralResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *PeripheralResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.PeripheralModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto, diags := peripheralModelToDTO(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating DLP Peripheral", map[string]any{"name": dto.Name})
	var created dlpclient.PeripheralDTO
	if err := r.client.Post(ctx, "/v2/api/peripherals", nil, dto, &created); err != nil {
		resp.Diagnostics.AddError("Error Creating DLP Peripheral", err.Error())
		return
	}

	state, diags := peripheralDTOToModel(&created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PeripheralResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.PeripheralModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Peripheral without an ID.")
		return
	}

	var dto dlpclient.PeripheralDTO
	err := r.client.Get(ctx, fmt.Sprintf("/v2/api/peripherals/%s", id), nil, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading DLP Peripheral", err.Error())
		return
	}

	updated, diags := peripheralDTOToModel(&dto)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *PeripheralResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.PeripheralModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state models.PeripheralModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot update Peripheral without an ID.")
		return
	}

	dto, diags := peripheralModelToDTO(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto.ID = &id

	var updated dlpclient.PeripheralDTO
	if err := r.client.Put(ctx, fmt.Sprintf("/v2/api/peripherals/%s", id), nil, dto, &updated); err != nil {
		resp.Diagnostics.AddError("Error Updating DLP Peripheral", err.Error())
		return
	}

	newState, diags := peripheralDTOToModel(&updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *PeripheralResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.PeripheralModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Peripheral without an ID.")
		return
	}

	if err := r.client.Delete(ctx, fmt.Sprintf("/v2/api/peripherals/%s", id), nil); err != nil {
		if !dlpclient.IsNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting DLP Peripheral", err.Error())
			return
		}
	}
}

func (r *PeripheralResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// DataSource: scm_dlp_peripheral
// ===========================================================================

var (
	_ datasource.DataSource              = &PeripheralDataSource{}
	_ datasource.DataSourceWithConfigure = &PeripheralDataSource{}
)

func NewPeripheralDataSource() datasource.DataSource {
	return &PeripheralDataSource{}
}

type PeripheralDataSource struct {
	client *dlpclient.Client
}

func (d *PeripheralDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_peripheral"
}

func (d *PeripheralDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.PeripheralDataSourceSchema
}

func (d *PeripheralDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *PeripheralDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.PeripheralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Attribute 'id' is required for scm_dlp_peripheral data source.")
		return
	}

	var dto dlpclient.PeripheralDTO
	if err := d.client.Get(ctx, fmt.Sprintf("/v2/api/peripherals/%s", id), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Error Reading DLP Peripheral", err.Error())
		return
	}

	state, diags := peripheralDTOToModel(&dto)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ===========================================================================
// DataSource: scm_dlp_peripheral_list
// ===========================================================================

var (
	_ datasource.DataSource              = &PeripheralListDataSource{}
	_ datasource.DataSourceWithConfigure = &PeripheralListDataSource{}
)

func NewPeripheralListDataSource() datasource.DataSource {
	return &PeripheralListDataSource{}
}

type PeripheralListDataSource struct {
	client *dlpclient.Client
}

func (d *PeripheralListDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "scm_dlp_peripheral_list"
}

func (d *PeripheralListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.PeripheralListDataSourceSchema
}

func (d *PeripheralListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *PeripheralListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	allPeripherals, err := dlpclient.GetAll[dlpclient.PeripheralDTO](
		ctx,
		d.client,
		"/v2/api/peripherals",
		nil,
		dlpclient.PaginationZeroBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DLP Peripherals", err.Error())
		return
	}

	var peripheralModels []models.PeripheralModel
	for i := range allPeripherals {
		p := &allPeripherals[i]
		m, diags := peripheralDTOToModel(p)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		peripheralModels = append(peripheralModels, m)
	}

	data := models.PeripheralListModel{
		Total: types.Int64Value(int64(len(peripheralModels))),
		Data:  peripheralModels,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ===========================================================================
// Helpers: Peripheral Model <-> DTO
// ===========================================================================

func peripheralModelToDTO(m *models.PeripheralModel) (*dlpclient.PeripheralDTO, diag.Diagnostics) {
	var diags diag.Diagnostics

	dto := &dlpclient.PeripheralDTO{
		Name: m.Name.ValueString(),
		Type: m.Type.ValueString(),
	}

	if !m.Id.IsNull() && m.Id.ValueString() != "" {
		v := m.Id.ValueString()
		dto.ID = &v
	}
	if !m.Description.IsNull() {
		v := m.Description.ValueString()
		dto.Description = &v
	}
	if !m.PrinterType.IsNull() {
		v := m.PrinterType.ValueString()
		dto.PrinterType = &v
	}
	if !m.ServerAddress.IsNull() {
		v := m.ServerAddress.ValueString()
		dto.ServerAddress = &v
	}
	if !m.VendorId.IsNull() {
		v := m.VendorId.ValueString()
		dto.VendorID = &v
	}
	if !m.ProductId.IsNull() {
		v := m.ProductId.ValueString()
		dto.ProductID = &v
	}

	return dto, diags
}

func peripheralDTOToModel(dto *dlpclient.PeripheralDTO) (models.PeripheralModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var idStr string
	if dto.ID != nil {
		idStr = *dto.ID
	}

	m := models.PeripheralModel{
		Tfid:          types.StringValue(idStr),
		Id:            types.StringValue(idStr),
		Name:          types.StringValue(dto.Name),
		Type:          types.StringValue(dto.Type),
		Description:   types.StringNull(),
		PrinterType:   types.StringNull(),
		ServerAddress: types.StringNull(),
		VendorId:      types.StringNull(),
		ProductId:     types.StringNull(),
	}

	if dto.Description != nil {
		m.Description = types.StringValue(*dto.Description)
	}
	if dto.PrinterType != nil {
		m.PrinterType = types.StringValue(*dto.PrinterType)
	}
	if dto.ServerAddress != nil {
		m.ServerAddress = types.StringValue(*dto.ServerAddress)
	}
	if dto.VendorID != nil {
		m.VendorId = types.StringValue(*dto.VendorID)
	}
	if dto.ProductID != nil {
		m.ProductId = types.StringValue(*dto.ProductID)
	}

	return m, diags
}
