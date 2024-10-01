package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

var (
	_ datasource.DataSourceWithConfigure = &projectDeploymentRetentionDataSource{}
)

func newProjectDeploymentRetentionDataSource() datasource.DataSource {
	return &projectDeploymentRetentionDataSource{
		dataSourceConfigurer: &dataSourceConfigurer{
			dataSourceNameSuffix: "_project_deployment_retention",
		},
		reader: &reader[ProjectDeploymentRetentionWithID]{
			// readFunc will read a deployment retention of a Vercel project by requesting it from the Vercel API, and will update Terraform
			// with this information.
			readFunc: func(ctx context.Context, config ProjectDeploymentRetentionWithID, c *client.Client, resp *datasource.ReadResponse) (ProjectDeploymentRetentionWithID, error) {
				out, err := c.GetDeploymentRetention(ctx, config.ProjectID.ValueString(), config.TeamID.ValueString())
				if client.NotFound(err) {
					resp.State.RemoveResource(ctx)
					return ProjectDeploymentRetentionWithID{}, err
				}
				if err != nil {
					resp.Diagnostics.AddError(
						"Error reading project deployment retention",
						fmt.Sprintf("Could not get project deployment retention %s %s, unexpected error: %s",
							config.ProjectID.ValueString(),
							config.TeamID.ValueString(),
							err,
						),
					)
					return ProjectDeploymentRetentionWithID{}, err
				}

				result := convertResponseToProjectDeploymentRetention(out, config.ProjectID, config.TeamID)
				tflog.Info(ctx, "read project deployment retention", map[string]interface{}{
					"team_id":    result.TeamID.ValueString(),
					"project_id": result.ProjectID.ValueString(),
				})

				return ProjectDeploymentRetentionWithID{
					ExpirationPreview:    result.ExpirationPreview,
					ExpirationProduction: result.ExpirationProduction,
					ExpirationCanceled:   result.ExpirationCanceled,
					ExpirationErrored:    result.ExpirationErrored,
					ProjectID:            result.ProjectID,
					TeamID:               result.TeamID,
					ID:                   result.ProjectID,
				}, nil
			},
		},
	}
}

type projectDeploymentRetentionDataSource struct {
	*dataSourceConfigurer
	*reader[ProjectDeploymentRetentionWithID]
}

// Schema returns the schema information for a project deployment retention datasource.
func (r *projectDeploymentRetentionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Project Deployment Retention datasource.

A Project Deployment Retention datasource details information about Deployment Retention on a Vercel Project.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/security/deployment-retention).
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"expiration_preview": schema.StringAttribute{
				Computed:    true,
				Description: "The retention period for preview deployments.",
			},
			"expiration_production": schema.StringAttribute{
				Computed:    true,
				Description: "The retention period for production deployments.",
			},
			"expiration_canceled": schema.StringAttribute{
				Computed:    true,
				Description: "The retention period for canceled deployments.",
			},
			"expiration_errored": schema.StringAttribute{
				Computed:    true,
				Description: "The retention period for errored deployments.",
			},
			"project_id": schema.StringAttribute{
				Description: "The ID of the Project for the retention policy",
				Required:    true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the Vercel team.",
			},
		},
	}
}

type ProjectDeploymentRetentionWithID struct {
	ExpirationPreview    types.String `tfsdk:"expiration_preview"`
	ExpirationProduction types.String `tfsdk:"expiration_production"`
	ExpirationCanceled   types.String `tfsdk:"expiration_canceled"`
	ExpirationErrored    types.String `tfsdk:"expiration_errored"`
	ProjectID            types.String `tfsdk:"project_id"`
	TeamID               types.String `tfsdk:"team_id"`
	ID                   types.String `tfsdk:"id"`
}
