package osutils

type RegistryKey interface {
	GetStringValue(name string) (val string, valtype uint32, err error)
	SetStringValue(name, value string) error
	SetExpandStringValue(name, value string) error
	DeleteValue(name string) error
	Close() error
}
