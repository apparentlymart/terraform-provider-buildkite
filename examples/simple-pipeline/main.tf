provider "buildkite" {
  # Enter your organization name here
  #organization = "example-org"

  # Credentials are loaded from the same sources as the Buildkite CLI. For example,
  # you can set an API token in the environment variable BUILDKITE_TOKEN.
}

resource "buildkite_pipeline" "example" {
  name       = "Example from Terraform Provider"
  repository = "git://github.com/apparentlymart/terraform-provider-buildkite.git"

  step {
    type = "script"

    command = "echo 'HELLO WORLD!!'"
  }
}
