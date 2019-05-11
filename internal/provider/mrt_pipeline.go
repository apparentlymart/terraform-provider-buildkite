package provider

import (
	"context"
	"fmt"
	"net/http"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tfobj"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/buildkite/go-buildkite/buildkite"
	"github.com/zclconf/go-cty/cty"
)

type pipelineMRT struct {
	ID          *string `cty:"id"`
	URL         *string `cty:"url"`
	WebURL      *string `cty:"web_url"`
	Name        string  `cty:"name"`
	Slug        *string `cty:"slug"`
	Repository  string  `cty:"repository"`
	BuildsURL   *string `cty:"builds_url"`
	BadgeURL    *string `cty:"badge_url"`
	CreatedTime *string `cty:"created_time"`

	// TODO: VCS-provider-specific settings.

	Steps []pipelineMRTStep `cty:"step"`

	Organization *string `cty:"organization"`
}

type pipelineMRTStep struct {
	Type  string  `cty:"type"`
	Label *string `cty:"label"`

	// For "script" steps only
	Command         *string            `cty:"command"`
	Env             *map[string]string `cty:"env"`
	AgentQueryRules *[]string          `cty:"agent_query_rules"`

	// TODO: All of the other supported attributes
}

func pipelineManagedResourceType() tfsdk.ManagedResourceType {
	return tfsdk.NewManagedResourceType(&tfsdk.ResourceTypeDef{
		ConfigSchema: &tfschema.BlockType{
			Attributes: map[string]*tfschema.Attribute{
				"name": {
					Type:     cty.String,
					Required: true,
				},
				"repository": {
					Type:     cty.String,
					Required: true,
				},

				"slug": {
					Type:        cty.String,
					Optional:    true,
					Computed:    true,
					Description: "Slug of the organization to retrieve. If not specified, then the organization slug configured in the provider is used.",

					ValidateFn: func(val string) tfsdk.Diagnostics {
						var diags tfsdk.Diagnostics
						if val == "" {
							diags = diags.Append(tfsdk.ValidationError(
								fmt.Errorf("an organization slug must not be empty"),
							))
						}
						return diags
					},
				},

				"id": {
					Type:     cty.String,
					Computed: true,
				},
				"url": {
					Type:     cty.String,
					Computed: true,
				},
				"web_url": {
					Type:     cty.String,
					Computed: true,
				},
				"builds_url": {
					Type:     cty.String,
					Computed: true,
				},
				"badge_url": {
					Type:     cty.String,
					Computed: true,
				},
				"created_time": {
					Type:     cty.String,
					Computed: true,
				},
				"organization": {
					Type:     cty.String,
					Computed: true,
				},
			},
			NestedBlockTypes: map[string]*tfschema.NestedBlockType{
				"step": {
					Nesting: tfschema.NestingList,
					Content: tfschema.BlockType{
						Attributes: map[string]*tfschema.Attribute{
							"type": {
								Type:     cty.String,
								Required: true,
							},
							"label": {
								Type:     cty.String,
								Optional: true,
							},

							// For "script" steps only
							"command": {
								Type:     cty.String,
								Optional: true,
							},
							"env": {
								Type:     cty.Map(cty.String),
								Optional: true,
							},
							"agent_query_rules": {
								Type:     cty.Set(cty.String),
								Optional: true,
							},
						},
					},
				},
			},
		},
		PlanFn: func(ctx context.Context, meta *Meta, plan tfobj.PlanBuilder) (cty.Value, cty.PathSet, tfsdk.Diagnostics) {
			var diags tfsdk.Diagnostics

			moreDiags := validateStepBlocks(plan.BlockList("step"))
			diags = diags.Append(moreDiags.UnderPath(cty.GetAttrPath("step")))

			newOrgSlug := cty.StringVal(*meta.org.Slug)
			plan.SetAttr("organization", newOrgSlug)
			if plan.Action() != tfobj.Create && plan.AttrHasChange("organization") {
				plan.SetAttrRequiresReplacement("organization")
			}

			return plan.ObjectVal(), plan.RequiresReplace(), diags
		},

		CreateFn: func(ctx context.Context, meta *Meta, obj *pipelineMRT) (*pipelineMRT, tfsdk.Diagnostics) {
			var diags tfsdk.Diagnostics

			pipeline := buildAPICreatePipelineFromMRT(obj)
			created, resp, err := meta.client.Pipelines.Create(*meta.org.Slug, pipeline)
			diags = diags.Append(apiWriteErrors(resp, err))
			if diags.HasErrors() {
				return obj, diags
			}

			return buildMRTPipelineFromAPI(created, meta.org), diags
		},

		ReadFn: func(ctx context.Context, meta *Meta, obj *pipelineMRT) (*pipelineMRT, tfsdk.Diagnostics) {
			var diags tfsdk.Diagnostics

			read, resp, err := meta.client.Pipelines.Get(*obj.Organization, *obj.Slug)
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, diags
			}
			diags = diags.Append(apiWriteErrors(resp, err))
			if diags.HasErrors() {
				return obj, diags
			}

			return buildMRTPipelineFromAPI(read, meta.org), diags
		},

		UpdateFn: func(ctx context.Context, meta *Meta, prior, new *pipelineMRT) (*pipelineMRT, tfsdk.Diagnostics) {
			var diags tfsdk.Diagnostics

			diags = diags.Append(tfsdk.Diagnostic{
				Severity: tfsdk.Error,
				Summary:  "Update not implemented",
				Detail:   "Updating is not yet implemented for buildkite_pipeline.",
			})

			return new, diags
		},

		DeleteFn: func(ctx context.Context, meta *Meta, obj *pipelineMRT) (*pipelineMRT, tfsdk.Diagnostics) {
			var diags tfsdk.Diagnostics

			resp, err := meta.client.Pipelines.Delete(*obj.Organization, *obj.Slug)
			diags = diags.Append(apiWriteErrors(resp, err))
			if diags.HasErrors() {
				return obj, diags
			}

			return nil, diags
		},
	})
}

