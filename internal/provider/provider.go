package provider

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	tfsdk "github.com/apparentlymart/terraform-sdk"
	"github.com/apparentlymart/terraform-sdk/tfschema"
	"github.com/buildkite/go-buildkite/buildkite"
	"github.com/zclconf/go-cty/cty"
)

const timestampFormat = "2006-01-02T15:04:05-0700"

func Provider() *tfsdk.Provider {
	return &tfsdk.Provider{
		ConfigSchema: &tfschema.BlockType{
			Attributes: map[string]*tfschema.Attribute{
				"organization": {Type: cty.String, Optional: true},
			},
		},
		ConfigureFn: configure,

		ManagedResourceTypes: map[string]tfsdk.ManagedResourceType{
			"buildkite_pipeline": pipelineManagedResourceType(),
		},

		DataResourceTypes: map[string]tfsdk.DataResourceType{
			"buildkite_organization": organizationDataResourceType(),
		},
	}
}

func configure(ctx context.Context, config *Config) (*Meta, tfsdk.Diagnostics) {
	var diags tfsdk.Diagnostics

	var orgName string
	if config.Organization != nil {
		orgName = *config.Organization
	} else {
		orgName = os.Getenv("BUILDKITE_ORGANIZATION")
	}
	if orgName == "" {
		diags = diags.Append(tfsdk.Diagnostic{
			Summary: "No Buildkite organization configured",
			Detail:  "The \"organization\" argument is required, unless the BUILDKITE_ORGANIZATION environment variable is set.",
			Path:    cty.GetAttrPath("organization"),
		})
	}

	token := apiToken()
	if token == "" {
		diags = diags.Append(tfsdk.Diagnostic{
			Summary: "No Buildkite API token available",
			Detail:  "Set the BUILDKITE_TOKEN environment variable to your Buildkite API key.",
		})
		return nil, diags
	}

	client, err := newBuildkiteClient(token)
	if err != nil {
		diags = diags.Append(tfsdk.Diagnostic{
			Summary: "Buildkite API client creation failed",
			Detail:  fmt.Sprintf("Failed to initialize the Buildkite API client: %s.", err),
		})
		return nil, diags
	}

	if diags.HasErrors() {
		return nil, diags
	}

	// We'll fetch our organization just to sure it exists and also
	// that the given credentials are valid to work with it.
	org, resp, err := client.Organizations.Get(orgName)
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusNotFound:
			diags = diags.Append(tfsdk.Diagnostic{
				Summary: "Buildkite organization not found",
				Detail:  fmt.Sprintf("Cannot find organization %q. Either the organization does not exist or your current API credentials do not have API access to it.", orgName),
				Path:    cty.GetAttrPath("organization"),
			})
			return nil, diags
		case http.StatusUnauthorized:
			diags = diags.Append(tfsdk.Diagnostic{
				Summary: "Invalid Buildkite API token",
				Detail:  "The Buildkite API rejected the given API token.",
			})
			return nil, diags
		case http.StatusOK:
			// This is fine.
		default:
			diags = diags.Append(tfsdk.Diagnostic{
				Summary: "Failed to retrieve Buildkite organization",
				Detail:  fmt.Sprintf("The Buildkite API returned an unexpected response code: %s.", resp.Status),
			})
			return nil, diags
		}
	}
	if err != nil {
		diags = diags.Append(apiConnectionError(err))
		return nil, diags
	}

	log.Printf("[INFO] Organization %q (%q) has id %q", *org.Slug, *org.Name, *org.ID)

	return &Meta{
		config: config,
		client: client,
		org:    org,
	}, nil
}

type Config struct {
	Organization *string `cty:"organization"`
}

type Meta struct {
	config *Config
	client *buildkite.Client
	org    *buildkite.Organization
}

func apiConnectionError(err error) tfsdk.Diagnostic {
	return tfsdk.Diagnostic{
		Severity: tfsdk.Error,
		Summary:  "Failed to connect to Buildkite REST API",
		Detail:   fmt.Sprintf("The Buildkite REST API is not available: %s.", err),
	}
}

func apiResponseError(status string) tfsdk.Diagnostic {
	return tfsdk.Diagnostic{
		Severity: tfsdk.Error,
		Summary:  "Buildkite API request failed",
		Detail:   fmt.Sprintf("The Buildkite API returned an unexpected response code: %s.", status),
	}
}

func apiWriteErrors(resp *buildkite.Response, err error) tfsdk.Diagnostics {
	var diags tfsdk.Diagnostics
	if resp != nil && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		diags = diags.Append(apiResponseError(resp.Status))
	}
	if err != nil {
		diags = diags.Append(apiConnectionError(err))
	}
	return diags
}
