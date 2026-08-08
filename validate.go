package projectspec

import (
	"fmt"
	"strings"
)

// validator accumulates spec problems so Load can report them all at once.
type validator struct {
	vars     Vars
	problems []string
}

func (v *validator) problemf(format string, args ...any) {
	v.problems = append(v.problems, fmt.Sprintf(format, args...))
}

func (v *validator) err() error {
	if len(v.problems) == 0 {
		return nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "project spec: %d problem(s):", len(v.problems))
	for _, p := range v.problems {
		fmt.Fprintf(&b, "\n  - %s", p)
	}
	return fmt.Errorf("%s", b.String())
}

// known backends. Extend as new updater backends appear.
var knownSourceBackends = map[string]bool{"github": true}
var knownApplyBackends = map[string]bool{"self": true, "files": true}

func (v *validator) validate(spec *Spec) {
	if len(spec.Updaters) == 0 {
		v.problemf("updaters: at least one updater is required")
		return
	}
	seen := map[string]bool{}
	for i := range spec.Updaters {
		u := &spec.Updaters[i]
		path := fmt.Sprintf("updaters[%d]", i)
		if u.Name == "" {
			v.problemf("%s.name: required", path)
		} else {
			path = fmt.Sprintf("updaters[%d](%s)", i, u.Name)
			if seen[u.Name] {
				v.problemf("%s.name: duplicate", path)
			}
			seen[u.Name] = true
		}
		v.validateSource(path, &u.Source)
		v.validateApply(path, &u.Apply)
	}
}

func (v *validator) validateSource(path string, s *SourceSpec) {
	path += ".source"
	if !knownSourceBackends[s.Backend] {
		v.problemf("%s.backend: unknown %q (known: %s)", path, s.Backend, keys(knownSourceBackends))
		return
	}
	if s.Owner == "" {
		v.problemf("%s.owner: required", path)
	}
	if s.Repo == "" {
		v.problemf("%s.repo: required", path)
	}
	s.BaseURL = v.resolve(path+".base_url", s.BaseURL)
	for i, tag := range s.AssetTags {
		s.AssetTags[i] = v.resolve(fmt.Sprintf("%s.asset_tags[%d]", path, i), tag)
	}
}

func (v *validator) validateApply(path string, a *ApplySpec) {
	path += ".apply"
	if !knownApplyBackends[a.Backend] {
		v.problemf("%s.backend: unknown %q (known: %s)", path, a.Backend, keys(knownApplyBackends))
		return
	}
	if a.BaseDir == "" {
		v.problemf("%s.base_dir: required", path)
	} else {
		a.BaseDir = v.resolve(path+".base_dir", a.BaseDir)
	}
	if a.InstallScript != "" && a.Backend != "files" {
		v.problemf("%s.install_script: only valid with backend \"files\"", path)
	}
}

func keys(m map[string]bool) string {
	list := make([]string, 0, len(m))
	for k := range m {
		list = append(list, k)
	}
	return strings.Join(list, ", ")
}
