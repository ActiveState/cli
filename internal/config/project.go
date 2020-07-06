package config

import (
	"os"

	"github.com/spf13/viper"
)

const projectsKey = "projects"

// SetProject associates the projectName with the project
// path in the config
func SetProject(projectName, projectPath string) {
	projects := viper.GetStringMapString(projectsKey)
	if projects == nil {
		projects = make(map[string]string)
	}

	projects[projectName] = projectPath
	viper.Set(projectsKey, projects)
}

// CleanStaleProjects removes projects that no longer exist
// on a user's filesystem from the projects config entry
func CleanStaleProjects() {
	projects := viper.GetStringMapString(projectsKey)

	for namespace, path := range projects {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			delete(projects, namespace)
		}
	}

	viper.Set("projects", projects)
}
