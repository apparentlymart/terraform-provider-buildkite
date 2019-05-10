package provider

import (
	"context"
	"fmt"
	"net/http"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/buildkite/go-buildkite/buildkite"
	"github.com/zclconf/go-cty/cty"
)

type organizationDRT struct {
	ID           *string `cty:"id"`
	URL          *string `cty:"url"`
	WebURL       *string `cty:"web_url"`
	Name         *string `cty:"name"`
	Slug         *string `cty:"slug"`
	Repository   *string `cty:"repository"`
	PipelinesURL *string `cty:"pipelines_url"`
	AgentsURL    *string `cty:"agents_url"`
	CreatedTime  *string `cty:"created_time"`
}

func organizationDataResourceType() tfsdk.DataResourceType {
	return tfsdk.NewDataResourceType(&tfsdk.ResourceTypeDef{
		ConfigSchema: &tfschema.BlockType{
			Attributes: map[string]*tfschema.Attribute{
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
				"name": {
					Type:     cty.String,
					Computed: true,
				},
				"repository": {
					Type:     cty.String,
					Computed: true,
				},
				"pipelines_url": {
					Type:     cty.String,
					Computed: true,
				},
				"agents_url": {
					Type:     cty.String,
					Computed: true,
				},
				"created_time": {
					Type:     cty.String,
					Computed: true,
				},
			},
		},

		ReadFn: func(ctx context.Context, meta *Meta, obj *organizationDRT) (*organizationDRT, tfsdk.Diagnostics) {
			var diags tfsdk.Diagnostics

			orgSlug := *meta.org.Slug
			if obj.Slug != nil {
				orgSlug = *obj.Slug
			}

			var apiOrg *buildkite.Organization
			switch {
			case orgSlug == *meta.org.Slug:
				// Easy! We already loaded this during configuration.
				apiOrg = meta.org
			default:
				org, resp, err := meta.client.Organizations.Get(orgSlug)
				if resp != nil {
					switch resp.StatusCode {
					case http.StatusNotFound:
						diags = diags.Append(tfsdk.Diagnostic{
							Severity: tfsdk.Error,
							Summary:  "Buildkite organization not found",
							Detail:   fmt.Sprintf("Cannot find organization %q. Either the organization does not exist or your current API credentials do not have API access to it.", orgSlug),
							Path:     cty.GetAttrPath("slug"),
						})
						return obj, diags
					case http.StatusOK:
						apiOrg = org
					default:
						diags = diags.Append(apiResponseError(resp.Status))
						return obj, diags
					}
				}
				if err != nil {
					diags = diags.Append(apiConnectionError(err))
					return obj, diags
				}
			}

			createdTime := apiOrg.CreatedAt.Format(timestampFormat)

			obj.ID = apiOrg.ID
			obj.URL = apiOrg.URL
			obj.WebURL = apiOrg.WebURL
			obj.Name = apiOrg.Name
			obj.Slug = apiOrg.Slug
			obj.Repository = apiOrg.Repository
			obj.PipelinesURL = apiOrg.PipelinesURL
			obj.AgentsURL = apiOrg.AgentsURL
			obj.CreatedTime = &createdTime

			return obj, diags
		},
	})
}
