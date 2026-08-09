// Package projectspec loads declarative project data (updaters and config
// schema) from an embedded YAML file, validates it, and resolves runtime
// placeholders.
//
// The spec file is embedded by the application and passed to Load as bytes;
// the library itself is application-agnostic.
package projectspec

// Spec is the root of a project.yaml file.
type Spec struct {
	Updaters []UpdaterSpec `yaml:"updaters"`
	// Config is the flattened, document-ordered list of settings declared in
	// the "config" section. It is populated by walking the YAML node tree,
	// not by unmarshalling, so entry order is preserved for the UI.
	Config []ConfigEntry `yaml:"-"`
}

// ConfigEntry is one flattened leaf of the "config" section: a single
// setting addressed by its path from the section root, with a declared type
// and optional UI control hints.
type ConfigEntry struct {
	Path    []string // e.g. ["core", "log", "level"]
	Type    string   // "bool" | "int" | "string"
	Default any      // decoded as bool/int/string per Type
	// Disabled marks the entry as not user-editable.
	Disabled bool
	Control  string // "" | "text" | "number" | "switch" | "select" | "action"
	// Action is the backend action id for "action" control entries; such
	// entries carry no value (no type/default) and only trigger the action.
	Action string
	// Confirm asks the UI to confirm before running an "action" entry.
	Confirm bool
	// Platforms restricts the entry to the listed GOOS values ("linux",
	// "windows", "darwin"); empty means the entry applies to all platforms.
	Platforms []string
	// Options holds the allowed values for the "select" control, decoded per
	// Type.
	Options []any
	// Min and Max bound int entries; nil when absent.
	Min, Max *int
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
