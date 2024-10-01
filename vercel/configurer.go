package vercel

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/vercel/terraform-provider-vercel/client"
)

type dataSourceConfigurer struct {
	client               *client.Client
	dataSourceNameSuffix string
}

func (d *dataSourceConfigurer) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + d.dataSourceNameSuffix
}

func (d *dataSourceConfigurer) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

type resourceConfigurer struct {
	client             *client.Client
	resourceNameSuffix string
}

func (r *resourceConfigurer) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.resourceNameSuffix
}

func (r *resourceConfigurer) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

type reader[T any] struct {
	client   *client.Client
	readFunc func(context.Context, T, *client.Client, *datasource.ReadResponse) (T, error)
}

func (r *reader[T]) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config T
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// out, err := r.client.GetAlias(ctx, config.Alias.ValueString(), config.TeamID.ValueString())
	result, err := r.readFunc(ctx, config, r.client, resp)
	if err != nil {
		return
	}

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
}
