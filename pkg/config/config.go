package config

import (
	"os"
	"strings"
)

type Config struct {
	ServiceName      string
	Environment      string
	MetricNamespace  string
	AllowedPaths     []string
	AWS              AWSConfig
}

type AWSConfig struct {
	Region string
}

func Load() *Config {
	serviceName := getEnv("SERVICE_NAME", "unknown")
	environment := getEnv("ENVIRONMENT", "prod")
	
	config := &Config{
		ServiceName:     serviceName,
		Environment:     environment,
		MetricNamespace: serviceName + "/SLI",
		AllowedPaths:    parseAllowedPaths(getEnv("ALLOWED_PATHS", "")),
		AWS: AWSConfig{
			Region: getEnv("AWS_REGION", "us-east-1"),
		},
	}
	
	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseAllowedPaths(pathsStr string) []string {
	if pathsStr == "" {
		return []string{}
	}
	
	paths := strings.Split(pathsStr, ",")
	var result []string
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (c *Config) IsPathAllowed(path string) bool {
	if len(c.AllowedPaths) == 0 {
		return true
	}
	
	for _, allowedPath := range c.AllowedPaths {
		if path == allowedPath {
			return true
		}
	}
	return false
}