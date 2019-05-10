package main

import (
	"github.com/apparentlymart/terraform-provider-buildkite/internal/provider"
	tfsdk "github.com/apparentlymart/terraform-sdk"
)

func main() {
	tfsdk.ServeProviderPlugin(provider.Provider())
}
