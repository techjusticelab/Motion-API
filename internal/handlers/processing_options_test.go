package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProcessOptionsJSON_UsesDefaults(t *testing.T) {
	opts, err := parseProcessOptionsJSON(`{"extract_text":false,"timeout_seconds":90}`)
	require.NoError(t, err)

	require.False(t, opts.ExtractText)
	require.True(t, opts.ClassifyDoc) // default remains true when omitted

	require.NoError(t, opts.Validate())
	opts.ApplyDefaults()

	require.Equal(t, 90, opts.TimeoutSeconds)
	require.Equal(t, 1, opts.RetryCount) // default applied
}

func TestParseProcessOptionsJSON_InvalidJSON(t *testing.T) {
	_, err := parseProcessOptionsJSON(`{"extract_text":}`)
	require.Error(t, err)
}

func TestParseProcessOptionsJSON_UnknownField(t *testing.T) {
	_, err := parseProcessOptionsJSON(`{"extract_text":true,"unexpected":true}`)
	require.Error(t, err)
}
