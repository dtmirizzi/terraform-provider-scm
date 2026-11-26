package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/paloaltonetworks/terraform-provider-scm/internal/provider/dlp/client"
)

func extractDLPClient(providerData any, diags *diag.Diagnostics) *dlpclient.Client {
	if providerData == nil {
		return nil
	}
	clients, ok := providerData.(map[string]any)
	if !ok {
		diags.AddError("Unexpected Configure Type", fmt.Sprintf("Expected map[string]interface{}, got: %T.", providerData))
		return nil
	}
	client, ok := clients["dlp"].(*dlpclient.Client)
	if !ok {
		diags.AddError("Unexpected Client Type", "Expected *dlpclient.Client for 'dlp' client.")
		return nil
	}
	return client
}
