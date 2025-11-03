package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterNormalise(t *testing.T) {
	var nilFilter *Filter
	nilFilter.Normalise() // ensure nil receiver safe

	filter := &Filter{
		LegalTags: []string{" Motion ", "motion"},
		SortBy:    "",
	}

	filter.Normalise()

	assert.Equal(t, "created_at", filter.SortBy)
	assert.Equal(t, SortDesc, filter.SortOrder)
	assert.ElementsMatch(t, []string{"Motion"}, filter.LegalTags)
}

func TestFilterHasPagination(t *testing.T) {
	filter := Filter{}
	assert.False(t, filter.HasPagination())

	filter.Limit = 10
	assert.True(t, filter.HasPagination())
}
