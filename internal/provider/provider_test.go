package provider

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/apparentlymart/terraform-sdk/tftest"
)

var testHelper *tftest.Helper

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Verbose() {
		// if we're verbose, use the logging requested by TF_LOG
		log.SetOutput(os.Stderr)
	} else {
		// otherwise silence all logs
		log.SetOutput(ioutil.Discard)
	}

	problems := 0
	if token := os.Getenv("BUILDKITE_TOKEN"); token == "" {
		fmt.Fprintln(os.Stderr, "BUILDKITE_TOKEN environment variable must be set")
		problems++
	}
	if orgName := os.Getenv("BUILDKITE_ORGANIZATION"); orgName == "" {
		fmt.Fprintln(os.Stderr, "BUILDKITE_ORGANIZATION environment variable must be set")
		problems++
	}
	if problems > 0 {
		os.Exit(1)
	}

	testHelper = tftest.InitProvider("buildkite", Provider())
	status := m.Run()
	testHelper.Close()
	os.Exit(status)
}

func TestConfigure(t *testing.T) {
	wd := testHelper.RequireNewWorkingDir(t)
	defer wd.Close()

	wd.RequireSetConfig(t, `provider "buildkite" {}`)
	wd.RequireInit(t)
	wd.RequireApply(t)
}
