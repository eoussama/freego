package models

import "os"

type Config struct {
	IsPartner bool
	Url       string
	Port      string
	Route     string
	Secret    string
	ApiKey    string
}

func (c Config) Build(options *Options) *Config {
	var resolveOption = func(optionValue string, fallbackValue string) string {
		if options != nil && len(optionValue) > 0 {
			return optionValue
		} else {
			return fallbackValue
		}
	}

	c.IsPartner = options.FreestuffPartner
	c.Url = "https://api.freestuffbot.xyz/v1"
	c.Port = resolveOption(options.WebhookPort, c.getEnv("FREEGO_WEBHOOK_PORT", "8080"))
	c.Secret = resolveOption(options.WebhookSecret, c.getEnv("FREEGO_WEBHOOK_SECRET", ""))
	c.ApiKey = resolveOption(options.FreestuffApiKey, c.getEnv("FREEGO_FREESTUFF_API_KEY", ""))
	c.Route = resolveOption(options.WebhookRoute, c.getEnv("FREEGO_WEBHOOK_ROUTE", "/webhook"))

	return &c
}

func (c Config) getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}
