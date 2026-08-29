# AGENTS.md

Go CLI that generates LLM-gateway configuration from a single in-code model registry.
The same registry (`myModels` in `models.go`) is projected into two output formats:

- `opencode` — a JSON map of models with per-million-token costs (consumed by the opencode client).
- `litellm` — a LiteLLM proxy `config.yaml` (YAML) with a `model_list` and `litellm_settings` (consumed by the LiteLLM proxy in the parent `karen` project, which runs with `custom_callbacks.py`).
- `pricing` — a human-readable table of all models with provider and per-million-token pricing (read-only display; shares the same `myModels` registry).

The module path is `models` and it is part of the larger `karen` repo. The git root is the parent `karen` directory.

Deployment-specific values that must not be committed live in `config.json` (gitignored): `vertex_project` (GCP project ID for Vertex AI), `mlx_host` (legacy single host of the local vLLM/MLX servers), and `mlx_hosts` (a map of `name -> "host:port"` for multiple named MLX servers). `config.example.json` is the committed template. Only the `litellm` command reads `config.json` (via `loadConfig` in `config.go`); if the file is absent the values default to empty strings, so `opencode` and `pricing` work without it. MLX endpoint resolution (`configFile.mlxEndpoint` in `config.go`) applies precedence: a named entry in `mlx_hosts` (port baked into the value) wins, then the legacy `mlx_host` combined with the model's per-entry `port`, then an error.

## Commands

There is no Makefile and no CI in this module; use plain `go` commands.

```sh
go build ./...                 # build (clean = success)
go test ./...                   # run tests (only the main package has tests)
go vet ./...                    # vet (clean)
gofmt -w .                      # format; litellm.go is currently unformatted

go run . opencode               # print opencode JSON config to stdout
go run . litellm                # print LiteLLM YAML config to stdout
go run . pricing                # print model pricing table (name, key, provider, $/Mtok)
go run . pricing --sort output  # sort by a column: input|output|cache|name (default input)
go run . pricing --desc         # sort descending (ties keep registry order via sort.SliceStable)
```

Toolchain: Go 1.26.2 (matches `go.mod`). Dependencies: `github.com/spf13/cobra` (CLI) and `sigs.k8s.io/yaml` (JSON→YAML). No network calls at runtime; the binary reads the hardcoded registry and `config.json`, then writes config to stdout.

## Architecture & data flow

1. `models.go` defines `myModels []modelDescription` — the single source of truth for every model (name, provider, upstream id, pricing, vision/multimodal flags, port for local servers).
2. `main.go` wires three cobra subcommands:
   - `opencode`: iterates `myModels`, builds a `models` map (types in `opencode.go`) keyed by the litellm model name, marshals to indented JSON.
   - `litellm`: calls `loadConfig()` (in `config.go`) then `litellmModels(cfg)` (in `models.go`) to convert `myModels` → `[]litellm.ModelParams`, then `litellm.ConfigJSON` marshals to JSON, then `sigs.k8s.io/yaml.JSONToYAML` converts that JSON to YAML.
   - `pricing`: iterates `myModels` and writes an aligned table via `text/tabwriter` (columns: model name, litellm key, provider, input/output/cache in USD per 1M tokens). Uses `modelProvider.String()` and `pricing.Decimal.String()`; for MLX models the public host key (from `modelWithMlxHost`) is appended to the provider cell (e.g. `vllm-mlx/gemma`) so the private IP from `config.json` is never shown. Supports `--sort input|output|cache|name` (default `input`) and `--desc`; sorts on a **copy** of `myModels` via `sort.SliceStable` so the shared registry (and thus the `litellm` output order) is never mutated.
3. `pricing/decimal.go` provides `Decimal`, a hand-rolled decimal type whose `MarshalJSON` emits a bare JSON **number** (not a string). Prices are stored as **USD per 1 million tokens**; `Decimal.PerToken()` divides by 10^6 for the litellm output (which expects per-token costs). `Decimal.Less` provides pure numeric ordering (ignoring `perToken`) for the `pricing` command's `--sort` flag; `Decimal.String()` implements `fmt.Stringer` for the table cells.

