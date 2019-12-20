package env

import (
	"os"
	"strconv"
	"testing"
)

func TestStrEnv(t *testing.T) {
	key := "SERVICE_NAME"
	value := "TESTE"
	os.Setenv(key, value)

	srvName := GetString(key, "")
	if srvName != value {
		t.Fatalf("Values are different, expect %s but got %s", value, srvName)
	}
	os.Clearenv()
}

func TestStrEnvDefault(t *testing.T) {
	key := "ENV"
	defaultValue := "DEFAULT_VALUE"

	envValue := GetString(key, defaultValue)
	if envValue != defaultValue {
		t.Fatalf("Values are different, expect %s but got %s", defaultValue, envValue)
	}

	os.Clearenv()
}

func TestNotFoundEnv(t *testing.T) {
	key := "ENV"

	envValue := GetString(key, "")
	if len(envValue) > 0 {
		t.Fatalf("Expected not found a value for the key = %s, but got %s", key, envValue)
	}

	os.Clearenv()
}

func TestIntEnv(t *testing.T) {
	key := "ENV_INT"
	value := 10
	os.Setenv(key, strconv.Itoa(value))

	envInt := GetInt(key, 0)
	if envInt != value {
		t.Fatalf("Values are different, expect %d but got %d", value, envInt)
	}

	os.Clearenv()
}
func TestIntEnvDefault(t *testing.T) {
	key := "ENV_INT"
	value := 10

	envInt := GetInt(key, value)
	if envInt != value {
		t.Fatalf("Values are different, expect %d but got %d", value, envInt)
	}

	os.Clearenv()
}

func TestBoolEnv(t *testing.T) {
	key := "ENV_BOOL"
	os.Setenv(key, "true")

	envValue := GetBool(key, false)
	if !envValue {
		t.Fatalf("Expected the env value be true")
	}

	os.Clearenv()
}

func TestBoolEnvDefault(t *testing.T) {
	key := "ENV_BOOL"
	value := true

	envValue := GetBool(key, value)
	if value != envValue {
		t.Fatalf("Values are different, expect %t but got %t", value, envValue)
	}

	os.Clearenv()
}
