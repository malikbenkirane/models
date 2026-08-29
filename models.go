package main

import (
	"fmt"
	"models/litellm"
	"models/pricing"
)

var myModels = []modelDescription{
	{
		name:     "GLM-5.1 [deepinfra.com]",
		litellm:  "di-glm-5.1",
		input:    pricing.NewDecimal(1, 1, 0, 5), // 1.05
		output:   pricing.NewDecimal(1, 3, 5),    // 3.5
		cache:    pricing.NewDecimal(0, 2, 0, 5), // 0.205
		provider: providerDeepinfra,
		model:    "zai-org/GLM-5.1",
	},
	{
		name:     "DeepSeek v4 Flash [deepinfra.com]",
		litellm:  "di-deepseek-v4-flash",
		input:    pricing.NewDecimal(0, 1),    // .1
		output:   pricing.NewDecimal(0, 2),    // .2
		cache:    pricing.NewDecimal(0, 0, 2), // .02
		provider: providerDeepinfra,
		model:    "deepseek-ai/DeepSeek-V4-Flash",
	},
	{
		name:     "GLM-4.7-Flash [deepinfra.com]",
		litellm:  "di-glm-4.7-flash",
		input:    pricing.NewDecimal(0, 0, 6), // .06
		output:   pricing.NewDecimal(0, 4),    // .4
		cache:    pricing.NewDecimal(0, 0, 1), //.01
		provider: providerDeepinfra,
		model:    "zai-org/GLM-4.7-Flash",
	},
	newModel("GLM-5 [deepinfra.com]", "di-glm-5",
		modelWithDeepinfra("zai-org/GLM-5"),
		modelWithInput(0, 6),        // 0.6
		modelWithOutput(1, 2, 0, 8), // 2.08
		modelWithCache(0, 1, 2)),    // 0.12
	newModel("Claude Opus 4.8 [deepinfra.com]", "di-opus-4.8",
		modelWithDeepinfra("anthropic/claude-opus-4-8"),
		modelWithInput(1, 5),     // 5.
		modelWithOutput(2, 2, 5), // 25.0
		modelWithMultimodalImageSupport()),
	newModel("Gemini 3.5 Flash [deepinfra.com]", "di-gemini-3.5-flash",
		modelWithDeepinfra("google/gemini-3.5-flash"),
		modelWithInput(1, 1, 5), // 1.5
		modelWithOutput(1, 9),   // 9.0
		modelWithMultimodalImageSupport()),
	newModel("Mimo V2.5 Pro [deepinfra.com]", "di-mimo-v2.5-pro",
		modelWithDeepinfra("XiaomiMiMo/MiMo-V2.5-Pro"),
		modelWithCache(0, 2),   // 0.2
		modelWithInput(1, 1),   // 1.0
		modelWithOutput(1, 3)), // 3.0
	newModel("Kimi-K2.6 [deepinfra.com]", "di-kimi-k2.6",
		modelWithDeepinfra("moonshotai/Kimi-K2.6"),
		modelWithInput(0, 7, 5),  // 0.75
		modelWithOutput(1, 3, 5), // 3.5
		modelWithCache(0, 1, 5)), // 0.15
	newModel("GLM-5 [Vertex AI]", "glm-5",
		modelWithVertexai("zai-org/glm-5-maas"),
		modelWithInput(1, 1),
		modelWithOutput(1, 3, 2),
		modelWithCache(0, 1)),
	newModel("Claude 4.6 [Vertex AI]", "opus-4.6",
		modelWithVertexai("claude-opus-4-6"),
		modelWithVision(),
		modelWithMultimodalImageSupport(),
		modelWithVertexLocation("global"),
		modelWithInput(1, 5),
		modelWithOutput(2, 2, 5)),
	newModel("GLM-5.2 [deepinfra.com]", "di-glm-5.2",
		modelWithDeepinfra("zai-org/GLM-5.2"),
		modelWithCache(0, 0, 9, 1),
		modelWithInput(0, 4, 8, 8),
		modelWithOutput(1, 1, 5, 6)),
	newModel("GLM-5.3-Flash [deepinfra.com]", "di-glm-5.3-flash",
		modelWithDeepinfra("zai-org/GLM-5.3-Flash"),
		modelWithCache(0, 0, 3),
		modelWithInput(0, 1, 5),
		modelWithOutput(0, 5)),
	newModel("Qwen/Qwen3-Max-Thinking [deepinfra.com]", "di-q3-max-thinking",
		modelWithDeepinfra("Qwen/Qwen3-Max-Thinking"),
		modelWithInput(1, 1, 2),
		modelWithOutput(1, 6)),
	newModel("deepseek-ai/DeepSeek-V4-Pro [deepinfra.com]", "di-deepseek-v4-pro",
		modelWithCache(0, 1),
		modelWithInput(1, 1, 3),
		modelWithOutput(1, 2, 6),
		modelWithDeepinfra("deepseek-ai/DeepSeek-V4-Pro")),
	newModel("claude-sonnet-5 [deepinfra.com]", "di-claude-sonnet-5",
		modelWithDeepinfra("anthropic/claude-sonnet-5"),
		modelWithInput(1, 2),
		modelWithOutput(2, 1, 0)),
	newModel("Qwen 3 Max Thinking [deepinfra.com]", "di-qwen3-max-thinking",
		modelWithDeepinfra("Qwen/Qwen3-Max-Thinking"),
		modelWithCache(0, 2, 4),
		modelWithInput(1, 1, 2),
		modelWithOutput(1, 6)),
	newModel("Qwen 3 Max [deepinfra.com]", "di-qwen3-max",
		modelWithDeepinfra("Qwen/Qwen3-Max"),
		modelWithCache(0, 2, 4),
		modelWithInput(1, 1, 2),
		modelWithOutput(1, 6)),
	newModel("Kimi K3 [deepinfra.com]", "di-kimi-k3",
		modelWithCache(0, 2, 8, 5),
		modelWithInput(1, 2, 8, 5),
		modelWithOutput(2, 1, 4, 2, 5)),
	newModel("Gemma 4 31B-it [deepinfra.com]", "gemma-4-31b-it",
		modelWithDeepinfra("google/gemma-4-31B-it"),
		modelWithVision(),
		modelWithInput(0, 1, 3),
		modelWithOutput(0, 3, 8)),
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
}

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

