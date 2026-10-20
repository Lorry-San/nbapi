package controller

import (
	"testing"

	"github.com/Lorry-San/nbapi/model"
	"github.com/stretchr/testify/assert"
)

// The selector lists models in the operator's row order. Abilities merely say what is
// routable, so a channel that returns them alphabetically must not reshuffle the catalog.
func TestOrderCatalogNamesFollowsPolicyRows(t *testing.T) {
	policies := []model.LightcodeGroupModel{
		{Model: "gemini-3.8-flash"},
		{Model: "gemini-3.5-flash-lite"},
		{Model: "gemini-3-flash"},
	}
	// Alphabetical, which is what the abilities query returns.
	routed := []string{"gemini-3-flash", "gemini-3.5-flash-lite", "gemini-3.8-flash"}

	assert.Equal(t, []string{"gemini-3.8-flash", "gemini-3.5-flash-lite", "gemini-3-flash"},
		orderCatalogNames(policies, routed))
}

// A model a channel routes but the operator has not configured keeps working; it is listed
// after the configured rows instead of disappearing.
func TestOrderCatalogNamesAppendsUnconfiguredRoutedModels(t *testing.T) {
	policies := []model.LightcodeGroupModel{{Model: "claude-opus-5"}}
	routed := []string{"gpt-6.1-sol", "claude-opus-5"}

	assert.Equal(t, []string{"claude-opus-5", "gpt-6.1-sol"}, orderCatalogNames(policies, routed))
}

// A configured row with no live channel is skipped, so the catalog never advertises a model
// that cannot be served.
func TestOrderCatalogNamesSkipsRowsThatAreNotRoutable(t *testing.T) {
	policies := []model.LightcodeGroupModel{
		{Model: "claude-opus-5"},
		{Model: "gpt-6-luna"},
		{Model: "claude-sonnet-5"},
	}
	routed := []string{"claude-sonnet-5", "claude-opus-5"}

	assert.Equal(t, []string{"claude-opus-5", "claude-sonnet-5"}, orderCatalogNames(policies, routed))
}

func TestOrderCatalogNamesHandlesEmptyInputs(t *testing.T) {
	assert.Empty(t, orderCatalogNames(nil, nil))
	assert.Equal(t, []string{"m"}, orderCatalogNames([]model.LightcodeGroupModel{{Model: "m"}}, []string{"m"}))
	assert.Equal(t, []string{"m"}, orderCatalogNames(nil, []string{"m"}))
}
