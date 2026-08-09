package projectspec

import (
	"strings"
	"testing"
)

const configUpdaters = `
updaters:
  - name: updater
    source: {backend: github, owner: a, repo: b}
    apply: {backend: self, base_dir: $BASE_DIR}
`

func loadConfigDoc(cfg string) (*Spec, error) {
	return Load([]byte(configUpdaters+cfg), testVars())
}

const configExample = `
config:
  core:
    url_test_url: {type: string, default: "http://cp.cloudflare.com/generate_204", control: text}
    log:
      level: {type: string, default: error, control: select, options: [debug, info, warn, error]}
    traffic_graph_history: {type: int, default: 60, control: select, options: [30, 60, 120, 300]}
  log:
    limit: {type: int, default: 100, control: number, min: 10}
  plugins:
    enabled: {type: bool, default: false, disabled: true}
`

func TestConfigOrderedFlatten(t *testing.T) {
	spec, err := loadConfigDoc(configExample)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantPaths := []string{
		"core.url_test_url",
		"core.log.level",
		"core.traffic_graph_history",
		"log.limit",
		"plugins.enabled",
	}
	if len(spec.Config) != len(wantPaths) {
		t.Fatalf("want %d entries, got %d", len(wantPaths), len(spec.Config))
	}
	for i, want := range wantPaths {
		if got := strings.Join(spec.Config[i].Path, "."); got != want {
			t.Errorf("entry %d: want path %s, got %s", i, want, got)
		}
	}
}

func TestConfigDefaultsTypes(t *testing.T) {
	spec, err := loadConfigDoc(configExample)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	url := spec.Config[0]
	if url.Type != "string" || url.Default != "http://cp.cloudflare.com/generate_204" || url.Control != "text" {
		t.Errorf("url_test_url: %+v", url)
	}
	level := spec.Config[1]
	if level.Default != "error" || level.Control != "select" {
		t.Errorf("level: %+v", level)
	}
	if len(level.Options) != 4 {
		t.Fatalf("level options: want 4, got %d", len(level.Options))
	}
	for i, o := range level.Options {
		if _, ok := o.(string); !ok {
			t.Errorf("level options[%d]: want string, got %T (%v)", i, o, o)
		}
	}
	hist := spec.Config[2]
	if d, ok := hist.Default.(int); !ok || d != 60 {
		t.Errorf("history default: want int 60, got %T (%v)", hist.Default, hist.Default)
	}
	for i, o := range hist.Options {
		if _, ok := o.(int); !ok {
			t.Errorf("history options[%d]: want int, got %T (%v)", i, o, o)
		}
	}
	limit := spec.Config[3]
	if d, ok := limit.Default.(int); !ok || d != 100 {
		t.Errorf("limit default: want int 100, got %T (%v)", limit.Default, limit.Default)
	}
	if limit.Min == nil || *limit.Min != 10 || limit.Max != nil {
		t.Errorf("limit min/max: got %v/%v", limit.Min, limit.Max)
	}
	enabled := spec.Config[4]
	if d, ok := enabled.Default.(bool); !ok || d {
		t.Errorf("enabled default: want bool false, got %T (%v)", enabled.Default, enabled.Default)
	}
	if !enabled.Disabled || enabled.Control != "" {
		t.Errorf("enabled: %+v", enabled)
	}
}

func TestConfigPlaceholdersLiteral(t *testing.T) {
	spec, err := loadConfigDoc("\nconfig:\n  x: {type: string, default: \"$DATA_DIR/literal\"}\n")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if spec.Config[0].Default != "$DATA_DIR/literal" {
		t.Errorf("default resolved unexpectedly: %v", spec.Config[0].Default)
	}
}

func TestConfigPlatforms(t *testing.T) {
	doc := `
config:
  privileges:
    run_as_admin: {type: bool, default: false, control: switch, platforms: [windows, darwin]}
  core:
    url_test_url: {type: string, default: "http://cp.cloudflare.com/generate_204", control: text}
`
	spec, err := loadConfigDoc(doc)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(spec.Config) != 2 {
		t.Fatalf("want 2 entries, got %d", len(spec.Config))
	}
	admin := spec.Config[0]
	if got := strings.Join(admin.Path, "."); got != "privileges.run_as_admin" {
		t.Errorf("entry 0: want path privileges.run_as_admin, got %s", got)
	}
	want := []string{"windows", "darwin"}
	if len(admin.Platforms) != len(want) {
		t.Fatalf("run_as_admin platforms: want %v, got %v", want, admin.Platforms)
	}
	for i, p := range want {
		if admin.Platforms[i] != p {
			t.Errorf("run_as_admin platforms[%d]: want %q, got %q", i, p, admin.Platforms[i])
		}
	}
	if spec.Config[1].Platforms != nil {
		t.Errorf("url_test_url: absent platforms must stay nil, got %v", spec.Config[1].Platforms)
	}
}