### Package map

| Path | Package | Role |
|------|---------|------|
| `.` (root) | `main` | CLI (`main.go`), model registry + builders + `litellmModels()` (`models.go`), config loader (`config.go`), opencode output types (`opencode.go`) |
| `pricing` | `pricing` | `Decimal` type: `NewDecimal`, `PerToken`, custom `MarshalJSON` |
| `litellm` | `litellm` | LiteLLM config structs, `ModelParams` builder, `ConfigJSON`, `ModelWith*` options |
| `litellm/logs/request` | `request` | Go structs mirroring LiteLLM request-log JSON (`Log`, `CallMetadata`, `HTTPHeaders`, …). **Not imported anywhere** in the current build — standalone definitions, likely for future log parsing. Adding a feature here will not affect output until wired in. |

`litellm.go` (root) is an empty stub (`package main` + blank lines) and is the only file `gofmt` flags. It is not dead code to delete blindly — it occupies the `main` package — but has no content.

## Adding / editing models

The registry uses **two coexisting styles**; prefer the builder for new entries.

- Legacy (first 3 entries in `myModels`): plain `modelDescription{...}` struct literals.
- Builder (all later entries): `newModel(name, litellmKey, opts...)` with functional options:
  `modelWithInput`, `modelWithOutput`, `modelWithCache`, `modelWithDeepinfra`, `modelWithVertexai`, `modelWithMlx`, `modelWithMlxHost`, `modelWithVertexLocation`, `modelWithVision`, `modelWithMultimodalImageSupport`.

`newModel` zero-initializes `input`/`output`/`cache` to `NewDecimal(0)` (= `0`), so omitting a price yields `0`, not a missing field.

### Vision vs multimodality — they target different outputs

This is the most subtle part of the registry:

- `modelWithVision()` sets `vision=true`, which **only** affects the litellm output (`model_info.supports_vision`). It has **no effect** on the opencode output. Caveat: the flag is emitted into the config but the LiteLLM proxy does not currently honor it (see README "Known issues"), so it does not by itself enable working vision support.
- `modelWithMultimodalImageSupport()` sets `modalities` (input: text+image, output: text) and `attachment=true`, which **only** affect the opencode output. It does **not** set `vision`, so litellm won't mark `supports_vision`.

A model that handles images in **both** configs needs **both** options (see "Claude 4.6 [Vertex AI]"). Using only one silently leaves the other config without the capability. Existing entries are intentionally inconsistent here (e.g. "Gemma 4 31B-it" has `modelWithVision()` only → no opencode modalities; "Claude Opus 4.8" has `modelWithMultimodalImageSupport()` only → no litellm vision flag). Match the intent of the specific model when editing.

### Provider wiring (`litellmModels`)

Provider selection happens in `litellmModels()` (`models.go`) and maps to litellm `ModelWith*` calls:

- `providerDeepinfra`: model prefixed `deepinfra/`, api_key = `os.environ/DEEPINFRA_API_KEY` (litellm env-reference syntax, not a literal key).
- `providerVertexAi`: model prefixed `vertex_ai/`, `vertex_project` read from `config.json` (`vertex_project` key), optional `vertex_ai_location` via `modelWithVertexLocation`.
- `providerMlx16` (local vLLM/MLX servers): model prefixed `hosted_vllm/`, `api_base` resolved by `configFile.mlxEndpoint(m.mlxHost, m.port)`. A named host key (set via `modelWithMlxHost(key)`) looks up `mlx_hosts[key]` and bakes the port into the URL; with no matching named host it falls back to the legacy `mlx_host` + the port from `modelWithMlx(name, port)`. If neither is configured, `litellmModels` returns an error instead of emitting a broken `http://:port/v1`.

All providers always emit `ModelWithPricing(input, output)`; vision is appended only when `m.vision` is true.

### Cache pricing is opencode-only

