package projectspec

import (
	"fmt"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Vars maps placeholder names (without the leading "$") to their runtime
// values. Load pre-populates GOOS and GOARCH from the runtime; callers add
// application-specific values such as BASE_DIR and DATA_DIR.
type Vars map[string]string

// Load parses and validates a project.yaml document, resolving "$TOKEN"
// placeholders in all string values using vars.
//
// Every problem found is reported: the returned error message lists all of
// them, one per line.
func Load(data []byte, vars Vars) (*Spec, error) {
	if vars == nil {
		vars = Vars{}
	}
	if _, ok := vars["GOOS"]; !ok {
		vars["GOOS"] = runtime.GOOS
	}
	if _, ok := vars["GOARCH"]; !ok {
		vars["GOARCH"] = runtime.GOARCH
	}

	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("project spec: invalid YAML: %w", err)
	}

	v := &validator{vars: vars}
	v.validate(&spec)
	if err := v.err(); err != nil {
		return nil, err
	}
	return &spec, nil
}

// resolve replaces "$TOKEN" occurrences in s. Unknown tokens are recorded as
// problems against the given field path.
func (v *validator) resolve(path, s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '$' {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := i + 1
		for j < len(s) && (s[j] == '_' || s[j] >= '0' && s[j] <= '9' || s[j] >= 'A' && s[j] <= 'Z') {
			j++
		}
		if j == i+1 {
			b.WriteByte(s[i])
			i++
			continue
		}
		name := s[i+1 : j]
		val, ok := v.vars[name]
		if !ok {
			v.problemf("%s: unknown placeholder \"$%s\"", path, name)
			val = s[i:j]
		}
		b.WriteString(val)
		i = j
	}
	return b.String()
}
