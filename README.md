# projectspec

Loads declarative project data from a YAML document: the updaters an application offers and the schema of its settings. The spec is embedded by the application and passed to `Load` as bytes; the library itself is application-agnostic.

```go
//go:embed project.yaml
var projectYAML []byte

spec, err := projectspec.Load(projectYAML, projectspec.Vars{
	"BASE_DIR": "/opt/example",
	"DATA_DIR": "/var/lib/example",
})
```

## Features

- **Updaters**: named release sources (`github`) paired with an install backend (`self` replaces the running binary, `files` unpacks a release archive into a base directory).
- **Runtime placeholders**: `$TOKEN` occurrences in updater strings are resolved from the `Vars` map; `GOOS` and `GOARCH` are pre-populated from the runtime. An unknown token is reported as a problem instead of being dropped silently.
- **Config schema**: the `config` section is flattened into document-ordered entries (`ConfigEntry`), each a single setting addressed by its path from the section root, with a declared type and optional UI control hints.
- **All problems at once**: validation collects every finding and returns them in a single error, one per line.
- Only dependency: `gopkg.in/yaml.v3`.

## Document shape

```yaml
updaters:
  - name: app
    source: {backend: github, owner: example, repo: example-app}
    apply: {backend: self, base_dir: $BASE_DIR}
  - name: core
    source: {backend: github, owner: example, repo: example-core, asset_tags: [$GOOS, $GOARCH]}
    apply: {backend: files, base_dir: $DATA_DIR, install_script: core.lua}

config:
  core:
    log:
      level:
        type: string
        default: info
        control: select
        options: [debug, info, warn, error]
      limit:
        type: int
        default: 1000
        control: number
        min: 100
        max: 100000
    tun:
      enabled:
        type: bool
        default: false
        control: switch
  ui:
    open_data_dir:
      control: action
      action: open_data_dir
      confirm: true
```

### Updaters

At least one updater is required. Each needs a unique `name`; `source` declares where releases come from and `apply` how they are installed.

- `source.backend` — the source kind. Known: `github`. `owner` and `repo` are required; `base_url` selects a GitHub Enterprise instance (empty means github.com); `asset_tags` filter release assets (for example `["$GOOS", "$GOARCH"]`, empty means no filtering).
- `apply.backend` — the install kind. Known: `self` and `files`. `base_dir` is the install root, usually a placeholder such as `$BASE_DIR` or `$DATA_DIR`. `install_script` is an optional Lua post-install script name and is only valid with the `files` backend.

Placeholders are resolved in `source.base_url`, `source.asset_tags` and `apply.base_dir`. The config section keeps its literals untouched.

### Config section

The section is walked as a `yaml.Node` tree rather than unmarshalled, so entry order is preserved for the UI. A mapping is a **leaf** (an entry) when it carries any of the attribute keys `type`, `default`, `control`, `options`, `min`, `max`, `disabled`, `platforms`, `action`, `confirm`; a mapping without them is a **section** and is recursed into. Mixing the two in one mapping, or repeating a path, is an error.

| Attribute | Meaning |
|---|---|
| `type` | `bool`, `int` or `string`. Required for a value-carrying entry. |
| `default` | Required for a value-carrying entry; decoded as `type`. |
| `control` | UI hint: `""` (default), `text`, `number`, `switch`, `select` or `action`. `number` requires `int`, `switch` requires `bool`, `text` requires `string`. |
| `options` | Allowed values for `select`, decoded per `type`; `select` needs at least one. |
| `min`, `max` | Bounds for `int` entries. `min > max`, or a default outside the bounds, is an error. |
| `disabled` | Marks the entry as not user-editable. |
| `platforms` | Restricts the entry to the listed `GOOS` values: `linux`, `windows`, `darwin`. Empty means all platforms. |
| `action`, `confirm` | Valid only with `control: action`: such an entry carries no value (no `type`/`default`) and only triggers the backend action id given in `action`. `confirm` asks the UI to confirm first. |

## Validation

`Load` reports every problem it finds, one per line:

```text
project spec: 3 problem(s):
  - updaters[1](updater).name: duplicate
  - updaters[1](updater).source.backend: unknown "gitlab" (known: github)
  - updaters[2](third).apply.base_dir: unknown placeholder "$NOPE"
```

## Code generation

`cmd/projectspec-gen` turns the config section into a typed accessor over a `github.com/he11ah0und/config` sheet: one struct per section, one `*fwconfig.Cell` field per entry, so reads and writes go straight to the sheet.

```console
$ projectspec-gen -spec project.yaml -out internal/project/config_gen.go -pkg project
```

`snake_case` segments become CamelCase with `URL`, `UI`, `ID` and `API` rendered as initialisms (`url_test_url` → `URLTestURL`). `control: action` entries have no cell and are excluded from the accessor.

## Checks

```bash
go build ./... && go vet ./... && go test ./...
```
