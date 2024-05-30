package runtime

type Environment struct {
	Variables              map[string]string
	VariablesWithExecutors map[string]string
	ExecutorsPath          string
}
