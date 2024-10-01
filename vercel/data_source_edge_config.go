package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &edgeConfigDataSource{}
	_ datasource.DataSourceWithConfigure = &edgeConfigDataSource{}
)

func newEdgeConfigDataSource() datasource.DataSource {
	return &edgeConfigDataSource{
		dataSourceConfigurer: &dataSourceConfigurer{
			dataSourceNameSuffix: "_edge_config",
		},
	}
}

type edgeConfigDataSource struct {
	*dataSourceConfigurer
}

// Schema returns the schema information for an edgeConfig data source
func (r *edgeConfigDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Edge Config.

An Edge Config is a global data store that enables experimentation with feature flags, A/B testing, critical redirects, and more.`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The Edge Config ID to be retrieved. This can be found by navigating to the Edge Config in the Vercel UI and looking at the URL. It should begin with `ecfg_`.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name/slug of the Edge Config.",
			},
		},
	}
}

// Read will read the edgeConfig information by requesting it from the Vercel API, and will update terraform
// with this information.
// It is called by the provider whenever data source values should be read to update state.
func (d *edgeConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EdgeConfig
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := d.client.GetEdgeConfig(ctx, config.ID.ValueString(), config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading EdgeConfig",
			fmt.Sprintf("Could not get Edge Config %s %s, unexpected error: %s",
				config.TeamID.ValueString(),
				config.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := responseToEdgeConfig(out)
	tflog.Info(ctx, "read edge config", map[string]interface{}{
		"team_id":        result.TeamID.ValueString(),
		"edge_config_id": result.ID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
