package projectspec

import (
	"strings"
	"testing"
)

const validDoc = `
updaters:
  - name: updater
    source: {backend: github, owner: example, repo: example-app}
    apply: {backend: self, base_dir: $BASE_DIR}
  - name: core-updater
    source: {backend: github, owner: example, repo: example-core, asset_tags: [$GOOS, $GOARCH]}
    apply: {backend: files, base_dir: $DATA_DIR, install_script: core.lua}
`

func testVars() Vars {
	return Vars{"BASE_DIR": "/app", "DATA_DIR": "/data", "GOOS": "linux", "GOARCH": "amd64"}
}

func TestLoadValid(t *testing.T) {
	spec, err := Load([]byte(validDoc), testVars())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(spec.Updaters) != 2 {
		t.Fatalf("want 2 updaters, got %d", len(spec.Updaters))
	}
	self := spec.Updaters[0]
	if self.Apply.BaseDir != "/app" {
		t.Errorf("self base_dir: want /app, got %q", self.Apply.BaseDir)
	}
	core := spec.Updaters[1]
	if core.Apply.BaseDir != "/data" {
		t.Errorf("core base_dir: want /data, got %q", core.Apply.BaseDir)
	}
	wantTags := []string{"linux", "amd64"}
	if len(core.Source.AssetTags) != 2 || core.Source.AssetTags[0] != wantTags[0] || core.Source.AssetTags[1] != wantTags[1] {
		t.Errorf("asset_tags: want %v, got %v", wantTags, core.Source.AssetTags)
	}
	if core.Apply.InstallScript != "core.lua" {
		t.Errorf("install_script: got %q", core.Apply.InstallScript)
	}
}

func TestLoadDefaultsGOOSGOARCH(t *testing.T) {
	spec, err := Load([]byte(validDoc), Vars{"BASE_DIR": "/app", "DATA_DIR": "/data"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	tags := spec.Updaters[1].Source.AssetTags
	if tags[0] == "$GOOS" || tags[1] == "$GOARCH" {
		t.Errorf("GOOS/GOARCH not resolved by default: %v", tags)
	}
}

func TestLoadNetUserAgent(t *testing.T) {
	doc := validDoc + `
net:
  user_agent: my-app/$VERSION/$GOOS
`
	vars := testVars()
	vars["VERSION"] = "1.2.3"
	spec, err := Load([]byte(doc), vars)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := "my-app/1.2.3/linux"; spec.Net.UserAgent != want {
		t.Errorf("net.user_agent: want %q, got %q", want, spec.Net.UserAgent)
	}
}

func TestLoadNetUserAgentUnknownPlaceholder(t *testing.T) {
	doc := validDoc + `
net:
  user_agent: my-app/$NOPE
`
	_, err := Load([]byte(doc), testVars())
	if err == nil || !strings.Contains(err.Error(), `unknown placeholder "$NOPE"`) {
		t.Fatalf("want unknown placeholder error, got %v", err)
	}
}

func TestLoadReportsAllProblems(t *testing.T) {
	doc := `
updaters:
  - name: updater
    source: {backend: gitlab, owner: a, repo: b}
    apply: {backend: self, base_dir: $BASE_DIR}
  - name: updater
    source: {backend: github}
    apply: {backend: files, install_script: x.lua}
  - name: third
    source: {backend: github, owner: a, repo: b}
    apply: {backend: files, base_dir: $NOPE}
`
	_, err := Load([]byte(doc), testVars())
	if err == nil {
		t.Fatal("want error")
	}
	msg := err.Error()
	for _, want := range []string{
		"problem(s)",
		"duplicate",
		`unknown "gitlab"`,
		".owner: required",
		".repo: required",
		".base_dir: required",
		`unknown placeholder "$NOPE"`,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error missing %q:\n%s", want, msg)
		}
	}
}

func TestLoadInstallScriptOnlyFiles(t *testing.T) {
	doc := `
updaters:
  - name: updater
    source: {backend: github, owner: a, repo: b}
    apply: {backend: self, base_dir: $BASE_DIR, install_script: x.lua}
`
	_, err := Load([]byte(doc), testVars())
	if err == nil || !strings.Contains(err.Error(), "install_script") {
		t.Fatalf("want install_script error, got %v", err)
	}
}

func TestLoadEmptyUpdaters(t *testing.T) {
	if _, err := Load([]byte("updaters: []"), testVars()); err == nil {
		t.Fatal("want error for empty updaters")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	if _, err := Load([]byte("{"), testVars()); err == nil {
		t.Fatal("want YAML error")
	}
}
