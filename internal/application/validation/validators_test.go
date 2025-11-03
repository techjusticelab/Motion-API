package validation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateFileName(t *testing.T) {
	require.NoError(t, ValidateFileName("document.pdf"))
	require.Error(t, ValidateFileName(" "))
	require.Error(t, ValidateFileName("document.exe"))
}

func TestValidateStoragePath(t *testing.T) {
	require.NoError(t, ValidateStoragePath("documents/2024/file.pdf"))
	require.Error(t, ValidateStoragePath(""))
	require.Error(t, ValidateStoragePath("invalid/path"))
}

func TestValidateConfidenceRange(t *testing.T) {
	require.NoError(t, ValidateConfidenceRange(0))
	require.NoError(t, ValidateConfidenceRange(0.5))
	require.Error(t, ValidateConfidenceRange(-0.1))
	require.Error(t, ValidateConfidenceRange(1.2))
}

func TestValidatePagination(t *testing.T) {
	require.NoError(t, ValidatePagination(0, 0))
	require.NoError(t, ValidatePagination(1, 10))
	require.Error(t, ValidatePagination(0, 10))
	require.Error(t, ValidatePagination(1, 0))
}

func TestValidateDateRange(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	require.NoError(t, ValidateDateRange(&now, &later))
	require.NoError(t, ValidateDateRange(nil, nil))
	require.Error(t, ValidateDateRange(&later, &now))
}

func TestValidateSort(t *testing.T) {
	require.NoError(t, ValidateSort("", ""))
	require.NoError(t, ValidateSort("created_at", "asc"))
	require.Error(t, ValidateSort("", "asc"))
	require.Error(t, ValidateSort("created_at", "invalid"))
}
