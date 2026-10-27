package config

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Config aggregates all runtime configuration for callcenter-middleware.
type Config struct {
	HTTPPort      string
	ARIURL        string
	ARIWSURL      string
	ARIUsername   string
	ARIPassword   string
	ARIApp        string
	DefaultQueue  string
	QueueContext  string
	IVRTimeout    time.Duration
	MaxRetries    int
	GreetingMedia string
	WebhookURL    string
	QueueMappings map[string]string

	DBPath        string
	JWTSecret     string
	JWTGenerated  bool
	TokenTTL      time.Duration
	AdminUsername string
	AdminPassword string
	SeedDemo      bool
	CORSOrigins   []string
	DrainTimeout  time.Duration

	OpenAIAPIKey          string
	OpenAIBaseURL         string
	OpenAIRealtimeURL     string
	OpenAIRealtimeModel   string
	OpenAIVoice           string
	OpenAISummaryModel    string
	OpenAITranscribeModel string

	MediaBindHost      string
	MediaAdvertiseHost string
	MediaPortMin       int
	MediaPortMax       int

	OutboundDialTemplate string
	OutboundCallerID     string
	AutoPauseMisses      int
	DefaultRingTimeout   int
	DefaultWrapupSec     int
	VoicemailMaxSec      int
	VoicemailMedia       string
	CallbackMedia        string
	ClosedMedia          string
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

		DBPath:        getEnv("DB_PATH", "data/callcenter.db"),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		TokenTTL:      time.Duration(getEnvInt("TOKEN_TTL_HOURS", 12)) * time.Hour,
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		SeedDemo:      getEnvBool("SEED_DEMO", false),
		CORSOrigins:   splitList(getEnv("CORS_ORIGINS", "")),
		DrainTimeout:  time.Duration(getEnvInt("DRAIN_TIMEOUT_SEC", 10)) * time.Second,

		OpenAIAPIKey:          getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL:         getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIRealtimeURL:     getEnv("OPENAI_REALTIME_URL", "wss://api.openai.com/v1/realtime"),
		OpenAIRealtimeModel:   getEnv("OPENAI_REALTIME_MODEL", "gpt-realtime"),
		OpenAIVoice:           getEnv("OPENAI_VOICE", "marin"),
		OpenAISummaryModel:    getEnv("OPENAI_SUMMARY_MODEL", "gpt-4o-mini"),
		OpenAITranscribeModel: getEnv("OPENAI_TRANSCRIBE_MODEL", "gpt-4o-mini-transcribe"),

		MediaBindHost:      getEnv("MEDIA_BIND_HOST", "0.0.0.0"),
		MediaAdvertiseHost: getEnv("MEDIA_ADVERTISE_HOST", "127.0.0.1"),
		MediaPortMin:       getEnvInt("MEDIA_PORT_MIN", 40000),
		MediaPortMax:       getEnvInt("MEDIA_PORT_MAX", 40199),

		OutboundDialTemplate: getEnv("OUTBOUND_DIAL_TEMPLATE", "PJSIP/{number}@trunk"),
		OutboundCallerID:     getEnv("OUTBOUND_CALLER_ID", ""),
		AutoPauseMisses:      getEnvInt("AUTO_PAUSE_MISSES", 3),
		DefaultRingTimeout:   getEnvInt("DEFAULT_RING_TIMEOUT_SEC", 20),
		DefaultWrapupSec:     getEnvInt("DEFAULT_WRAPUP_SEC", 10),
		VoicemailMaxSec:      getEnvInt("VOICEMAIL_MAX_SEC", 120),
		VoicemailMedia:       getEnv("VOICEMAIL_MEDIA", "sound:vm-intro"),
		CallbackMedia:        getEnv("CALLBACK_MEDIA", "sound:custom/callback-confirm"),
		ClosedMedia:          getEnv("CLOSED_MEDIA", "sound:custom/closed"),
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = randomHex(32)
		cfg.JWTGenerated = true
	}
	return cfg
}

// AIEnabled reports whether an OpenAI key is configured.
func (c *Config) AIEnabled() bool { return c.OpenAIAPIKey != "" }

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
