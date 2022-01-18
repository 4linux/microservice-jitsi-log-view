package utils

import "os"

// Getenv gets the value associated with the environtment variable
// passed in as `key` and returns it, if found. Otherwise, it returns
// `defaultValue`.
func Getenv(key, defaultValue string) string {
	if value, found := os.LookupEnv(key); found {
		return value
	}

	return defaultValue
}
