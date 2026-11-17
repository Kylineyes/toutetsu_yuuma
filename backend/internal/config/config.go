package config

import (
	"os"
	"strconv"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Host                      string
	Port                      string
	DataDir                   string
	TmpDir                    string
	Password                  string // Web UI login password (empty = web open)
	APIToken                  string // third-party API token via header (empty = not required)
	SessionSecret             string // cookie signing secret (defaults to Password)
	MaxConcurrency            int
	DefaultScreenshotInterval int
	MaxFrames                 int
	MaxUploadBytes            int64 // 单次请求上传总大小上限（字节）
	Timezone                  string
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// getenvInt 读取整数环境变量（字节/兆等以十进制整数表达），失败回退默认值。
func getenvInt(k string, def int) int { return getint(k, def) }

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
	pw := getenv("PASSWORD", "")
	secret := getenv("SESSION_SECRET", "")
	if secret == "" {
		secret = pw
	}
	if secret == "" {
		secret = "toutetsu_yuuma_default_secret"
	}
	return &Config{
		Host:                      getenv("HOST", "0.0.0.0"),
		Port:                      getenv("PORT", "8080"),
		DataDir:                   getenv("DATA_DIR", "./data"),
		TmpDir:                    getenv("TMP_DIR", os.TempDir()),
		Password:                  pw,
		APIToken:                  getenv("API_TOKEN", ""),
		SessionSecret:             secret,
		MaxConcurrency:            getint("MAX_GO_CONCURRENCY", 4),
		DefaultScreenshotInterval: getint("DEFAULT_SCREENSHOT_INTERVAL", 60),
		MaxFrames:                 getint("MAX_FRAMES", 120),
		MaxUploadBytes:            int64(getenvInt("MAX_UPLOAD_BYTES", 2048)) * 1024 * 1024, // 默认 2GiB
		Timezone:                  getenv("TZ", "Asia/Shanghai"),
	}
}
