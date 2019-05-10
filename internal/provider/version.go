package provider

import "fmt"

var versionBase = "0.0.0"
var versionPrerelease = "dev"
var gitCommit = ""

func version() string {
	if versionPrerelease != "" {
		return fmt.Sprintf("%s-%s", versionBase, versionPrerelease)
	}
	return versionBase
}
