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
// Resource: scm_dlp_forward_profile
// ===========================================================================

var (
	_ resource.Resource                = &ForwardProfileResource{}
	_ resource.ResourceWithConfigure   = &ForwardProfileResource{}
	_ resource.ResourceWithImportState = &ForwardProfileResource{}
)

func NewForwardProfileResource() resource.Resource {
	return &ForwardProfileResource{}
}

type ForwardProfileResource struct {
	client *dlpclient.Client
}

func (r *ForwardProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_forward_profile"
}

func (r *ForwardProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.ForwardProfileResourceSchema
}

func (r *ForwardProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *ForwardProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.ForwardProfileModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto, diags := forwardProfileModelToDTO(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating DLP Forward Profile", map[string]any{"name": dto.Name})
	var created dlpclient.ForwardProfileDTO
	if err := r.client.Post(ctx, "/v1/api/forward-profile", nil, dto, &created); err != nil {
		resp.Diagnostics.AddError("Error Creating DLP Forward Profile", err.Error())
		return
	}

	state, diags := forwardProfileDTOToModel(ctx, &created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ForwardProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.ForwardProfileModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Forward Profile without an ID.")
		return
	}

	var dto dlpclient.ForwardProfileDTO
	err := r.client.Get(ctx, fmt.Sprintf("/v1/api/forward-profile/%s", dlpclient.EscapeID(id)), nil, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading DLP Forward Profile", err.Error())
		return
	}

	updated, diags := forwardProfileDTOToModel(ctx, &dto)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *ForwardProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.ForwardProfileModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state models.ForwardProfileModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot update Forward Profile without an ID.")
		return
	}

	dto, diags := forwardProfileModelToDTO(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto.ID = &id
	dto.ProfileID = &id

	var updated dlpclient.ForwardProfileDTO
	if err := r.client.Put(ctx, fmt.Sprintf("/v1/api/forward-profile/%s", dlpclient.EscapeID(id)), nil, dto, &updated); err != nil {
		resp.Diagnostics.AddError("Error Updating DLP Forward Profile", err.Error())
		return
	}

	newState, diags := forwardProfileDTOToModel(ctx, &updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *ForwardProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.ForwardProfileModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Forward Profile without an ID.")
		return
	}

	if err := r.client.Delete(ctx, fmt.Sprintf("/v1/api/forward-profile/%s", dlpclient.EscapeID(id)), nil); err != nil {
		if !dlpclient.IsNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting DLP Forward Profile", err.Error())
			return
		}
	}
}

func (r *ForwardProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// DataSource: scm_dlp_forward_profile
// ===========================================================================

var (
	_ datasource.DataSource              = &ForwardProfileDataSource{}
	_ datasource.DataSourceWithConfigure = &ForwardProfileDataSource{}
)

func NewForwardProfileDataSource() datasource.DataSource {
	return &ForwardProfileDataSource{}
}

type ForwardProfileDataSource struct {
	client *dlpclient.Client
}

func (d *ForwardProfileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_forward_profile"
}

func (d *ForwardProfileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.ForwardProfileDataSourceSchema
}

func (d *ForwardProfileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *ForwardProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.ForwardProfileModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Attribute 'id' is required for scm_dlp_forward_profile data source.")
		return
	}

	var dto dlpclient.ForwardProfileDTO
	if err := d.client.Get(ctx, fmt.Sprintf("/v1/api/forward-profile/%s", dlpclient.EscapeID(id)), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Error Reading DLP Forward Profile", err.Error())
		return
	}

	state, diags := forwardProfileDTOToModel(ctx, &dto)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ===========================================================================
// DataSource: scm_dlp_forward_profile_list
// ===========================================================================

var (
	_ datasource.DataSource              = &ForwardProfileListDataSource{}
	_ datasource.DataSourceWithConfigure = &ForwardProfileListDataSource{}
)

func NewForwardProfileListDataSource() datasource.DataSource {
	return &ForwardProfileListDataSource{}
}

type ForwardProfileListDataSource struct {
	client *dlpclient.Client
}

func (d *ForwardProfileListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_forward_profile_list"
}

func (d *ForwardProfileListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.ForwardProfileListDataSourceSchema
}

func (d *ForwardProfileListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *ForwardProfileListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	allProfiles, err := dlpclient.GetAll[dlpclient.ForwardProfileDTO](
		ctx,
		d.client,
		"/v1/api/forward-profile",
		nil,
		dlpclient.PaginationOneBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DLP Forward Profiles", err.Error())
		return
	}

	var profileModels []models.ForwardProfileModel
	for i := range allProfiles {
		p := &allProfiles[i]
		m, diags := forwardProfileDTOToModel(ctx, p)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		profileModels = append(profileModels, m)
	}

	data := models.ForwardProfileListModel{
		Total: types.Int64Value(int64(len(profileModels))),
		Data:  profileModels,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ===========================================================================
// Helpers: Forward Profile Model <-> DTO
// ===========================================================================

func forwardProfileModelToDTO(ctx context.Context, m *models.ForwardProfileModel) (*dlpclient.ForwardProfileDTO, diag.Diagnostics) {
	var diags diag.Diagnostics

	dto := &dlpclient.ForwardProfileDTO{
		Name:    m.Name.ValueString(),
		Enabled: m.Enabled.ValueBool(),
	}

	if !m.Id.IsNull() && !m.Id.IsUnknown() && m.Id.ValueString() != "" {
		v := m.Id.ValueString()
		dto.ID = &v
		dto.ProfileID = &v
	}

	if !m.Regions.IsNull() && !m.Regions.IsUnknown() {
		var regions []string
		diags.Append(m.Regions.ElementsAs(ctx, &regions, false)...)
		dto.Filter = &dlpclient.ForwardProfileFilterDTO{
			Region: regions,
		}
	}

	return dto, diags
}

func forwardProfileDTOToModel(ctx context.Context, dto *dlpclient.ForwardProfileDTO) (models.ForwardProfileModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var idStr string
	if dto.ID != nil {
		idStr = *dto.ID
	} else if dto.ProfileID != nil {
		idStr = *dto.ProfileID
	}

	m := models.ForwardProfileModel{
		Tfid:    types.StringValue(idStr),
		Id:      types.StringValue(idStr),
		Name:    types.StringValue(dto.Name),
		Enabled: types.BoolValue(dto.Enabled),
		Regions: types.ListNull(types.StringType),
	}

	if dto.Filter != nil && dto.Filter.Region != nil {
		regList, d := types.ListValueFrom(ctx, types.StringType, dto.Filter.Region)
		diags.Append(d...)
		m.Regions = regList
	}

	return m, diags
}
