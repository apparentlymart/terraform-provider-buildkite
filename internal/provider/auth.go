package provider

import (
	"encoding/json"
	"log"
	"os"

	"github.com/99designs/keyring"
)

// apiToken follows the same process as the Buildkite CLI to try to find suitable
// credentials to use for requests to the Buildkite API.
func apiToken() string {
	if apiKey := os.Getenv("BUILDKITE_TOKEN"); apiKey != "" {
		log.Println("[INFO] Using API token from BUILDKITE_TOKEN environment variable")
		return apiKey
	}

	keyringBackend := os.Getenv("BUILDKITE_CLI_KEYRING_BACKEND")
	keyringFileDir := os.Getenv("BUILDKITE_CLI_KEYRING_FILE_DIR")
	if keyringFileDir == "" {
		keyringFileDir = "~/.buildkite/keyring/"
	}
	keyringKeychain := os.Getenv("BUILDKITE_CLI_KEYRING_KEYCHAIN")
	if keyringBackend == `keychain` && keyringKeychain == `` {
		keyringKeychain = `login`
	}
	keyringPassDir := os.Getenv("BUILDKITE_CLI_KEYRING_PASS_DIR")
	keyringPassCmd := os.Getenv("BUILDKITE_CLI_KEYRING_PASS_CMD")
	keyringPassPrefix := os.Getenv("BUILDKITE_CLI_KEYRING_PASS_PREFIX")

	var allowedBackends []keyring.BackendType
	if keyringBackend != `` {
		allowedBackends = append(allowedBackends, keyring.BackendType(keyringBackend))
	} else {
		// Otherwise we'll just try everything, except a couple that the Buildkite
		// CLI skips too.
		for _, k := range keyring.AvailableBackends() {
			switch k {
			case keyring.KWalletBackend, keyring.SecretServiceBackend:
				// Buildkite CLI skips these, so we'll follow suit
				continue
			default:
				allowedBackends = append(allowedBackends, k)
			}
		}
	}

	kr, err := keyring.Open(keyring.Config{
		ServiceName:              "buildkite",
		AllowedBackends:          allowedBackends,
		KeychainName:             keyringKeychain,
		FileDir:                  keyringFileDir,
		PassDir:                  keyringPassDir,
		PassCmd:                  keyringPassCmd,
		PassPrefix:               keyringPassPrefix,
		LibSecretCollectionName:  "buildkite",
		KWalletAppID:             "buildkite",
		KWalletFolder:            "buildkite",
		KeychainTrustApplication: true,
	})
	if err != nil {
		log.Printf("[WARN] Failed to open keychain: %s", err)
		return ""
	}

	item, err := kr.Get("graphql-token")
	if err != nil {
		log.Printf("[WARN] Failed to fetch API token from keychain: %s", err)
		return ""
	}

	var token string
	err = json.Unmarshal(item.Data, &token)
	if err != nil {
		log.Printf("[WARN] Failed to decode API token from keychain: %s", err)
		return ""
	}

	return token
}
