package haikuhammer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCountSyllables(t *testing.T) {
	tests := []struct {
		input string
		expectedCount int
		ok bool
	}{
		{"shitposting", 3, true},
		{"don't", 1, true},
		{"A.B.C.", 3, true},
		{"W.P.A", 5, true},
		{"hello", 2, true},
		{"yesterday", 3, true},
		{"sadfhgdh", 0, false},
		{"shit", 1, true},
		{"posting", 2, true},
		{"bookkeeper", 3, true},
		{"walking", 2, true},
	}

	for _, tt := range tests {
		count, ok := CountSyllables(tt.input)
		assert.Equal(t, ok, tt.ok, tt.input)
		if ok {
			assert.Equal(t, count, tt.expectedCount, tt.input)
		}
	}
}