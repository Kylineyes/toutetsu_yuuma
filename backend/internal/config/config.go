package config

import (
	"os"
	"strconv"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port                     string
	DataDir                  string
	DefaultScreenshotInterval int   // seconds
	MaxFrames                int   // max frames per task
	Timezone                 string
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getint(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// Load reads configuration from the environment with sane defaults.
func Load() *Config {
	return &Config{
		Port:                      getenv("PORT", "8080"),
		DataDir:                   getenv("DATA_DIR", "./data"),
		DefaultScreenshotInterval: getint("DEFAULT_SCREENSHOT_INTERVAL", 60),
		MaxFrames:                 getint("MAX_FRAMES", 120),
		Timezone:                  getenv("TZ", "Asia/Shanghai"),
	}
}
