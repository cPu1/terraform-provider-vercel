package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSourceWithConfigure = &deploymentDataSource{}
)

func newDeploymentDataSource() datasource.DataSource {
	return &deploymentDataSource{
		dataSourceConfigurer: &dataSourceConfigurer{
			dataSourceNameSuffix: "_deployment",
		},
		reader: &reader[DeploymentDataSource]{
			// readFunc will read the deployment information by requesting it from the Vercel API, and will update terraform
			// with this information.
			// It is called by the provider whenever data source values should be read to update state.
			readFunc: func(ctx context.Context, config DeploymentDataSource, c *client.Client, resp *datasource.ReadResponse) (DeploymentDataSource, error) {
				out, err := c.GetDeployment(ctx, config.ID.ValueString(), config.TeamID.ValueString())
				if client.NotFound(err) {
					resp.State.RemoveResource(ctx)
					return DeploymentDataSource{}, err
				}
				if err != nil {
					resp.Diagnostics.AddError(
						"Error reading deployment",
						fmt.Sprintf("Could not get deployment %s %s, unexpected error: %s",
							config.TeamID.ValueString(),
							config.ID.ValueString(),
							err,
						),
					)
					return DeploymentDataSource{}, err
				}

				result := convertResponseToDeploymentDataSource(out)
				tflog.Info(ctx, "read deployment", map[string]interface{}{
					"team_id":    result.TeamID.ValueString(),
					"project_id": result.ID.ValueString(),
				})
				return result, nil
			},
		},
	}
}

type deploymentDataSource struct {
	*dataSourceConfigurer
	*reader[DeploymentDataSource]
}

// Schema returns the schema information for an deployment data source
func (r *deploymentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about an existing Deployment.

A Deployment is the result of building your Project and making it available through a live URL.
`,
		Attributes: map[string]schema.Attribute{
			"team_id": schema.StringAttribute{
				Description: "The Team ID to the Deployment belong to. Required when reading a team resource if a default team has not been set in the provider.",
				Optional:    true,
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID or URL of the Deployment to read.",
			},
			"domains": schema.ListAttribute{
				Description: "A list of all the domains (default domains, staging domains and production domains) that were assigned upon deployment creation.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"project_id": schema.StringAttribute{
				Description: "The project ID to add the deployment to.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "A unique URL that is automatically generated for a deployment.",
				Computed:    true,
			},
			"production": schema.BoolAttribute{
				Description: "true if the deployment is a production deployment, meaning production aliases will be assigned.",
				Computed:    true,
			},
			"ref": schema.StringAttribute{
				Description: "The branch or commit hash that has been deployed. Note this will only work if the project is configured to use a Git repository.",
				Computed:    true,
			},
		},
	}
}

type DeploymentDataSource struct {
	Domains    types.List   `tfsdk:"domains"`
	ID         types.String `tfsdk:"id"`
	Production types.Bool   `tfsdk:"production"`
	ProjectID  types.String `tfsdk:"project_id"`
	TeamID     types.String `tfsdk:"team_id"`
	URL        types.String `tfsdk:"url"`
	Ref        types.String `tfsdk:"ref"`
}

func convertResponseToDeploymentDataSource(in client.DeploymentResponse) DeploymentDataSource {
	ref := types.StringNull()
	if in.GitSource.Ref != "" {
		ref = types.StringValue(in.GitSource.Ref)
	}

	var domains []attr.Value
	for _, a := range in.Aliases {
		domains = append(domains, types.StringValue(a))
	}
	return DeploymentDataSource{
		Domains:    types.ListValueMust(types.StringType, domains),
		Production: types.BoolValue(in.Target != nil && *in.Target == "production"),
		TeamID:     toTeamID(in.TeamID),
		ProjectID:  types.StringValue(in.ProjectID),
		ID:         types.StringValue(in.ID),
		URL:        types.StringValue(in.URL),
		Ref:        ref,
	}
}
