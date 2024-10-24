package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{
			name:     "Valid days duration",
			input:    "10d",
			expected: 10 * 24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "Valid hours duration",
			input:    "10h",
			expected: 10 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "Valid minutes duration",
			input:    "10m",
			expected: 10 * time.Minute,
			wantErr:  false,
		},
		{
			name:     "Valid seconds duration",
			input:    "10s",
			expected: 10 * time.Second,
			wantErr:  false,
		},
		{
			name:    "Invalid days duration",
			input:   "10x",
			wantErr: true,
		},
		{
			name:    "Invalid format",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := ParseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, duration)
			}
		})
	}
}
