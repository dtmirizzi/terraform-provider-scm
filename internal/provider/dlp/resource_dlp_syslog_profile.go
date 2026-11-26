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

	models "github.com/paloaltonetworks/terraform-provider-scm/internal/models/dlp"
	"github.com/paloaltonetworks/terraform-provider-scm/internal/provider/dlp/client"
)

// ===========================================================================
// Resource: scm_dlp_syslog_profile
// ===========================================================================

var (
	_ resource.Resource                = &SyslogProfileResource{}
	_ resource.ResourceWithConfigure   = &SyslogProfileResource{}
	_ resource.ResourceWithImportState = &SyslogProfileResource{}
)

func NewSyslogProfileResource() resource.Resource {
	return &SyslogProfileResource{}
}

type SyslogProfileResource struct {
	client *dlpclient.Client
}

func (r *SyslogProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_syslog_profile"
}

func (r *SyslogProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = models.SyslogProfileResourceSchema
}

func (r *SyslogProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (r *SyslogProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan models.SyslogProfileModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto, diags := syslogProfileModelToDTO(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating DLP Syslog Profile", map[string]any{"profile_name": dto.ProfileName})
	var created dlpclient.SyslogProfileDTO
	if err := r.client.Post(ctx, "/v1/api/syslog/server-profile", nil, dto, &created); err != nil {
		resp.Diagnostics.AddError("Error Creating DLP Syslog Profile", err.Error())
		return
	}

	state, diags := syslogProfileDTOToModel(&created)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SyslogProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state models.SyslogProfileModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Syslog Profile without an ID.")
		return
	}

	var dto dlpclient.SyslogProfileDTO
	err := r.client.Get(ctx, fmt.Sprintf("/v1/api/syslog/server-profile/%s", dlpclient.EscapeID(id)), nil, &dto)
	if err != nil {
		if dlpclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading DLP Syslog Profile", err.Error())
		return
	}

	updated, diags := syslogProfileDTOToModel(&dto)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &updated)...)
}

func (r *SyslogProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan models.SyslogProfileModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state models.SyslogProfileModel
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
		resp.Diagnostics.AddError("Missing ID", "Cannot update Syslog Profile without an ID.")
		return
	}

	dto, diags := syslogProfileModelToDTO(&plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dto.ID = &id
	dto.ProfileID = &id

	var updated dlpclient.SyslogProfileDTO
	if err := r.client.Put(ctx, fmt.Sprintf("/v1/api/syslog/server-profile/%s", dlpclient.EscapeID(id)), nil, dto, &updated); err != nil {
		resp.Diagnostics.AddError("Error Updating DLP Syslog Profile", err.Error())
		return
	}

	newState, diags := syslogProfileDTOToModel(&updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *SyslogProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state models.SyslogProfileModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.Id.ValueString()
	if id == "" {
		id = state.Tfid.ValueString()
	}
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Syslog Profile without an ID.")
		return
	}

	if err := r.client.Delete(ctx, fmt.Sprintf("/v1/api/syslog/server-profile/%s", dlpclient.EscapeID(id)), nil); err != nil {
		if !dlpclient.IsNotFound(err) {
			resp.Diagnostics.AddError("Error Deleting DLP Syslog Profile", err.Error())
			return
		}
	}
}

