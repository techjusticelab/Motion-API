package validation

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorSetAppendAndError(t *testing.T) {
	set := NewErrorSet()
	assert.Nil(t, set.Error())

	set.Append("field", errors.New("invalid"))
	err := set.Error()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field: invalid")
}

func TestErrorSetEmpty(t *testing.T) {
	var set *ErrorSet
	assert.Nil(t, set.Error())

	set = NewErrorSet()
	assert.True(t, set.Empty())
	set.Append("field", nil)
	assert.True(t, set.Empty())
}