The opencode command sets `cache_read` and `cache_write` both to `m.cache`. The litellm config structs (`InputCostPerToken`/`OutputCostPerToken` only) **have no cache fields**, so cache pricing is computed and stored but **dropped from the litellm output**. Don't expect cache costs to round-trip into the YAML.

## `pricing.Decimal` semantics (important)

`NewDecimal(offset, digits...)` builds a decimal from raw digits:

- `offset` = the digit index at which the decimal point sits = number of integer digits.
- `digits` = **all** significant digits (integer + fractional parts concatenated), each a single decimal digit `0–9`.
- `offset == 0` ⇒ value < 1, emitted as `0.` + digits.
- `offset >= len(digits)` ⇒ pure integer, no decimal point emitted.

Examples (from `models.go` comments and `models_test.go`):

| Call | Value |
|------|-------|
| `NewDecimal(0, 1)` | `0.1` |
| `NewDecimal(0, 0, 2)` | `0.02` |
| `NewDecimal(0)` | `0` |
| `NewDecimal(1, 2, 3)` | `2.3` |
| `NewDecimal(2, 2, 5)` | `25` |
| `NewDecimal(1, 5)` | `5` |

Gotchas:

- **Digits must be 0–9.** The variadic type is `uint8`, but `MarshalJSON` does `FormatUint(...)[0]` — it takes only the first byte of the stringified digit, so a value ≥ 10 (e.g. `12`) silently becomes `1`. There is no validation.
- **`PerToken()` assumes `offset ≤ 6`.** It emits `0.` followed by `6 - offset` zeros, then the digits. Since `offset` is `uint`, `6 - offset` underflows to a huge value when `offset > 6`, causing the zero-padding loop to run ~10^19 times (effectively a hang/OOM). Real model prices keep `offset` small (≤ 2), so this is latent, not active — but keep it in mind for expensive models.
- `MarshalJSON` always returns `nil` error and emits a **bare number token** (no quotes), so prices appear as numeric literals in both JSON and YAML.

## Output-format gotchas

- **Why JSON→YAML instead of direct `yaml.Marshal`:** the litellm command encodes to JSON (`litellm.ConfigJSON`) then converts with `sigs.k8s.io/yaml.JSONToYAML`. `sigs.k8s.io/yaml` marshals via the JSON representation, so it **preserves struct field order** (as defined by `json` tags). Switching to `gopkg.in/yaml.v2` directly would **alphabetize keys** and churn the generated `config.yaml` diffs. Keep the JSON→YAML path.
- The opencode output is a `map[string]model` **keyed by `m.litellm`** (the litellm model name). If two registry entries ever share the same litellm key, the opencode map silently keeps the **last** one. The litellm output is a `model_list` slice, so it does **not** dedupe — duplicate keys produce duplicate entries. The registry currently has upstream models referenced by more than one entry under different litellm keys (e.g. `Qwen/Qwen3-Max-Thinking` appears as both `di-q3-max-thinking` and `di-qwen3-max-thinking` — same input/output price, but only the latter specifies cache pricing); this is intentional, not a bug.
- In `main.go`'s litellm branch, `b` (a `bytes.Buffer`) is shadowed by a new `b []byte` from `yaml.JSONToYAML(b.Bytes())` inside an inner block — valid but easy to misread.
- The litellm output hardcodes `litellm_settings`: `drop_params: true` and `proxy_handler_instance: "custom_callbacks.proxy_handler_instance"` (a reference to the parent project's `custom_callbacks.py`).

## Testing

- `models_test.go` is table-driven (`t.Run` with empty names) and covers only `pricing.Decimal` — `MarshalJSON` (both the per-million form and the `.PerToken()` form) and `Less` (numeric ordering). It lives in package `main` and is the **only** package with tests (`pricing`, `litellm`, `litellm/logs/request` report `[no test files]`).
- When changing `Decimal` behavior, update/extend these cases; they are the de facto spec for the encoding and ordering.
- The `MarshalJSON` tests assert exact string output via `t.Fatalf("\nexp: %q\ngot: %q", ...)`; the `Less` test asserts boolean ordering via `t.Fatalf("\nexp: %v\ngot: %v", ...)`.
