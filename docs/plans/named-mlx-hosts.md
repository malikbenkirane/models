# Named MLX Hosts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use supervised-plan-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Support multiple named MLX hosts in `config.json` so each local vLLM/MLX server is referenced by a stable key (e.g. `gemma`) instead of a hardcoded IP, while keeping the legacy single `mlx_host` working as a fallback.

**Architecture:** Add a `mlx_hosts` map (`name -> "host:port"`) to `configFile` plus a single `mlxEndpoint(key, port)` resolver that applies precedence: named host (port baked into value) → legacy `mlx_host` + the model's per-entry `port` → error. Tag each MLX registry entry with a `modelWithMlxHost(key)` selector; `litellmModels` resolves the endpoint at runtime and fails loudly when a host is unconfigured. The `pricing` command shows the public host key in the provider column and never the private IP. Rename `modelWithMlx16` → `modelWithMlx` (the `16` bit-width detail no longer belongs in the option name).

**Tech Stack:** Go 1.26.2, `github.com/spf13/cobra`, `sigs.k8s.io/yaml`. Tests in package `main` (`models_test.go`), table-driven with `t.Fatalf`. VCS is `jj` (git commands prohibited); commits use the mktemp + `jj describe --stdin` + `jj new` pattern.

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `config.go` | Modify | Add `MlxHosts` map field + `mlxEndpoint` resolver method (precedence in one place) |
| `config.example.json` | Modify | Document `mlx_hosts` map alongside the legacy `mlx_host` fallback |
| `models.go` | Modify | Rename `modelWithMlx16`→`modelWithMlx`; add `mlxHost` field + `modelWithMlxHost` option; tag the two MLX entries; make `litellmModels` return `(models, error)` and use the resolver |
| `main.go` | Modify | Thread the `litellmModels` error through the `litellm` command; show MLX host key in the `pricing` provider column |
| `models_test.go` | Modify | Add `TestMlxEndpoint` (resolver precedence), `TestLitellmModelsMlxMissingHost` (error path), `TestLitellmModelsMlxNamedHost` (api_base wiring) |
| `AGENTS.md` | Modify | Document `mlx_hosts`, the resolver precedence, `modelWithMlxHost`, the rename, and that pricing shows the key not the IP |

Design notes:
- `modelWithMlx(name, port)` keeps its `port` argument; it is only consulted on the **legacy fallback path** (so existing single-host `config.json` files keep working without touching call sites).
- `providerMlx16` enum + its `String()` label `"vllm-mlx"` stay unchanged — those are output-facing labels, not bit-width.
- The `pricing` command reads **no config** (AGENTS.md invariant); it shows the host *key* (public, lives in the registry) never the IP (private, lives in `config.json`). No unit test is added for pricing because it writes to `os.Stdout` via `text/tabwriter` and is not writer-injectable; it is verified by running `go run . pricing`.

---

## Task 1: Add `mlxEndpoint` resolver to `config.go` (TDD)

**Files:**
- Modify: `config.go` (full file)
- Test: `models_test.go` (append `TestMlxEndpoint`)

- [ ] **Step 1: Write the failing test**

Append to `models_test.go` (after `TestDecimalLess`, still in `package main`):

```go
func TestMlxEndpoint(t *testing.T) {
	for _, test := range []struct {
		name     string
		cfg      configFile
		key      string
		port     int
		expected string
		ok       bool
	}{
		{
			name:     "named host wins",
			cfg:      configFile{MlxHosts: map[string]string{"gemma": "192.168.1.101:8000"}},
			key:      "gemma",
			port:     8000,
			expected: "http://192.168.1.101:8000/v1",
			ok:       true,
		},
		{
			name:     "legacy host fallback uses model port",
			cfg:      configFile{MlxHost: "192.168.1.100"},
			key:      "gemma",
			port:     8001,
			expected: "http://192.168.1.100:8001/v1",
			ok:       true,
		},
		{
			name:     "named host preferred over legacy",
			cfg:      configFile{MlxHost: "192.168.1.100", MlxHosts: map[string]string{"gemma": "10.0.0.1:8000"}},
			key:      "gemma",
			port:     9999,
			expected: "http://10.0.0.1:8000/v1",
			ok:       true,
		},
		{
			name:     "missing named host with no legacy returns false",
			cfg:      configFile{MlxHosts: map[string]string{"other": "10.0.0.2:8000"}},
			key:      "gemma",
			port:     8000,
			expected: "",
			ok:       false,
		},
		{
			name:     "empty config returns false",
			cfg:      configFile{},
			key:      "gemma",
			port:     8000,
			expected: "",
			ok:       false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := test.cfg.mlxEndpoint(test.key, test.port)
			if ok != test.ok || got != test.expected {
				t.Fatalf("\nexp: %q %v\ngot: %q %v", test.expected, test.ok, got, ok)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./... -run TestMlxEndpoint`
