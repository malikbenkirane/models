# models

A Go CLI that turns one in-code model registry (`myModels` in `models.go`) into the configs my LLM tooling consumes: a JSON map for opencode and a `config.yaml` for the LiteLLM proxy. I run both daily alongside Crush, and this binary keeps their model lists and pricing in sync from a single source of truth.

`myModels` is also a running log of the models I've picked up and actually tried, the ones that earned a slot in my daily rotation. They span DeepInfra and Vertex AI in the cloud, plus a couple of local vLLM/MLX servers. If you're shopping for models, browse the registry and steal whatever looks useful.

## See what I'm running

The `pricing` command prints the whole collection, sorted however you like:

```sh
go run . pricing                # full table, sorted by input price (default)
go run . pricing --sort output  # sort by: input|output|cache|name
go run . pricing --desc         # highest first
```

That's the easiest way to see what's here. Every entry maps to a model I've spent real time with.

## Install

```sh
go install .
```

Or build a local binary:

```sh
go build -o models .
```

## Run

```sh
go run . opencode               # JSON config for the opencode client (stdout)
go run . litellm                # LiteLLM proxy config.yaml (stdout)
go run . pricing                # pricing table: name, key, provider, $/Mtok
go run . pricing --sort output  # sort by: input|output|cache|name (default input)
go run . pricing --desc         # sort descending
```

### Configuration

`litellm` reads `config.json` (gitignored) for deployment-specific values:

```json
{
  "vertex_project": "your-gcp-project-id",
  "mlx_host": "192.168.1.100"
}
```

Copy `config.example.json` to `config.json` and fill in your values. `opencode` and `pricing` don't need it; if `config.json` is absent `litellm` still runs with empty values.

## Make it yours

This registry is personal, but the tool isn't. Fork it and swap in the models you care about, then point opencode and LiteLLM at the output. Add entries using the builder pattern in `myModels` (`models.go`):

```go
newModel("Display Name", "litellm-key",
    modelWithDeepinfra("provider/model-id"),   // or modelWithVertexai / modelWithMlx
    modelWithInput(1, 2),                       // $2/Mtok input  (offset, digits...)
    modelWithOutput(2, 1, 5),                   // $15/Mtok output
    modelWithCache(0, 1, 2),                    // $0.12/Mtok cache
    modelWithVision(),                          // litellm: supports_vision flag
    modelWithMultimodalImageSupport(),          // opencode: image input + attachment
)
```

### Providers

| Option | Provider | Details |
|--------|----------|---------|
| `modelWithDeepinfra(id)` | DeepInfra | API key via `DEEPINFRA_API_KEY` env var |
| `modelWithVertexai(id)` | Vertex AI | Project from `config.json`; add `modelWithVertexLocation("global")` if needed |
| `modelWithMlx(id, port)` | local vLLM/MLX | Endpoint resolved from `config.json` via `mlx_hosts` (named) or legacy `mlx_host` + `port`; tag with `modelWithMlxHost(key)` to select a named host |

### Pricing

`modelWithInput`/`modelWithOutput`/`modelWithCache` use `pricing.NewDecimal(offset, digits...)`:

- `offset` = number of integer digits
- `digits` = all significant digits concatenated
- Prices are USD per 1 million tokens

Examples: `NewDecimal(1, 2)` = `2`, `NewDecimal(0, 6)` = `0.6`, `NewDecimal(2, 2, 5)` = `25`.

### Vision vs multimodality

These target different outputs: a model that handles images in both configs needs both.

- `modelWithVision()` → sets `supports_vision` in the litellm output only
- `modelWithMultimodalImageSupport()` → sets modalities + attachment in the opencode output only

## Known issues

- `modelWithVision()` does not produce working litellm vision support. The `supports_vision` flag is emitted in the litellm config but the proxy does not honor it (tested with the Crush client only). Use `modelWithMultimodalImageSupport()` for opencode; litellm vision support is unverified and may not work.

## Testing

```sh
go test ./...
```

---

I put this together because keeping opencode and LiteLLM in lockstep by hand was getting tedious, and because I wanted a single honest record of the models I've actually run, the ones that made it into my workflow. PRs welcome if you spot a price that's drifted or a model worth adding.
