package litellm

import (
	"encoding/json"
	"io"
	"models/pricing"
)

type config struct {
	ModelList       modelList       `json:"model_list"`
	LitellmSettings litellmSettings `json:"litellm_settings"`
}

type modelList []model

type model struct {
	ModelName     string        `json:"model_name"`
	LitellmParams litellmParams `json:"litellm_params"`
}

// See pricing in litellm/v1.83.14/types/utils.py, CustomPricingLiteLLMParams.
type litellmParams struct {
	Model              string          `json:"model"`
	VertexProject      string          `json:"vertex_project,omitempty"`
	VertexAiLocation   string          `json:"vertex_ai_location,omitempty"`
	ModelInfo          *modelInfo      `json:"model_info,omitempty"`
	ApiKey             string          `json:"api_key,omitempty"`
	ApiBase            string          `json:"api_base,omitempty"`
	InputCostPerToken  pricing.Decimal `json:"input_cost_per_token"`
	OutputCostPerToken pricing.Decimal `json:"output_cost_per_token"`
}

type modelInfo struct {
	SupportsVision bool `json:"supports_vision,omitempty"`
}

type litellmSettings struct {
	DropParams bool   `json:"drop_params"`
	Callbacks  string `json:"proxy_handler_instance"`
}

type ModelParams struct {
	name     string
	model    string
	vertexai vertexai
	vision   bool
	apiKey   string
	apiBase  string
	pricing  pricingParams
}

type pricingParams struct {
	input  pricing.Decimal
	output pricing.Decimal
}

type vertexai struct {
	project  string
	location string
}

type ModelOpt func(ModelParams) ModelParams

func NewModel(name string, opts ...ModelOpt) ModelParams {
	var p ModelParams
	p.name = name
	for _, opt := range opts {
		p = opt(p)
	}
	return p
}

func ConfigJSON(w io.Writer, models []ModelParams) error {
	var config config
	config.LitellmSettings = litellmSettings{
		DropParams: true,
		Callbacks:  "custom_callbacks.proxy_handler_instance",
	}
	for _, m := range models {
		var info *modelInfo
		if m.vision {
			info = &modelInfo{
				SupportsVision: true,
			}
		}
		config.ModelList = append(config.ModelList, model{
			ModelName: m.name,
			LitellmParams: litellmParams{
				Model:              m.model,
				VertexProject:      m.vertexai.project,
				VertexAiLocation:   m.vertexai.location,
				ModelInfo:          info,
				ApiKey:             m.apiKey,
				ApiBase:            m.apiBase,
				InputCostPerToken:  m.pricing.input.PerToken(),
				OutputCostPerToken: m.pricing.output.PerToken(),
			},
		})
	}
	return json.NewEncoder(w).Encode(config)
}

func ModelWithVllm(model, base string) ModelOpt {
	return func(mp ModelParams) ModelParams {
		mp.model = "hosted_vllm/" + model
		mp.apiBase = base
		return mp
	}
}

func ModelWithDeepinfra(model string) ModelOpt {
	return func(mp ModelParams) ModelParams {
		mp.model = "deepinfra/" + model
		return mp
	}
}

func ModelWithVertexAi(project, model string) ModelOpt {
	return func(mp ModelParams) ModelParams {
		mp.vertexai.project = project
		mp.model = "vertex_ai/" + model
		return mp
	}
}

func ModelWithVertexLocation(location string) ModelOpt {
	return func(mp ModelParams) ModelParams {
		mp.vertexai.location = location
		return mp
	}
}

func ModelWithVision() ModelOpt {
	return func(mp ModelParams) ModelParams {
		mp.vision = true
		return mp
	}
}

func ModelWithApikey(key string) ModelOpt {
	return func(mp ModelParams) ModelParams {
		mp.apiKey = key
		return mp
	}
}

func ModelWithPricing(input, output pricing.Decimal) ModelOpt {
	return func(mp ModelParams) ModelParams {
		mp.pricing.input = input
		mp.pricing.output = output
		return mp
	}
}
