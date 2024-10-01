package vercel

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &fileDataSource{}
)

func newFileDataSource() datasource.DataSource {
	return &fileDataSource{
		dataSourceConfigurer: &dataSourceConfigurer{
			dataSourceNameSuffix: "_file",
		},
	}
}

type fileDataSource struct {
	*dataSourceConfigurer
}

// Schema returns the schema information for a file data source
func (d *fileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides information about a file on disk.

This will read a single file, providing metadata for use with a ` + "`vercel_deployment`.",
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "The path to the file on your filesystem. Note that the path is relative to the root of the terraform files.",
				Required:    true,
			},
			"file": schema.MapAttribute{
				Description: "A map of filename to metadata about the file. The metadata contains the file size and hash, and allows a deployment to be created if the file changes.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// FileData represents the information terraform knows about a File data source
type FileData struct {
	Path types.String      `tfsdk:"path"`
	ID   types.String      `tfsdk:"id"`
	File map[string]string `tfsdk:"file"`
}

// Read will read a file from the filesytem and provide terraform with information about it.
// It is called by the provider whenever data source values should be read to update state.
func (d *fileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FileData
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := os.ReadFile(config.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading file",
			fmt.Sprintf("Could not read file %s, unexpected error: %s",
				config.Path.ValueString(),
				err,
			),
		)
		return
	}

	rawSha := sha1.Sum(content)
	sha := hex.EncodeToString(rawSha[:])
	config.File = map[string]string{
		config.Path.ValueString(): fmt.Sprintf("%d~%s", len(content), sha),
	}
	config.ID = config.Path

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
