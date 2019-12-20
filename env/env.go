package env

import (
	"os"
	"strconv"
)

// GetString Simple helper function to read an environment or return a default value
// return an enviroment string value
func GetString(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

// GetBool Simple helper function to read an environment or return a default value
// return an enviroment bool value
func GetBool(key string, defaultVal bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		i, _ := strconv.ParseBool(value)
		return i
	}
	return defaultVal
}

// GetInt Simple helper function to read an environment or return a default value
// return an enviroment int value
func GetInt(key string, defaultVal int) int {
	if value, exists := os.LookupEnv(key); exists {
		i, _ := strconv.Atoi(value)
		return i
	}
	return defaultVal
}
