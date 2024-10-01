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
	_ datasource.DataSource              = &attackChallengeModeDataSource{}
	_ datasource.DataSourceWithConfigure = &attackChallengeModeDataSource{}
)

func newAttackChallengeModeDataSource() datasource.DataSource {
	return &attackChallengeModeDataSource{
		dataSourceConfigurer: &dataSourceConfigurer{
			dataSourceNameSuffix: "_attack_challenge_mode",
		},
		reader: &reader[AttackChallengeMode]{
			readFunc: func(ctx context.Context, config AttackChallengeMode, c *client.Client, resp *datasource.ReadResponse) (AttackChallengeMode, error) {
				out, err := c.GetAttackChallengeMode(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
				if client.NotFound(err) {
					resp.State.RemoveResource(ctx)
					return AttackChallengeMode{}, err
				}
				if err != nil {
					resp.Diagnostics.AddError(
						"Error reading Attack Challenge Mode",
						fmt.Sprintf("Could not get Attack Challenge Mode %s %s, unexpected error: %s",
							config.TeamID.ValueString(),
							config.ProjectID.ValueString(),
							err,
						),
					)
					return AttackChallengeMode{}, err
				}

				result := responseToAttackChallengeMode(out)
				tflog.Info(ctx, "read attack challenge mode", map[string]interface{}{
					"team_id":    result.TeamID.ValueString(),
					"project_id": result.ProjectID.ValueString(),
				})
				return result, nil
			},
		},
	}
}

type attackChallengeModeDataSource struct {
	*dataSourceConfigurer
	*reader[AttackChallengeMode]
}

func (r *attackChallengeModeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides an Attack Challenge Mode resource.

Attack Challenge Mode prevent malicious traffic by showing a verification challenge for every visitor.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The resource identifier.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the Project to adjust the CPU for.",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Project exists under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
			"enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether Attack Challenge Mode is enabled or not.",
			},
		},
	}
}
