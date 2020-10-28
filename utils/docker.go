package utils

import "github.com/docker/docker/api/types"

// ProjectName get project name.
func ProjectName(c *types.Container) string {
	value, ok := c.Labels["com.docker.compose.project"]
	if ok && value != "" {
		return value
	}
	return ""
}

// ConfigFile get project name.
func ConfigFile(c *types.Container) string {
	value, ok := c.Labels["com.docker.compose.project.config_files"]
	if ok && value != "" {
		return value
	}
	return ""
}

// ConfigDir get project name.
func ConfigDir(c *types.Container) string {
	value, ok := c.Labels["com.docker.compose.project.working_dir"]
	if ok && value != "" {
		return value
	}
	return ""
}