func buildAPICreatePipelineFromMRT(obj *pipelineMRT) *buildkite.CreatePipeline {
	ret := &buildkite.CreatePipeline{
		Name:       obj.Name,
		Repository: obj.Repository,
		Steps:      make([]buildkite.Step, 0, len(obj.Steps)),
	}

	for _, stepObj := range obj.Steps {
		step := buildkite.Step{
			Type:    &stepObj.Type,
			Name:    stepObj.Label,
			Command: stepObj.Command,
		}

		if stepObj.Env != nil {
			step.Env = *stepObj.Env
		}

		ret.Steps = append(ret.Steps, step)
	}

	return ret
}

func buildMRTPipelineFromAPI(pipeline *buildkite.Pipeline, org *buildkite.Organization) *pipelineMRT {
	ret := &pipelineMRT{
		ID:         pipeline.ID,
		URL:        pipeline.URL,
		WebURL:     pipeline.WebURL,
		Name:       *pipeline.Name,
		Slug:       pipeline.Slug,
		Repository: *pipeline.Repository,
		BuildsURL:  pipeline.BuildsURL,
		BadgeURL:   pipeline.BadgeURL,
		Steps:      make([]pipelineMRTStep, 0, len(pipeline.Steps)),

		Organization: org.Slug,
	}
	createdTime := pipeline.CreatedAt.Format(timestampFormat)
	ret.CreatedTime = &createdTime

	for _, step := range pipeline.Steps {
		step := pipelineMRTStep{
			Type:    *step.Type,
			Label:   step.Name,
			Command: step.Command,
		}

		// TODO: Env and AgentQueryRules

		ret.Steps = append(ret.Steps, step)
	}
	return ret
}

func validateStepBlocks(readers []tfobj.ObjectReader) tfsdk.Diagnostics {
	var diags tfsdk.Diagnostics

	if len(readers) == 0 {
		diags = diags.Append(tfsdk.Diagnostic{
			Severity: tfsdk.Error,
			Summary:  "No Buildkite pipeline steps",
			Detail:   "A Buildkite pipeline must have at least one \"step\" block.",
		})
	}

	for i, reader := range readers {
		moreDiags := validateStepBlock(reader)
		diags = diags.Append(moreDiags.UnderPath(cty.IndexPath(cty.NumberIntVal(int64(i)))))
	}

	return diags
}

func validateStepBlock(reader tfobj.ObjectReader) tfsdk.Diagnostics {
	var diags tfsdk.Diagnostics

	stepTypeVal := reader.Attr("type")
	if !stepTypeVal.IsKnown() {
		// Can't validate at all yet, then
		return diags
	}
	switch stepType := stepTypeVal.AsString(); stepType {

	case "script":
		if reader.Attr("command").IsNull() {
			diags = diags.Append(tfsdk.ValidationError(fmt.Errorf("\"command\" argument is required for %q steps", stepType)))
		}

	case "trigger":
		if !reader.Attr("command").IsNull() {
			diags = diags.Append(tfsdk.ValidationError(
				cty.GetAttrPath("command").NewErrorf("\"command\" is not used for %q steps", stepType),
			))
		}

	case "manual":
		if !reader.Attr("command").IsNull() {
			diags = diags.Append(tfsdk.ValidationError(
				cty.GetAttrPath("command").NewErrorf("\"command\" is not used for %q steps", stepType),
			))
		}

	case "waiter":
		if !reader.Attr("command").IsNull() {
			diags = diags.Append(tfsdk.ValidationError(
				cty.GetAttrPath("command").NewErrorf("\"command\" is not used for %q steps", stepType),
			))
		}

	case "":
		diags = diags.Append(tfsdk.ValidationError(
			cty.GetAttrPath("type").NewErrorf("empty string is not a valid step type"),
		))
	default:
		diags = diags.Append(tfsdk.ValidationError(
			cty.GetAttrPath("type").NewErrorf("%q is not a valid step type", stepType),
		))
	}

	return diags
}
