// Package projectspec loads declarative project data (updaters, and later
// config schema) from an embedded YAML file, validates it, and resolves
// runtime placeholders.
//
// The spec file is embedded by the application and passed to Load as bytes;
// the library itself is application-agnostic.
package projectspec

// Spec is the root of a project.yaml file.
type Spec struct {
	Updaters []UpdaterSpec `yaml:"updaters"`
}

// UpdaterSpec declares one update manager: where releases come from and how
// they are applied.
type UpdaterSpec struct {
	Name   string     `yaml:"name"`
	Source SourceSpec `yaml:"source"`
	Apply  ApplySpec  `yaml:"apply"`
}

// SourceSpec declares a release source backend.
type SourceSpec struct {
	// Backend is the source kind. Known: "github".
	Backend string `yaml:"backend"`
	// BaseURL selects a GitHub Enterprise instance when set; empty means
	// github.com.
	BaseURL string `yaml:"base_url"`
	Owner   string `yaml:"owner"`
	Repo    string `yaml:"repo"`
	// AssetTags filter release assets (e.g. ["$GOOS", "$GOARCH"]). Empty
	// means no filtering.
	AssetTags []string `yaml:"asset_tags"`
}

// ApplySpec declares an install backend.
type ApplySpec struct {
	// Backend is the apply kind. Known: "self" (replace the running binary),
	// "files" (unpack a release archive into BaseDir).
	Backend string `yaml:"backend"`
	// BaseDir is the install root, usually a placeholder such as "$BASE_DIR"
	// or "$DATA_DIR".
	BaseDir string `yaml:"base_dir"`
	// InstallScript is an optional Lua post-install script name, valid only
	// with the "files" backend.
	InstallScript string `yaml:"install_script"`
}
