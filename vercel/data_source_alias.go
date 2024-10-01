package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &aliasDataSource{}
)

func newAliasDataSource() datasource.DataSource {
	return &aliasDataSource{
		dataSourceConfigurer: &dataSourceConfigurer{
			dataSourceNameSuffix: "_alias",
		},
		reader: &reader[Alias]{
			// readFunc will read the alias information by requesting it from the Vercel API, and will update terraform
			// with this information.
			// It is called by the provider whenever data source values should be read to update state.
			readFunc: func(ctx context.Context, a Alias, client *client.Client, resp *datasource.ReadResponse) (Alias, error) {
				out, err := client.GetAlias(ctx, a.Alias.ValueString(), a.TeamID.ValueString())
				if err != nil {
					resp.Diagnostics.AddError(
						"Error reading alias",
						fmt.Sprintf("Could not read alias %s %s, unexpected error: %s",
							a.TeamID.ValueString(),
							a.Alias.ValueString(),
							err,
						),
					)
					return Alias{}, err
				}

				result := convertResponseToAlias(out, a)
				tflog.Info(ctx, "read alias", map[string]interface{}{
					"team_id": result.TeamID.ValueString(),
					"alias":   result.Alias.ValueString(),
				})
				return result, nil
			},
		},
	}
}

type aliasDataSource struct {
	*dataSourceConfigurer
	*reader[Alias]
}

// Schema returns the schema information for an alias data source
func (r *aliasDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Alias resource.

An Alias allows a ` + "`vercel_deployment` to be accessed through a different URL.",
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Alias and Deployment exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"alias": schema.StringAttribute{
				Required:    true,
				Description: "The Alias or Alias ID to be retrieved.",
			},
			"deployment_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the Deployment the Alias is associated with.",
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}
