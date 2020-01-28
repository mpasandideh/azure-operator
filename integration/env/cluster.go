package env

import (
	"fmt"
	"os"
)

const (
	EnvVarCommonDomain         = "COMMON_DOMAIN"
	EnvVarIgnitionAdditionPath = "IGNITION_ADDITION_PATH"
	EnvVarVaultToken           = "VAULT_TOKEN"
)

var (
	commonDomain         string
	ignitionAdditionPath string
	vaultToken           string
)

func init() {
	commonDomain = os.Getenv(EnvVarCommonDomain)
	if commonDomain == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarCommonDomain))
	}

	ignitionAdditionPath = os.Getenv(EnvVarIgnitionAdditionPath)
	if ignitionAdditionPath == "" {
		ignitionAdditionPath = "/tmp"
	}

	vaultToken = os.Getenv(EnvVarVaultToken)
	if vaultToken == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarVaultToken))
	}
}

func CommonDomain() string {
	return commonDomain
}

func IgnitionAdditionPath() string {
	return ignitionAdditionPath
}

func VaultToken() string {
	return vaultToken
}
