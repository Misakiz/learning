package utils

import "os"

func GetEnvDefault(key, defaultValue string) string {
	if "" == os.Getenv(key) {
		return defaultValue
	} else {
		return os.Getenv(key)
	}
}
