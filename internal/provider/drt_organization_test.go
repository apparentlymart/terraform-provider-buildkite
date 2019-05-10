package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/apparentlymart/terraform-sdk/tftest"
)

func TestDRTOrganization(t *testing.T) {
	tftest.AcceptanceTest(t)

	t.Run("implied current", func(t *testing.T) {
		wd := testHelper.RequireNewWorkingDir(t)
		defer wd.Close()

		wd.RequireSetConfig(t, `
data "buildkite_organization" "current" {}
`)

		wd.RequireInit(t)
		wd.RequireApply(t)

		// TODO: Check the state, once the tftest package allows that.
	})
	t.Run("explicit current", func(t *testing.T) {
		wd := testHelper.RequireNewWorkingDir(t)
		defer wd.Close()

		orgSlug := os.Getenv("BUILDKITE_ORGANIZATION")

		wd.RequireSetConfig(t, fmt.Sprintf(`
data "buildkite_organization" "current" {
	slug = %q
}
`, orgSlug))

		wd.RequireInit(t)
		wd.RequireApply(t)
	})
	t.Run("non-existent", func(t *testing.T) {
		wd := testHelper.RequireNewWorkingDir(t)
		defer wd.Close()

		wd.RequireSetConfig(t, `
data "buildkite_organization" "current" {
	slug = "xyz-does-not-exist"
}
`)

		wd.RequireInit(t)
		err := wd.Apply()
		if err == nil {
			t.Fatalf("apply succeeded; want error")
		}
		if got, want := err.Error(), "Buildkite organization not found"; !strings.Contains(got, want) {
			t.Errorf("wrong error\ngot:\n%s\nwant: %s", got, want)
		}
	})
}
