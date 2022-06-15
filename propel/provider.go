package propel

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	pc "github.com/propeldata/terraform-provider/propel_client"
)

// Provider -
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"client_id": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   false,
				DefaultFunc: schema.EnvDefaultFunc("PROPEL_CLIENT_ID", nil),
				Description: "The CLIENT_ID for API operations.",
			},
			"client_secret": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("PROPEL_CLIENT_SECRET", nil),
				Description: "The CLIENT_SECRET for API operations.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"propel_data_source": resourceDataSource(),
			"propel_data_pool":   resourceDataPool(),
			"propel_metric":      resourceMetric(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	clientID := d.Get("client_id").(string)
	clientSecret := d.Get("client_secret").(string)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	if (clientID == "") || (clientSecret == "") {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Credentials are required",
			Detail:   "Unable to authenticate for the Propel client",
		})

		return nil, diags
	}

	c, err := pc.NewPropelClient(clientID, clientSecret)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return c, nil
}
