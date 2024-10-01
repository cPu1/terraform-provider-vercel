package vercel

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = &edgeConfigSchemaDataSource{}
)

func newEdgeConfigSchemaDataSource() datasource.DataSource {
	return &edgeConfigSchemaDataSource{
		dataSourceConfigurer: &dataSourceConfigurer{
			dataSourceNameSuffix: "_edge_config_schema",
		},
		reader: &reader[EdgeConfigSchema]{
			// readFunc will read the edgeConfig information by requesting it from the Vercel API, and will update terraform
			// with this information.
			// It is called by the provider whenever data source values should be read to update state.
			readFunc: func(ctx context.Context, config EdgeConfigSchema, c *client.Client, resp *datasource.ReadResponse) (EdgeConfigSchema, error) {
				out, err := c.GetEdgeConfigSchema(ctx, config.ID.ValueString(), config.TeamID.ValueString())
				if err != nil {
					resp.Diagnostics.AddError(
						"Error reading Edge Config Schema",
						fmt.Sprintf("Could not get Edge Config Schema %s %s, unexpected error: %s",
							config.TeamID.ValueString(),
							config.ID.ValueString(),
							err,
						),
					)
					return EdgeConfigSchema{}, err
				}

				def, err := json.Marshal(out.Definition)
				if err != nil {
					resp.Diagnostics.AddError(
						"Error reading Edge Config Schema",
						fmt.Sprintf("Could not marshal Edge Config Schema %s %s, unexpected error: %s",
							config.TeamID.ValueString(), config.ID.ValueString(), err,
						),
					)
					return EdgeConfigSchema{}, err
				}
				result := responseToEdgeConfigSchema(out, types.StringValue(string(def)))
				tflog.Info(ctx, "read edge config schema", map[string]interface{}{
					"team_id":        result.TeamID.ValueString(),
					"edge_config_id": result.ID.ValueString(),
				})
				return result, nil
			},
		},
	}
}

type edgeConfigSchemaDataSource struct {
	*dataSourceConfigurer
	*reader[EdgeConfigSchema]
}

// Schema returns the schema information for an edgeConfig data source
func (r *edgeConfigSchemaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
An Edge Config Schema provides an existing Edge Config with a JSON schema. Use schema protection to prevent unexpected updates that may cause bugs or downtime.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Edge Config that the schema should be for.",
				Required:    true,
			},
			"definition": schema.StringAttribute{
				Computed:    true,
				Description: "A JSON schema that will be used to validate data in the Edge Config.",
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
		},
	}
}
