package installCmd

type opts struct{}

// Register the install command
func Register() (command string, shortDescription string, longDescription string, data interface{}) {
	return "install", "install a project", "install a project", &opts{}
}
