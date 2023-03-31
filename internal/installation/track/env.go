package track

import "gopkg.in/yaml.v3"

type Env struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func NewEnv(name, value string) *Env {
	return &Env{name, value}
}

func (e *Env) Type() TrackingType {
	return EnvType
}

type Envs []*Env

func (e Envs) UnmarshalTrackable(value string) error {
	return yaml.Unmarshal([]byte(value), &e)
}
