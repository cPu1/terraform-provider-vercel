package vercel

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/vercel/terraform-provider-vercel/client"
)

var (
	_ resource.ResourceWithConfigure   = &sharedEnvironmentVariableResource{}
	_ resource.ResourceWithImportState = &sharedEnvironmentVariableResource{}
	_ resource.ResourceWithModifyPlan  = &sharedEnvironmentVariableResource{}
)

func newSharedEnvironmentVariableResource() resource.Resource {
	return &sharedEnvironmentVariableResource{
		resourceConfigurer: &resourceConfigurer{
			resourceNameSuffix: "_shared_environment_variable",
		},
	}
}

type sharedEnvironmentVariableResource struct {
	*resourceConfigurer
}

func (r *sharedEnvironmentVariableResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var config SharedEnvironmentVariable
	diags := req.Plan.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ID.ValueString() != "" {
		// The resource already exists, so this is okay.
		return
	}
	if config.Sensitive.IsUnknown() || config.Sensitive.IsNull() || config.Sensitive.ValueBool() {
		// Sensitive is either true, or computed, which is fine.
		return
	}

	// if sensitive is explicitly set to `false`, then validate that an env var can be created with the given
	// team sensitive environment variable policy.
	team, err := r.client.Team(ctx, config.TeamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error validating shared environment variable",
			"Could not validate shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	if team.SensitiveEnvironmentVariablePolicy == nil || *team.SensitiveEnvironmentVariablePolicy != "on" {
		// the policy isn't enabled
		return
	}

	resp.Diagnostics.AddAttributeError(
		path.Root("sensitive"),
		"Shared Environment Variable Invalid",
		"This team has a policy that forces all environment variables to be sensitive. Please remove the `sensitive` field or set the `sensitive` field to `true` in your configuration.",
	)
}

// Schema returns the schema information for a shared environment variable resource.
func (r *sharedEnvironmentVariableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `
Provides a Shared Environment Variable resource.

A Shared Environment Variable resource defines an Environment Variable that can be shared between multiple Vercel Projects.

For more detailed information, please see the [Vercel documentation](https://vercel.com/docs/concepts/projects/environment-variables/shared-environment-variables).
`,
		Attributes: map[string]schema.Attribute{
			"target": schema.SetAttribute{
				Required:    true,
				Description: "The environments that the Environment Variable should be present on. Valid targets are either `production`, `preview`, or `development`.",
				ElementType: types.StringType,
				Validators: []validator.Set{
					stringSetItemsIn("production", "preview", "development"),
					stringSetMinCount(1),
				},
			},
			"key": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The name of the Environment Variable.",
			},
			"value": schema.StringAttribute{
				Required:    true,
				Description: "The value of the Environment Variable.",
				Sensitive:   true,
			},
			"project_ids": schema.SetAttribute{
				Required:    true,
				Description: "The ID of the Vercel project.",
				ElementType: types.StringType,
			},
			"team_id": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The ID of the Vercel team. Shared environment variables require a team.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured(), stringplanmodifier.UseStateForUnknown()},
			},
			"id": schema.StringAttribute{
				Description:   "The ID of the Environment Variable.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Computed:      true,
			},
			"sensitive": schema.BoolAttribute{
				Description:   "Whether the Environment Variable is sensitive or not. (May be affected by a [team-wide environment variable policy](https://vercel.com/docs/projects/environment-variables/sensitive-environment-variables#environment-variables-policy))",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
		},
	}
}

// SharedEnvironmentVariable reflects the state terraform stores internally for a project environment variable.
type SharedEnvironmentVariable struct {
	Target     types.Set    `tfsdk:"target"`
	Key        types.String `tfsdk:"key"`
	Value      types.String `tfsdk:"value"`
	TeamID     types.String `tfsdk:"team_id"`
	ProjectIDs types.Set    `tfsdk:"project_ids"`
	ID         types.String `tfsdk:"id"`
	Sensitive  types.Bool   `tfsdk:"sensitive"`
}

func (e *SharedEnvironmentVariable) toCreateSharedEnvironmentVariableRequest(ctx context.Context, diags diag.Diagnostics) (req client.CreateSharedEnvironmentVariableRequest, ok bool) {
	var target []string
	ds := e.Target.ElementsAs(ctx, &target, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
	}

	var projectIDs []string
	ds = e.ProjectIDs.ElementsAs(ctx, &projectIDs, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
	}

	var envVariableType string

	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}

	return client.CreateSharedEnvironmentVariableRequest{
		EnvironmentVariable: client.SharedEnvironmentVariableRequest{
			Target:     target,
			Type:       envVariableType,
			ProjectIDs: projectIDs,
			EnvironmentVariables: []client.SharedEnvVarRequest{
				{
					Key:   e.Key.ValueString(),
					Value: e.Value.ValueString(),
				},
			},
		},
		TeamID: e.TeamID.ValueString(),
	}, true
}

