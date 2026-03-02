package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePort(t *testing.T) {
	tests := []struct {
		port    int
		wantErr bool
	}{
		{0, false},
		{1, false},
		{8080, false},
		{65535, false},
		{-1, true},
		{65536, true},
		{-9999, true},
	}
	for _, tt := range tests {
		err := validatePort(tt.port)
		if tt.wantErr {
			require.Error(t, err, "port %d should be invalid", tt.port)
		} else {
			assert.NoError(t, err, "port %d should be valid", tt.port)
		}
	}
}