Expected: compile error — `cfg.mlxEndpoint undefined` (and `configFile.MlxHosts` undefined).

- [ ] **Step 3: Write minimal implementation**

Replace the entire contents of `config.go` with:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type configFile struct {
	VertexProject string            `json:"vertex_project"`
	MlxHost       string            `json:"mlx_host"`  // legacy single-host fallback
	MlxHosts      map[string]string `json:"mlx_hosts"` // name -> "host:port"
}

// mlxEndpoint resolves the v1 base URL for an MLX model. Precedence:
//  1. a named entry in mlx_hosts (port is part of the value),
//  2. the legacy mlx_host combined with the model's per-entry port.
//
// Returns ("", false) when no host is configured.
func (c configFile) mlxEndpoint(key string, port int) (string, bool) {
	if c.MlxHosts != nil {
		if ep, ok := c.MlxHosts[key]; ok {
			return fmt.Sprintf("http://%s/v1", ep), true
		}
	}
	if c.MlxHost != "" {
		return fmt.Sprintf("http://%s:%d/v1", c.MlxHost, port), true
	}
	return "", false
}

func loadConfig() (configFile, error) {
	var cfg configFile
	data, err := os.ReadFile("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./... -run TestMlxEndpoint`
Expected: PASS

- [ ] **Step 5: Commit with caveman-commit**

Draft message (caveman style):

```
feat(config): resolve named MLX hosts with legacy fallback

Adds mlx_hosts map (name -> "host:port") and a mlxEndpoint resolver.
Precedence: named host wins, then legacy mlx_host + model port, else
false. Keeps existing single-host config.json files working.
```

```bash
mktemp
# Write tool: save the message above to the returned /tmp/tmp.XXXXXX
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

## Task 2: Rename `modelWithMlx16` → `modelWithMlx` and add `modelWithMlxHost` option

**Files:**
- Modify: `models.go:156-167` (struct), `models.go:220-227` (option), append new option

This is a pure refactor + additive option; the existing build is the safety net (no behavioral change yet — the new field defaults to `""`).

- [ ] **Step 1: Add the `mlxHost` field to `modelDescription`**

In `models.go`, the struct currently reads (lines 156-167):

```go
type modelDescription struct {
	name                 string
	model                string
	location             string
	litellm              string
	input, output, cache pricing.Decimal
	modalities           *modalities
	attachment           bool
	vision               bool
	provider             modelProvider
	port                 int
}
```

Replace with:

```go
type modelDescription struct {
	name                 string
	model                string
	location             string
	litellm              string
	input, output, cache pricing.Decimal
	modalities           *modalities
	attachment           bool
	vision               bool
	provider             modelProvider
	port                 int
	mlxHost              string
}
```

- [ ] **Step 2: Rename `modelWithMlx16` → `modelWithMlx`**

The option currently reads (lines 220-227):

```go
func modelWithMlx16(name string, port int) modelOpt {
	return func(md modelDescription) modelDescription {
		md.provider = providerMlx16
		md.port = port
		md.model = name
		return md
	}
}
```

Replace with:

```go
func modelWithMlx(name string, port int) modelOpt {
	return func(md modelDescription) modelDescription {
		md.provider = providerMlx16
		md.port = port
		md.model = name
		return md
	}
}
```

- [ ] **Step 3: Add `modelWithMlxHost` option**

Immediately after the `modelWithMlx` function (before `modelWithDeepinfra`), insert:

```go
func modelWithMlxHost(key string) modelOpt {
	return func(md modelDescription) modelDescription {
		md.mlxHost = key
		return md
	}
}
```

- [ ] **Step 4: Update the two MLX call sites to the new name**

In `models.go`, the two MLX entries (lines 116-123) currently read:

```go
	newModel("Gemma-4-12B-it-4bitB [vllm-mlx/m1/16]", "mini-gemma-4-12B",
		modelWithMlx16("mlx-community/Gemma-4-12B-it-4bit", 8000),
		modelWithMultimodalImageSupport(),
		modelWithVision()),
	newModel("MiniCPM-V-4.6-4bit [mlx_vlm.server/m1/16]", "mini-cpm-4-6",
		modelWithMlx16("mlx-community/MiniCPM-V-4.6-4bit", 8001),
		modelWithMultimodalImageSupport(),
		modelWithVision()),
```

Replace with:

```go
	newModel("Gemma-4-12B-it-4bitB [vllm-mlx/m1/16]", "mini-gemma-4-12B",
		modelWithMlx("mlx-community/Gemma-4-12B-it-4bit", 8000),
		modelWithMlxHost("gemma"),
		modelWithMultimodalImageSupport(),
		modelWithVision()),
	newModel("MiniCPM-V-4.6-4bit [mlx_vlm.server/m1/16]", "mini-cpm-4-6",
		modelWithMlx("mlx-community/MiniCPM-V-4.6-4bit", 8001),
		modelWithMlxHost("minicpm"),
		modelWithMultimodalImageSupport(),
		modelWithVision()),
```

- [ ] **Step 5: Build to verify the rename compiles**

Run: `go build ./...`
Expected: success (no references to `modelWithMlx16` remain).

- [ ] **Step 6: Commit with caveman-commit**

```
refactor(models): rename modelWithMlx16 to modelWithMlx

The 16 (M1/16-bit MLX) bit-width is an impl detail that does not
belong in the option name. Adds mlxHost field + modelWithMlxHost
selector; providerMlx16 enum label unchanged.
```

```bash
mktemp
# Write tool: save the message above to the returned /tmp/tmp.XXXXXX
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

## Task 3: Make `litellmModels` resolve endpoints and return an error (TDD)

**Files:**
- Modify: `models.go:126-154` (`litellmModels`)
- Modify: `main.go:47-64` (litellm command)
- Test: `models_test.go` (append two tests)

- [ ] **Step 1: Write the failing tests**

Add imports to `models_test.go`. The import block currently reads:

```go
import (
	"models/pricing"
	"testing"
)
```

Replace with:

```go
import (
	"bytes"
	"models/litellm"
	"models/pricing"
	"strings"
	"testing"
)
```

Append these two tests to `models_test.go`:

```go
func TestLitellmModelsMlxMissingHost(t *testing.T) {
	// No mlx_host or mlx_hosts configured: every MLX model must error.
	if _, err := litellmModels(configFile{}); err == nil {
		t.Fatalf("expected error when no MLX host is configured")
	}
}

func TestLitellmModelsMlxNamedHost(t *testing.T) {
	cfg := configFile{MlxHosts: map[string]string{
		"gemma":   "192.168.1.101:8000",
		"minicpm": "192.168.1.102:8001",
	}}
	params, err := litellmModels(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var b bytes.Buffer
	if err := litellm.ConfigJSON(&b, params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := b.String()
	for _, want := range []string{
		"http://192.168.1.101:8000/v1",
		"http://192.168.1.102:8001/v1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected api_base %q in output, got:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./... -run 'TestLitellmModelsMlx'`
Expected: compile error — `litellmModels(cfg)` returns a single value, not `(models, error)`, and the new import of `litellm` is unused until the signature changes.

- [ ] **Step 3: Change `litellmModels` to return `(models, error)` and use the resolver**

The function currently reads (lines 126-154):

```go
func litellmModels(cfg configFile) []litellm.ModelParams {
	var models []litellm.ModelParams
	for _, m := range myModels {
		var opts []litellm.ModelOpt
		switch m.provider {
		case providerDeepinfra:
			opts = append(opts,
				litellm.ModelWithApikey("os.environ/DEEPINFRA_API_KEY"),
				litellm.ModelWithDeepinfra(m.model))
		case providerMlx16:
			opts = append(opts,
				litellm.ModelWithVllm(m.model, fmt.Sprintf("http://%s:%d/v1", cfg.MlxHost, m.port)))
		case providerVertexAi:
			opts = append(opts,
				litellm.ModelWithVertexAi(cfg.VertexProject, m.model),
			)
			if len(m.location) > 0 {
				opts = append(opts,
					litellm.ModelWithVertexLocation(m.location))
			}
		}
		if m.vision {
			opts = append(opts, litellm.ModelWithVision())
		}
		opts = append(opts, litellm.ModelWithPricing(m.input, m.output))
		models = append(models, litellm.NewModel(m.litellm, opts...))
	}
	return models
}
```

Replace with:

```go
func litellmModels(cfg configFile) ([]litellm.ModelParams, error) {
	var models []litellm.ModelParams
	for _, m := range myModels {
		var opts []litellm.ModelOpt
		switch m.provider {
		case providerDeepinfra:
			opts = append(opts,
				litellm.ModelWithApikey("os.environ/DEEPINFRA_API_KEY"),
				litellm.ModelWithDeepinfra(m.model))
		case providerMlx16:
			endpoint, ok := cfg.mlxEndpoint(m.mlxHost, m.port)
			if !ok {
				return nil, fmt.Errorf("mlx host %q not configured for %s", m.mlxHost, m.litellm)
			}
			opts = append(opts,
				litellm.ModelWithVllm(m.model, endpoint))
		case providerVertexAi:
			opts = append(opts,
				litellm.ModelWithVertexAi(cfg.VertexProject, m.model),
			)
			if len(m.location) > 0 {
				opts = append(opts,
					litellm.ModelWithVertexLocation(m.location))
			}
		}
		if m.vision {
			opts = append(opts, litellm.ModelWithVision())
		}
		opts = append(opts, litellm.ModelWithPricing(m.input, m.output))
		models = append(models, litellm.NewModel(m.litellm, opts...))
	}
	return models, nil
}
```

- [ ] **Step 4: Thread the error through `main.go`'s litellm command**

The litellm command in `main.go` currently reads (lines 45-65):

```go
	litellmCommand := &cobra.Command{
		Use: "litellm",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			var b bytes.Buffer
			if err := litellm.ConfigJSON(&b, litellmModels(cfg)); err != nil {
				return err
			}
			{
				b, err := yaml.JSONToYAML(b.Bytes())
				if err != nil {
					return err
				}
				fmt.Println(string(b))
			}
			return nil
		},
	}
```

Replace with:

```go
	litellmCommand := &cobra.Command{
		Use: "litellm",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			params, err := litellmModels(cfg)
			if err != nil {
				return err
			}
			var b bytes.Buffer
			if err := litellm.ConfigJSON(&b, params); err != nil {
				return err
			}
			{
				b, err := yaml.JSONToYAML(b.Bytes())
				if err != nil {
					return err
				}
				fmt.Println(string(b))
			}
			return nil
		},
	}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./... -run 'TestLitellmModelsMlx'`
Expected: PASS (both the error-path and named-host wiring tests).

- [ ] **Step 6: Run the full suite**

Run: `go test ./... && go vet ./...`
Expected: PASS, vet clean.

- [ ] **Step 7: Commit with caveman-commit**

```
feat(models): error on unconfigured MLX host in litellm output

litellmModels now resolves MLX endpoints via mlxEndpoint and returns
an error when a named host is missing instead of emitting a silently
broken http://:port/v1 base. Threaded through the litellm command.
```

```bash
mktemp
# Write tool: save the message above to the returned /tmp/tmp.XXXXXX
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

## Task 4: Show the MLX host key in the `pricing` provider column

**Files:**
- Modify: `main.go:92-95` (pricing row loop)

The `pricing` command reads no config (AGENTS.md invariant); it shows the public host key, never the private IP.

- [ ] **Step 1: Append the host key to the provider label for MLX models**

The row loop in `main.go` currently reads (lines 92-95):

```go
		for _, m := range rows {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
				m.name, m.litellm, m.provider, m.input, m.output, m.cache)
		}
```

Replace with:

```go
		for _, m := range rows {
			provider := m.provider.String()
			if m.provider == providerMlx16 && m.mlxHost != "" {
				provider += "/" + m.mlxHost
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
				m.name, m.litellm, provider, m.input, m.output, m.cache)
		}
```

- [ ] **Step 2: Verify by running the pricing command**

Run: `go run . pricing`
Expected: MLX rows show provider `vllm-mlx/gemma` and `vllm-mlx/minicpm`; no IPs anywhere in the table.

- [ ] **Step 3: Build and vet**

Run: `go build ./... && go vet ./...`
Expected: success.

- [ ] **Step 4: Commit with caveman-commit**

```
feat(pricing): show MLX host key in provider column

Appends the public host selector (e.g. vllm-mlx/gemma) for MLX rows.
Keeps the private IP out of the config-free pricing output.
```

```bash
mktemp
# Write tool: save the message above to the returned /tmp/tmp.XXXXXX
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

## Task 5: Document `mlx_hosts` in `config.example.json`

**Files:**
- Modify: `config.example.json` (full file)

- [ ] **Step 1: Replace the example file**

The file currently reads:

```json
{
  "vertex_project": "your-gcp-project-id",
  "mlx_host": "192.168.1.100"
}
```

Replace with:

```json
{
  "vertex_project": "your-gcp-project-id",
  "mlx_host": "192.168.1.100",
  "mlx_hosts": {
    "gemma": "192.168.1.101:8000",
    "minicpm": "192.168.1.102:8001"
  }
}
```

- [ ] **Step 2: Verify the litellm command still parses the example**

Run: `cp config.example.json config.json && go run . litellm > /dev/null && rm config.json`
Expected: success (named hosts resolve; legacy `mlx_host` is ignored because named entries win).

- [ ] **Step 3: Commit with caveman-commit**

```
docs(config): document mlx_hosts in example

Shows the named-hosts map alongside the legacy mlx_host fallback.
mlx_hosts values include the port; mlx_host does not.
```

```bash
mktemp
# Write tool: save the message above to the returned /tmp/tmp.XXXXXX
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

## Task 6: Document named MLX hosts in `AGENTS.md`

**Files:**
- Modify: `AGENTS.md` (deployment-specific values section, adding/editing models section, output-format gotchas)

- [ ] **Step 1: Update the deployment-values paragraph**

Find the paragraph beginning `Deployment-specific values that must not be committed`:

> Deployment-specific values that must not be committed live in `config.json` (gitignored): `vertex_project` (GCP project ID for Vertex AI) and `mlx_host` (host of the local vLLM/MLX servers). `config.example.json` is the committed template. Only the `litellm` command reads `config.json` (via `loadConfig` in `config.go`); if the file is absent the values default to empty strings, so `opencode` and `pricing` work without it.

Replace with:

> Deployment-specific values that must not be committed live in `config.json` (gitignored): `vertex_project` (GCP project ID for Vertex AI), `mlx_host` (legacy single host of the local vLLM/MLX servers), and `mlx_hosts` (a map of `name -> "host:port"` for multiple named MLX servers). `config.example.json` is the committed template. Only the `litellm` command reads `config.json` (via `loadConfig` in `config.go`); if the file is absent the values default to empty strings, so `opencode` and `pricing` work without it. MLX endpoint resolution (`configFile.mlxEndpoint` in `config.go`) applies precedence: a named entry in `mlx_hosts` (port baked into the value) wins, then the legacy `mlx_host` combined with the model's per-entry `port`, then an error.

- [ ] **Step 2: Update the builder options list in "Adding / editing models"**

Find the builder option list:

> `newModel(name, litellmKey, opts...)` with functional options:
>   `modelWithInput`, `modelWithOutput`, `modelWithCache`, `modelWithDeepinfra`, `modelWithVertexai`, `modelWithMlx16`, `modelWithVertexLocation`, `modelWithVision`, `modelWithMultimodalImageSupport`.

Replace with:

> `newModel(name, litellmKey, opts...)` with functional options:
>   `modelWithInput`, `modelWithOutput`, `modelWithCache`, `modelWithDeepinfra`, `modelWithVertexai`, `modelWithMlx`, `modelWithMlxHost`, `modelWithVertexLocation`, `modelWithVision`, `modelWithMultimodalImageSupport`.

- [ ] **Step 3: Update the provider wiring section**

Find the `providerMlx16` bullet:

> - `providerMlx16` (local vLLM/MLX servers): model prefixed `hosted_vllm/`, `api_base` set to `http://<mlx_host>:<port>/v1` where `mlx_host` comes from `config.json` and the port from `modelWithMlx16(name, port)` (used ports: 8000, 8001).

Replace with:

> - `providerMlx16` (local vLLM/MLX servers): model prefixed `hosted_vllm/`, `api_base` resolved by `configFile.mlxEndpoint(m.mlxHost, m.port)`. A named host key (set via `modelWithMlxHost(key)`) looks up `mlx_hosts[key]` and bakes the port into the URL; with no matching named host it falls back to the legacy `mlx_host` + the port from `modelWithMlx(name, port)`. If neither is configured, `litellmModels` returns an error instead of emitting a broken `http://:port/v1`.

- [ ] **Step 4: Add a note to the pricing command description**

Find the `pricing` bullet in section 2:

> - `pricing`: iterates `myModels` and writes an aligned table via `text/tabwriter` (columns: model name, litellm key, provider, input/output/cache in USD per 1M tokens). Uses `modelProvider.String()` and `pricing.Decimal.String()`. Supports `--sort input|output|cache|name` (default `input`) and `--desc`; sorts on a **copy** of `myModels` via `sort.SliceStable` so the shared registry (and thus the `litellm` output order) is never mutated.

Replace with:

> - `pricing`: iterates `myModels` and writes an aligned table via `text/tabwriter` (columns: model name, litellm key, provider, input/output/cache in USD per 1M tokens). Uses `modelProvider.String()` and `pricing.Decimal.String()`; for MLX models the public host key (from `modelWithMlxHost`) is appended to the provider cell (e.g. `vllm-mlx/gemma`) so the private IP from `config.json` is never shown. Supports `--sort input|output|cache|name` (default `input`) and `--desc`; sorts on a **copy** of `myModels` via `sort.SliceStable` so the shared registry (and thus the `litellm` output order) is never mutated.

- [ ] **Step 5: Commit with caveman-commit**

```
docs(AGENTS): document named MLX hosts and resolver

Covers mlx_hosts config key, mlxEndpoint precedence, the
modelWithMlx rename + modelWithMlxHost selector, and that pricing
shows the host key rather than the private IP.
```

```bash
mktemp
# Write tool: save the message above to the returned /tmp/tmp.XXXXXX
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

## Task 7: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Format the touched files**

Run: `gofmt -w config.go models.go main.go models_test.go`
Expected: no output (already formatted). Note: the pre-existing `litellm.go` stub is the only file `gofmt` flags repo-wide; it is intentionally not touched here.

- [ ] **Step 2: Full build, test, vet**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: build success, all tests pass, vet clean.

- [ ] **Step 3: Smoke-test all three commands**

Run:
```bash
cp config.example.json config.json
go run . opencode > /dev/null
go run . litellm  > /dev/null
go run . pricing
rm config.json
```
Expected: all succeed; `pricing` MLX rows show `vllm-mlx/gemma` and `vllm-mlx/minicpm`; `litellm` resolves named hosts (no errors).

- [ ] **Step 4: Confirm no legacy references remain**

Run: `grep -rn "modelWithMlx16" . || true`
Expected: no matches.

- [ ] **Step 5: Final commit if gofmt changed anything (otherwise skip)**

If `gofmt` in Step 1 modified any file, commit:

```
style: gofmt touched files for named MLX hosts
```

```bash
mktemp
# Write tool: save the message above to the returned /tmp/tmp.XXXXXX
jj describe --stdin < "/tmp/tmp.XXXXXX"
rm "/tmp/tmp.XXXXXX"
jj new
```

---

## Self-Review

**1. Spec coverage** (issue `docs/issues/1.json`):
- Multiple named MLX hosts in config (`mlx_hosts` map) → Task 1. ✓
- Reference a host by key without exposing IPs (`modelWithMlxHost` + resolver) → Tasks 1, 2. ✓
- `modelWithMlx16(...)` still accepts the model/port; host selector added → Task 2. ✓
- Resolve to the actual URL at runtime → Task 3. ✓
- Wire into pricing (`--sort` / `modelProvider.String()`) → Task 4 (provider column shows key; `--sort` keys unchanged — they still operate on numeric pricing/name columns, no new key needed). ✓
- Backward compatible (legacy `mlx_host` fallback) → Task 1 resolver + kept `port` arg. ✓
- Rename `modelWithMlx16` → `modelWithMlx` → Task 2. ✓

**2. Placeholder scan:** No TBD/TODO; every code step contains the complete final code. The only "skip if unchanged" is the conditional gofmt commit in Task 7, which is intentional.

**3. Type consistency:** `mlxEndpoint(key string, port int) (string, bool)` — used identically in `config.go` (Task 1), `litellmModels` (Task 3), and tested in `TestMlxEndpoint`/`TestLitellmModelsMlxNamedHost` (Tasks 1, 3). Field `mlxHost string` (Task 2) is read in `litellmModels` (`m.mlxHost`, Task 3) and the pricing loop (`m.mlxHost`, Task 4). `litellmModels` signature `([]litellm.ModelParams, error)` matches the `main.go` call site (`params, err := litellmModels(cfg)`, Task 3) and both tests. `modelWithMlx(name, port)` / `modelWithMlxHost(key)` match the registry call sites (Task 2). No mismatched names.
