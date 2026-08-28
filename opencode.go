package main

import (
	"models/pricing"
)

type models map[string]model

type model struct {
	Cost       cost        `json:"cost"`
	Name       string      `json:"name"`
	Attachment bool        `json:"attachment,omitempty"`
	Modalities *modalities `json:"modalities,omitempty"`
}

type modality string

const (
	modalText  modality = "text"
	modalImage modality = "image"
)

type modalities struct {
	Input  []modality `json:"input"`
	Output []modality `json:"output"`
}

type cost struct {
	Input      pricing.Decimal `json:"input"`
	Output     pricing.Decimal `json:"output"`
	CacheRead  pricing.Decimal `json:"cache_read"`
	CacheWrite pricing.Decimal `json:"cache_write"`
}
