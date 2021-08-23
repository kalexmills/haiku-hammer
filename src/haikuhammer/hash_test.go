package haikuhammer_test

import (
	"github.com/kalexmills/haiku-enforcer/src/haikuhammer"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_DuplicateHash(t *testing.T) {
	equal := [][]string {
		{"asdf", "asdf"},
		{"asdf", "ASDF"},
		{"asdf", "asd'f"},
		{"Asdf,", "\"asDf\""},
	}
	notEqual := [][]string {
		{"asdf", "Asdfs"},
		{"gasdf", "asdf"},
		{"asdf", "as df"},
		{"asdf", "as\ndf"},
	}

	for _, tt := range equal {
		assert.Equal(t, haikuhammer.DuplicateHash(tt[0]), haikuhammer.DuplicateHash(tt[1]), "hash('%s') != hash('%s')", tt[0], tt[1])
	}
	for _, tt := range notEqual {
		assert.NotEqual(t, haikuhammer.DuplicateHash(tt[0]), haikuhammer.DuplicateHash(tt[1]), "hash('%s') == hash('%s')", tt[0], tt[1])
	}
}