type modelProvider int

const (
	providerDeepinfra modelProvider = iota
	providerVertexAi
	providerMlx16
)

func (p modelProvider) String() string {
	switch p {
	case providerDeepinfra:
		return "deepinfra"
	case providerVertexAi:
		return "vertex-ai"
	case providerMlx16:
		return "vllm-mlx"
	default:
		return "unknown"
	}
}

type modelOpt func(modelDescription) modelDescription

func modelWithInput(offset uint, digits ...uint8) modelOpt {
	return func(md modelDescription) modelDescription {
		md.input = pricing.NewDecimal(offset, digits...)
		return md
	}
}
func modelWithOutput(offset uint, digits ...uint8) modelOpt {
	return func(md modelDescription) modelDescription {
		md.output = pricing.NewDecimal(offset, digits...)
		return md
	}
}
func modelWithCache(offset uint, digits ...uint8) modelOpt {
	return func(md modelDescription) modelDescription {
		md.cache = pricing.NewDecimal(offset, digits...)
		return md
	}
}
func modelWithMultimodalImageSupport() modelOpt {
	return func(md modelDescription) modelDescription {
		md.modalities = &modalities{
			Input:  []modality{modalText, modalImage},
			Output: []modality{modalText},
		}
		md.attachment = true
		return md
	}
}
func modelWithMlx(name string, port int) modelOpt {
	return func(md modelDescription) modelDescription {
		md.provider = providerMlx16
		md.port = port
		md.model = name
		return md
	}
}
func modelWithMlxHost(key string) modelOpt {
	return func(md modelDescription) modelDescription {
		md.mlxHost = key
		return md
	}
}
func modelWithDeepinfra(name string) modelOpt {
	return func(md modelDescription) modelDescription {
		md.provider = providerDeepinfra
		md.model = name
		return md
	}
}
func modelWithVertexai(name string) modelOpt {
	return func(md modelDescription) modelDescription {
		md.provider = providerVertexAi
		md.model = name
		return md
	}
}
func modelWithVertexLocation(location string) modelOpt {
	return func(md modelDescription) modelDescription {
		md.location = location
		return md
	}
}

func modelWithVision() modelOpt {
	return func(md modelDescription) modelDescription {
		md.vision = true
		return md
	}
}

func newModel(name, litellm string, opts ...modelOpt) modelDescription {
	description := modelDescription{
		input:   pricing.NewDecimal(0),
		output:  pricing.NewDecimal(0),
		cache:   pricing.NewDecimal(0),
		litellm: litellm,
		name:    name,
	}
	for _, opt := range opts {
		description = opt(description)
	}
	return description
}