func TestConfigValidation(t *testing.T) {
	cases := []struct {
		name   string
		config string
		wants  []string
	}{
		{"unknown type", `config: {x: {type: float, default: 1.5}}`,
			[]string{`config.x.type: unknown "float"`}},
		{"missing type", `config: {x: {default: 1}}`,
			[]string{"config.x.type: required"}},
		{"missing default", `config: {x: {type: bool}}`,
			[]string{"config.x.default: required"}},
		{"string default for int", `config: {x: {type: int, default: "60"}}`,
			[]string{"config.x.default: must be an int"}},
		{"int default for bool", `config: {x: {type: bool, default: 1}}`,
			[]string{"config.x.default: must be a bool"}},
		{"unknown control", `config: {x: {type: string, default: a, control: slider}}`,
			[]string{`config.x.control: unknown "slider"`}},
		{"select without options", `config: {x: {type: string, default: a, control: select}}`,
			[]string{"config.x.options: select control requires at least one option"}},
		{"number control for string", `config: {x: {type: string, default: a, control: number}}`,
			[]string{`config.x.control: "number" only valid for type int`}},
		{"switch control for int", `config: {x: {type: int, default: 1, control: switch}}`,
			[]string{`config.x.control: "switch" only valid for type bool`}},
		{"text control for int", `config: {x: {type: int, default: 1, control: text}}`,
			[]string{`config.x.control: "text" only valid for type string`}},
		{"min for bool", `config: {x: {type: bool, default: true, min: 1}}`,
			[]string{"config.x.min: only valid for type int"}},
		{"min over max", `config: {x: {type: int, default: 5, min: 10, max: 2}}`,
			[]string{"config.x: min 10 > max 2"}},
		{"default below min", `config: {x: {type: int, default: 5, min: 10}}`,
			[]string{"config.x.default: 5 below min 10"}},
		{"default above max", `config: {x: {type: int, default: 60, max: 30}}`,
			[]string{"config.x.default: 60 above max 30"}},
		{"option type mismatch", `config: {x: {type: int, default: 1, control: select, options: [1, two]}}`,
			[]string{"config.x.options[1]: must be an int"}},
		{"options not a list", `config: {x: {type: int, default: 1, options: 3}}`,
			[]string{"config.x.options: must be a list"}},
		{"duplicate path", "config:\n  x: {type: bool, default: true}\n  x: {type: int, default: 1}",
			[]string{"config.x: duplicate path"}},
		{"leaf and section", `config: {x: {type: bool, default: true, sub: {type: int, default: 1}}}`,
			[]string{"config.x: cannot be both an entry and a section"}},
		{"disabled not bool", `config: {x: {type: bool, default: false, disabled: 1}}`,
			[]string{"config.x.disabled: must be a bool"}},
		{"unknown platform", `config: {x: {type: bool, default: false, platforms: [freebsd]}}`,
			[]string{`config.x.platforms[0]: unknown "freebsd"`}},
		{"platform not a string", `config: {x: {type: bool, default: false, platforms: [1]}}`,
			[]string{"config.x.platforms[0]: must be a string"}},
		{"platforms not a list", `config: {x: {type: bool, default: false, platforms: linux}}`,
			[]string{"config.x.platforms: must be a list"}},
		{"min not int", `config: {x: {type: int, default: 5, min: "10"}}`,
			[]string{"config.x.min: must be an int"}},
		{"scalar value", `config: {x: 5}`,
			[]string{"config.x: expected a mapping"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadConfigDoc("\n" + tc.config + "\n")
			if err == nil {
				t.Fatalf("want error")
			}
			msg := err.Error()
			for _, want := range tc.wants {
				if !strings.Contains(msg, want) {
					t.Errorf("error missing %q:\n%s", want, msg)
				}
			}
		})
	}
}

func TestConfigReportsAllProblems(t *testing.T) {
	doc := `
config:
  a: {type: float, default: 1}
  b: {type: int, default: "x"}
  c: {type: string, default: s, control: number}
  d: {type: bool, default: false, min: 1}
`
	_, err := loadConfigDoc(doc)
	if err == nil {
		t.Fatal("want error")
	}
	msg := err.Error()
	for _, want := range []string{
		"problem(s)",
		`config.a.type: unknown "float"`,
		"config.b.default: must be an int",
		`config.c.control: "number" only valid for type int`,
		"config.d.min: only valid for type int",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error missing %q:\n%s", want, msg)
		}
	}
}