func (e *SharedEnvironmentVariable) toUpdateSharedEnvironmentVariableRequest(ctx context.Context, diags diag.Diagnostics) (req client.UpdateSharedEnvironmentVariableRequest, ok bool) {
	var target []string
	ds := e.Target.ElementsAs(ctx, &target, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
	}

	var projectIDs []string
	ds = e.ProjectIDs.ElementsAs(ctx, &projectIDs, false)
	diags = append(diags, ds...)
	if diags.HasError() {
		return req, false
	}
	var envVariableType string

	if e.Sensitive.ValueBool() {
		envVariableType = "sensitive"
	} else {
		envVariableType = "encrypted"
	}
	return client.UpdateSharedEnvironmentVariableRequest{
		Value:      e.Value.ValueString(),
		Target:     target,
		Type:       envVariableType,
		TeamID:     e.TeamID.ValueString(),
		EnvID:      e.ID.ValueString(),
		ProjectIDs: projectIDs,
	}, true
}

// convertResponseToSharedEnvironmentVariable is used to populate terraform state based on an API response.
// Where possible, values from the API response are used to populate state. If not possible,
// values from plan are used.
func convertResponseToSharedEnvironmentVariable(response client.SharedEnvironmentVariableResponse, v types.String) SharedEnvironmentVariable {
	target := []attr.Value{}
	for _, t := range response.Target {
		target = append(target, types.StringValue(t))
	}

	projectIDs := []attr.Value{}
	for _, t := range response.ProjectIDs {
		projectIDs = append(projectIDs, types.StringValue(t))
	}

	value := types.StringValue(response.Value)
	if response.Type == "sensitive" {
		value = v
	}

	return SharedEnvironmentVariable{
		Target:     types.SetValueMust(types.StringType, target),
		Key:        types.StringValue(response.Key),
		Value:      value,
		ProjectIDs: types.SetValueMust(types.StringType, projectIDs),
		TeamID:     toTeamID(response.TeamID),
		ID:         types.StringValue(response.ID),
		Sensitive:  types.BoolValue(response.Type == "sensitive"),
	}
}

// Create will create a new shared environment variable.
// This is called automatically by the provider when a new resource should be created.
func (r *sharedEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SharedEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request, ok := plan.toCreateSharedEnvironmentVariableRequest(ctx, resp.Diagnostics)
	if !ok {
		return
	}
	response, err := r.client.CreateSharedEnvironmentVariable(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating shared environment variable",
			"Could not create shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(response, plan.Value)

	tflog.Info(ctx, "created shared environment variable", map[string]interface{}{
		"id":      result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read will read an shared environment variable by requesting it from the Vercel API, and will update terraform
// with this information.
func (r *sharedEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SharedEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetSharedEnvironmentVariable(ctx, state.TeamID.ValueString(), state.ID.ValueString())
	if client.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading shared environment variable",
			fmt.Sprintf("Could not get shared environment variable %s %s, unexpected error: %s",
				state.TeamID.ValueString(),
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(out, state.Value)
	tflog.Info(ctx, "read shared environment variable", map[string]interface{}{
		"id":      result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the shared environment variable of a Vercel project state.
func (r *sharedEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SharedEnvironmentVariable
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	request, ok := plan.toUpdateSharedEnvironmentVariableRequest(ctx, resp.Diagnostics)
	if !ok {
		return
	}
	response, err := r.client.UpdateSharedEnvironmentVariable(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating shared environment variable",
			"Could not update shared environment variable, unexpected error: "+err.Error(),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(response, plan.Value)

	tflog.Info(ctx, "updated shared environment variable", map[string]interface{}{
		"id":      result.ID.ValueString(),
		"team_id": result.TeamID.ValueString(),
	})

	diags = resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes a Vercel shared environment variable.
func (r *sharedEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SharedEnvironmentVariable
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSharedEnvironmentVariable(ctx, state.TeamID.ValueString(), state.ID.ValueString())
	if client.NotFound(err) {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting shared environment variable",
			fmt.Sprintf(
				"Could not delete shared environment variable %s, unexpected error: %s",
				state.ID.ValueString(),
				err,
			),
		)
		return
	}

	tflog.Info(ctx, "deleted shared environment variable", map[string]interface{}{
		"id":      state.ID.ValueString(),
		"team_id": state.TeamID.ValueString(),
	})
}

// splitID is a helper function for splitting an import ID into the corresponding parts.
// It also validates whether the ID is in a correct format.
func splitSharedEnvironmentVariableID(id string) (teamID, envID string, ok bool) {
	attributes := strings.Split(id, "/")
	if len(attributes) == 2 {
		return attributes[0], attributes[1], true
	}

	return "", "", false
}

// ImportState takes an identifier and reads all the shared environment variable information from the Vercel API.
// The results are then stored in terraform state.
func (r *sharedEnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamID, envID, ok := splitSharedEnvironmentVariableID(req.ID)
	if !ok {
		resp.Diagnostics.AddError(
			"Error importing shared environment variable",
			fmt.Sprintf("Invalid id '%s' specified. should be in format \"team_id/env_id\"", req.ID),
		)
	}

	out, err := r.client.GetSharedEnvironmentVariable(ctx, teamID, envID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading shared environment variable",
			fmt.Sprintf("Could not get shared environment variable %s %s, unexpected error: %s",
				teamID,
				envID,
				err,
			),
		)
		return
	}

	result := convertResponseToSharedEnvironmentVariable(out, types.StringNull())
	tflog.Info(ctx, "imported shared environment variable", map[string]interface{}{
		"team_id": result.TeamID.ValueString(),
		"env_id":  result.ID.ValueString(),
	})

	diags := resp.State.Set(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
