package provider

import (
	"testing"

	"github.com/apparentlymart/terraform-sdk/tftest"
)

func TestMRTPipeline(t *testing.T) {
	tftest.AcceptanceTest(t)

	t.Run("basic", func(t *testing.T) {
		wd := testHelper.RequireNewWorkingDir(t)
		defer func() {
			wd.RequireSetConfig(t, `// empty for destroy`)
			wd.RequireApply(t)
		}()
		defer wd.Close()

		wd.RequireSetConfig(t, `
resource "buildkite_pipeline" "test" {
	name = "foo"
	repository = "git://github.com/apparentlymart/terraform-sdk.git"

	step {
		type = "waiter"
	}
}
`)

		wd.RequireInit(t)
		wd.RequireApply(t)

		// TODO: Check the state, once the tftest package allows that.
	})
}
