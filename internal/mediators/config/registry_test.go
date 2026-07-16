package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalEnvVarName(t *testing.T) {
	assert.Equal(t, "ACTIVESTATE_CONFIG_API_HOST", CanonicalEnvVarName("api.host"))
	assert.Equal(t, "ACTIVESTATE_CONFIG_AUTOUPDATE", CanonicalEnvVarName("autoupdate"))
	assert.Equal(t, "ACTIVESTATE_CONFIG_SECURITY_PROMPT_LEVEL", CanonicalEnvVarName("security.prompt.level"))
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
	RegisterOption("test.alias.key", String, "default", "LEGACY_ALIAS_VAR")
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

func TestEnvOverrideInvalidIgnored(t *testing.T) {
	RegisterOption("test.invalid.bool", Bool, false)
	RegisterOption("test.invalid.int", Int, 0)
	RegisterOption("test.invalid.enum", Enum, NewEnum([]string{"low", "high"}, "low"))

	// An unparseable bool must be ignored rather than coerced to false.
	t.Setenv("ACTIVESTATE_CONFIG_TEST_INVALID_BOOL", "notabool")
	_, _, ok := EnvOverride(GetOption("test.invalid.bool"))
	assert.False(t, ok, "an unparseable bool env var must not apply")

	// An unparseable int must be ignored rather than coerced to 0.
	t.Setenv("ACTIVESTATE_CONFIG_TEST_INVALID_INT", "notanint")
	_, _, ok = EnvOverride(GetOption("test.invalid.int"))
	assert.False(t, ok, "an unparseable int env var must not apply")

	// An enum value outside the allowed set must be ignored.
	t.Setenv("ACTIVESTATE_CONFIG_TEST_INVALID_ENUM", "banana")
	_, _, ok = EnvOverride(GetOption("test.invalid.enum"))
	assert.False(t, ok, "an out-of-set enum env var must not apply")

	// A valid enum value still applies.
	t.Setenv("ACTIVESTATE_CONFIG_TEST_INVALID_ENUM", "high")
	value, _, ok := EnvOverride(GetOption("test.invalid.enum"))
	assert.True(t, ok)
	assert.Equal(t, "high", value)
}
