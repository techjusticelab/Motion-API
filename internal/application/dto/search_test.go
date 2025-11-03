package dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSearchDocumentsRequestValidate(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	req := &SearchDocumentsRequest{
		Query:         "motion",
		MinConfidence: 0.5,
		Page:          1,
		PageSize:      20,
		FromDate:      &now,
		ToDate:        &later,
		SortBy:        "created_at",
		SortOrder:     "desc",
	}
	assert.NoError(t, req.Validate())

	req.Page = 0
	assert.Error(t, req.Validate())
}

func TestSearchDocumentsRequestValidateInvalidConfidence(t *testing.T) {
	req := &SearchDocumentsRequest{
		Query:         "motion",
		MinConfidence: 1.5,
		Page:          1,
		PageSize:      20,
		SortBy:        "created_at",
		SortOrder:     "desc",
	}
	assert.Error(t, req.Validate())
}

func TestSearchDocumentsRequestValidateInvalidDateRange(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)

	req := &SearchDocumentsRequest{
		Query:         "motion",
		MinConfidence: 0.5,
		Page:          1,
		PageSize:      20,
		FromDate:      &now,
		ToDate:        &earlier,
		SortBy:        "created_at",
		SortOrder:     "desc",
	}
	assert.Error(t, req.Validate())
}

func TestSearchDocumentsRequestValidateInvalidSort(t *testing.T) {
	req := &SearchDocumentsRequest{
		Query:         "motion",
		MinConfidence: 0.5,
		Page:          1,
		PageSize:      20,
		SortBy:        "invalid_field",
		SortOrder:     "invalid_order",
	}
	assert.Error(t, req.Validate())
}
