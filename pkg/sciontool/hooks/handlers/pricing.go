/*
Copyright 2026 The Scion Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handlers

import (
	"strings"
	"sync"
)

// ModelPricing defines the cost per million tokens for a model.
type ModelPricing struct {
	InputPerMTok  float64 `json:"input_per_mtok" yaml:"input_per_mtok"`
	OutputPerMTok float64 `json:"output_per_mtok" yaml:"output_per_mtok"`
}

// defaultPricingTable contains pricing for known models (USD per million tokens).
// These can be overridden via SCION_MODEL_PRICING_FILE environment variable.
var defaultPricingTable = map[string]ModelPricing{
	"opus":              {InputPerMTok: 15.0, OutputPerMTok: 75.0},
	"sonnet":            {InputPerMTok: 3.0, OutputPerMTok: 15.0},
	"haiku":             {InputPerMTok: 0.25, OutputPerMTok: 1.25},
	"gemini-2.5-pro":    {InputPerMTok: 1.25, OutputPerMTok: 10.0},
	"gemini-2.5-flash":  {InputPerMTok: 0.15, OutputPerMTok: 0.60},
	"gpt-4o":            {InputPerMTok: 2.50, OutputPerMTok: 10.0},
	"gpt-4o-mini":       {InputPerMTok: 0.15, OutputPerMTok: 0.60},
	"o3":                {InputPerMTok: 2.50, OutputPerMTok: 10.0},
	"o4-mini":           {InputPerMTok: 1.10, OutputPerMTok: 4.40},
}

var (
	pricingOnce  sync.Once
	pricingTable map[string]ModelPricing
)

// LookupPricing returns the pricing for a model, matching by substring.
// Falls back to Sonnet pricing if no match found.
func LookupPricing(model string) ModelPricing {
	pricingOnce.Do(func() {
		pricingTable = defaultPricingTable
		// TODO: load overrides from SCION_MODEL_PRICING_FILE or settings
	})

	m := strings.ToLower(model)
	// Try exact key match first
	for pattern, pricing := range pricingTable {
		if strings.Contains(m, pattern) {
			return pricing
		}
	}
	// Default to Sonnet-class pricing
	return ModelPricing{InputPerMTok: 3.0, OutputPerMTok: 15.0}
}

// EstimateCost calculates the estimated cost in USD for token usage.
func EstimateCost(model string, inputTokens, outputTokens int64) float64 {
	p := LookupPricing(model)
	return (float64(inputTokens)*p.InputPerMTok + float64(outputTokens)*p.OutputPerMTok) / 1_000_000
}
