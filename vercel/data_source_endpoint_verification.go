package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vercel/terraform-provider-vercel/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &endpointVerificationDataSource{}
)

func newEndpointVerificationDataSource() datasource.DataSource {
	return &endpointVerificationDataSource{
		dataSourceConfigurer: &dataSourceConfigurer{
			dataSourceNameSuffix: "_endpoint_verification",
		},
		reader: &reader[EndpointVerification]{
			// readFunc will read the endpoint verification information by requesting it from the Vercel API, and will update Terraform
			// with this information.
			// It is called by the provider whenever data source values should be read to update state.
			readFunc: func(ctx context.Context, config EndpointVerification, c *client.Client, resp *datasource.ReadResponse) (EndpointVerification, error) {
				code, err := c.GetEndpointVerificationCode(ctx, config.TeamID.ValueString())
				if err != nil {
					resp.Diagnostics.AddError(
						"Failed to get verification code",
						fmt.Sprintf("Failed to get verification code, unexpected error: %s", err),
					)
					return EndpointVerification{}, err
				}
				return EndpointVerification{
					TeamID:           config.TeamID,
					ID:               types.StringValue(code),
					VerificationCode: types.StringValue(code),
				}, nil
			},
		},
	}
}

type endpointVerificationDataSource struct {
	*dataSourceConfigurer
	*reader[EndpointVerification]
}

// Schema returns the schema information for a file data source
func (d *endpointVerificationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a verification code that can be used to prove ownership over an API.",
		Attributes: map[string]schema.Attribute{
			"verification_code": schema.StringAttribute{
				Description: "A verification code that should be set in the `x-vercel-verify` response header for your API. This is used to verify that the endpoint belongs to you.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
			"team_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the team the Edge Config should exist under. Required when configuring a team resource if a default team has not been set in the provider.",
			},
		},
	}
}

// FileData represents the information terraform knows about a File data source
type EndpointVerification struct {
	ID               types.String `tfsdk:"id"`
	TeamID           types.String `tfsdk:"team_id"`
	VerificationCode types.String `tfsdk:"verification_code"`
}
