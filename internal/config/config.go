package config

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

func GetLocalManifestPath() string {
	return path.Join(xdg.DataHome, "devcleaner", "manifest.json")
}

type RuntimeConfig struct {
	ManifestUrl string
	ManifestTtl time.Duration
	LogLevel    string
}

var Runtime = RuntimeConfig{
	ManifestUrl: defaultManifestUrl,
	ManifestTtl: defaultLocalManifestTTL,
	LogLevel:    defaultLogLevel,
}

const defaultLogLevel = "INFO"
const defaultManifestUrl = "https://sao.gaetans.dev/manifest.json"
const defaultLocalManifestTTL = time.Hour * 24
const LogLevelEnvKey = "DEVCLEANER_LOGLEVEL"
const ManifestUrlEnvKey = "DEVCLEANER_MANIFEST_URL"
const ManifestTtlEnvKey = "DEVCLEANER_MANIFEST_TTL"

const ansiRed = "\033[31m"
const ansiReset = "\033[0m"

func invalidConfigError(name string, value string) {
	fmt.Printf("%sInvalid %s: %s%s\n", ansiRed, name, value, ansiReset)
	os.Exit(1)
}

func init() {
	environ := os.Environ()
	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		if parts[1] == "" {
			continue
		}

		if parts[0] == ManifestUrlEnvKey {
			Runtime.ManifestUrl = parts[1]
		} else if parts[0] == ManifestTtlEnvKey {
			ttl, err := time.ParseDuration(parts[1])
			if err != nil {
				invalidConfigError("manifest ttl", parts[1])
			}
			Runtime.ManifestTtl = ttl
		} else if parts[0] == LogLevelEnvKey {
			Runtime.LogLevel = parts[1]
		}
	}
}
