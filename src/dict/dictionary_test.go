package dict

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSyllableCounts(t *testing.T) {
	counts, ok := SyllableCounts("HAIKU")
	assert.True(t, ok)
	assert.Len(t, counts, 1)
	assert.Equal(t, 2, counts[0])

	_, ok = SyllableCounts("ASDFGF")
	assert.False(t, ok)
}

func TestIsWord(t *testing.T) {
	assert.True(t, IsWord("HOLOGRAPHIC"))
	assert.False(t, IsWord("HADGASDGF"))
}