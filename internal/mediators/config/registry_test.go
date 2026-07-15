package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalEnvVarName(t *testing.T) {
	assert.Equal(t, "ACTIVESTATE_CONFIG_API_HOST", CanonicalEnvVarName("api.host"))
	assert.Equal(t, "ACTIVESTATE_CONFIG_AUTOUPDATE", CanonicalEnvVarName("autoupdate"))
	assert.Equal(t, "ACTIVESTATE_CONFIG_UPDATE_INFO_ENDPOINT", CanonicalEnvVarName("update.info.endpoint"))
	assert.Equal(t, "ACTIVESTATE_CONFIG_PRIVATEINGREDIENT_MTLS_CERT", CanonicalEnvVarName("privateingredient.mtls_cert"))
}

func TestEnvOverrideCanonical(t *testing.T) {
	RegisterOption("test.canonical.key", String, "default")
	opt := GetOption("test.canonical.key")

	// Not set -> no override.
	_, _, ok := EnvOverride(opt)
	assert.False(t, ok)

	// Canonical variable applies.
	t.Setenv("ACTIVESTATE_CONFIG_TEST_CANONICAL_KEY", "from-canonical")
	value, envVar, ok := EnvOverride(opt)
	assert.True(t, ok)
	assert.Equal(t, "from-canonical", value)
	assert.Equal(t, "ACTIVESTATE_CONFIG_TEST_CANONICAL_KEY", envVar)
}

func TestEnvOverrideAliasAndPrecedence(t *testing.T) {
	RegisterOptionWithEnv("test.alias.key", String, "default", "LEGACY_ALIAS_VAR")
	opt := GetOption("test.alias.key")

	// A legacy alias applies when the canonical var is unset.
	t.Setenv("LEGACY_ALIAS_VAR", "from-alias")
	value, envVar, ok := EnvOverride(opt)
	assert.True(t, ok)
	assert.Equal(t, "from-alias", value)
	assert.Equal(t, "LEGACY_ALIAS_VAR", envVar)

	// The canonical variable takes precedence over the alias.
	t.Setenv("ACTIVESTATE_CONFIG_TEST_ALIAS_KEY", "from-canonical")
	value, envVar, ok = EnvOverride(opt)
	assert.True(t, ok)
	assert.Equal(t, "from-canonical", value)
	assert.Equal(t, "ACTIVESTATE_CONFIG_TEST_ALIAS_KEY", envVar)
}

func TestEnvOverrideTypeCoercion(t *testing.T) {
	RegisterOption("test.coerce.bool", Bool, false)
	RegisterOption("test.coerce.int", Int, 0)

	t.Setenv("ACTIVESTATE_CONFIG_TEST_COERCE_BOOL", "true")
	v, _, ok := EnvOverride(GetOption("test.coerce.bool"))
	assert.True(t, ok)
	assert.Equal(t, true, v, "bool env value should be coerced to a real bool")

	t.Setenv("ACTIVESTATE_CONFIG_TEST_COERCE_INT", "42")
	v, _, ok = EnvOverride(GetOption("test.coerce.int"))
	assert.True(t, ok)
	assert.Equal(t, 42, v, "int env value should be coerced to a real int")
}

func TestEnvOverrideEmptyIgnored(t *testing.T) {
	RegisterOption("test.empty.key", String, "default")
	t.Setenv("ACTIVESTATE_CONFIG_TEST_EMPTY_KEY", "")
	_, _, ok := EnvOverride(GetOption("test.empty.key"))
	assert.False(t, ok, "an empty env var should be treated as unset")
}
