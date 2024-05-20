package models

import "os"

type Config struct {
	Url    string
	Port   string
	Route  string
	Secret string
	ApiKey string
}

func (c Config) Build(options *Options) Config {
	var resolveOption = func(optionValue string, fallbackValue string) string {
		if options != nil && len(optionValue) > 0 {
			return optionValue
		} else {
			return fallbackValue
		}
	}

	return Config{
		Url:    "https://api.freestuffbot.xyz/v1",
		Port:   resolveOption(options.WebhookPort, c.getEnv("FREEGO_WEBHOOK_PORT", "8080")),
		Secret: resolveOption(options.WebhookSecret, c.getEnv("FREEGO_WEBHOOK_SECRET", "")),
		ApiKey: resolveOption(options.FreestuffApiKey, c.getEnv("FREEGO_FREESTUFF_API_KEY", "")),
		Route:  resolveOption(options.WebhookRoute, c.getEnv("FREEGO_WEBHOOK_ROUTE", "/webhook")),
	}
}

func (c Config) getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return fallback
}
