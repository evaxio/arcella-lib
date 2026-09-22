package rateLimiter

type ConfigRateLimiter struct {
	Limit   int `yaml:"limit"   env:"WEB_RATE_LIMIT"    default:"4"`
	Seconds int `yaml:"seconds" env:"WEB_RATE_SECONDS"  default:"1" `
}