func (r *SyslogProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ===========================================================================
// DataSource: scm_dlp_syslog_profile
// ===========================================================================

var (
	_ datasource.DataSource              = &SyslogProfileDataSource{}
	_ datasource.DataSourceWithConfigure = &SyslogProfileDataSource{}
)

func NewSyslogProfileDataSource() datasource.DataSource {
	return &SyslogProfileDataSource{}
}

type SyslogProfileDataSource struct {
	client *dlpclient.Client
}

func (d *SyslogProfileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_syslog_profile"
}

func (d *SyslogProfileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.SyslogProfileDataSourceSchema
}

func (d *SyslogProfileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *SyslogProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config models.SyslogProfileModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := config.Id.ValueString()
	if id == "" {
		resp.Diagnostics.AddError("Missing ID", "Attribute 'id' is required for scm_dlp_syslog_profile data source.")
		return
	}

	var dto dlpclient.SyslogProfileDTO
	if err := d.client.Get(ctx, fmt.Sprintf("/v1/api/syslog/server-profile/%s", dlpclient.EscapeID(id)), nil, &dto); err != nil {
		resp.Diagnostics.AddError("Error Reading DLP Syslog Profile", err.Error())
		return
	}

	state, diags := syslogProfileDTOToModel(&dto)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ===========================================================================
// DataSource: scm_dlp_syslog_profile_list
// ===========================================================================

var (
	_ datasource.DataSource              = &SyslogProfileListDataSource{}
	_ datasource.DataSourceWithConfigure = &SyslogProfileListDataSource{}
)

func NewSyslogProfileListDataSource() datasource.DataSource {
	return &SyslogProfileListDataSource{}
}

type SyslogProfileListDataSource struct {
	client *dlpclient.Client
}

func (d *SyslogProfileListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dlp_syslog_profile_list"
}

func (d *SyslogProfileListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = models.SyslogProfileListDataSourceSchema
}

func (d *SyslogProfileListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = extractDLPClient(req.ProviderData, &resp.Diagnostics)
}

func (d *SyslogProfileListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	allProfiles, err := dlpclient.GetAll[dlpclient.SyslogProfileDTO](
		ctx,
		d.client,
		"/v1/api/syslog/server-profile",
		nil,
		dlpclient.PaginationOneBased,
		100,
		50,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error Listing DLP Syslog Profiles", err.Error())
		return
	}

	var profileModels []models.SyslogProfileModel
	for i := range allProfiles {
		p := &allProfiles[i]
		m, diags := syslogProfileDTOToModel(p)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		profileModels = append(profileModels, m)
	}

	data := models.SyslogProfileListModel{
		Total: types.Int64Value(int64(len(profileModels))),
		Data:  profileModels,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// ===========================================================================
// Helpers: Syslog Profile Model <-> DTO
// ===========================================================================

func syslogProfileModelToDTO(m *models.SyslogProfileModel) (*dlpclient.SyslogProfileDTO, diag.Diagnostics) {
	var diags diag.Diagnostics

	dto := &dlpclient.SyslogProfileDTO{
		ProfileName: m.ProfileName.ValueString(),
		Address:     m.Address.ValueString(),
		Port:        m.Port.ValueInt64(),
		AlertEmail:  m.AlertEmail.ValueString(),
		Format:      m.Format.ValueString(),
	}

	if !m.Id.IsNull() && !m.Id.IsUnknown() && m.Id.ValueString() != "" {
		v := m.Id.ValueString()
		dto.ID = &v
		dto.ProfileID = &v
	}
	if !m.ConnectionType.IsNull() && !m.ConnectionType.IsUnknown() {
		v := m.ConnectionType.ValueString()
		dto.ConnectionType = &v
	}
	if !m.Cert.IsNull() && !m.Cert.IsUnknown() {
		v := m.Cert.ValueString()
		dto.Cert = &v
	}

	return dto, diags
}

func syslogProfileDTOToModel(dto *dlpclient.SyslogProfileDTO) (models.SyslogProfileModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	var idStr string
	if dto.ID != nil {
		idStr = *dto.ID
	} else if dto.ProfileID != nil {
		idStr = *dto.ProfileID
	}

	m := models.SyslogProfileModel{
		Tfid:           types.StringValue(idStr),
		Id:             types.StringValue(idStr),
		ProfileName:    types.StringValue(dto.ProfileName),
		Address:        types.StringValue(dto.Address),
		Port:           types.Int64Value(dto.Port),
		AlertEmail:     types.StringValue(dto.AlertEmail),
		Format:         types.StringValue(dto.Format),
		ConnectionType: types.StringNull(),
		Cert:           types.StringNull(),
	}

	if dto.ConnectionType != nil {
		m.ConnectionType = types.StringValue(*dto.ConnectionType)
	}
	if dto.Cert != nil {
		m.Cert = types.StringValue(*dto.Cert)
	}

	return m, diags
}
