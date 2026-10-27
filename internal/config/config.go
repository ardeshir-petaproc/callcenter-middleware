package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config aggregates all runtime configuration for callcenter-middleware.
type Config struct {
	HTTPPort       string
	ARIURL         string
	ARIWSURL       string
	ARIUsername    string
	ARIPassword    string
	ARIApp         string
	DefaultQueue   string
	QueueContext   string
	IVRTimeout     time.Duration
	MaxRetries     int
	GreetingMedia  string
	WebhookURL     string
	QueueMappings  map[string]string
}

// LoadFromEnv loads configuration parameters with sensible defaults.
func LoadFromEnv() *Config {
	cfg := &Config{
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
		ARIURL:        getEnv("ARI_URL", "http://127.0.0.1:8088/ari"),
		ARIWSURL:      getEnv("ARI_WS_URL", "ws://127.0.0.1:8088/ari/events"),
		ARIUsername:   getEnv("ARI_USERNAME", "callcenter-app"),
		ARIPassword:   getEnv("ARI_PASSWORD", "pbx_ari_secret_2026"),
		ARIApp:        getEnv("ARI_APP", "callcenter-app"),
		DefaultQueue:  getEnv("DEFAULT_QUEUE", "600"),
		QueueContext:  getEnv("QUEUE_CONTEXT", "ext-queues"),
		IVRTimeout:    time.Duration(getEnvInt("IVR_TIMEOUT_SEC", 5)) * time.Second,
		MaxRetries:    getEnvInt("MAX_RETRIES", 2),
		GreetingMedia: getEnv("GREETING_MEDIA", "sound:custom/ivr-welcome"),
		WebhookURL:    getEnv("WEBHOOK_URL", ""),
		QueueMappings: parseQueueMappings(getEnv("QUEUE_MAPPINGS", "1=600,2=601,0=600")),
	}
	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func parseQueueMappings(raw string) map[string]string {
	out := make(map[string]string)
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			out[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return out
}
