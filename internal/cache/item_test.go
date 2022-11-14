package cache

import (
	"testing"
	"time"
)

func TestItemIsExpired(t *testing.T) {
	testCases := []struct {
		name     string
		i        item
		expected bool
	}{
		{
			name:     "Zero expiration value",
			i:        item{[]byte(""), 0},
			expected: false,
		},
		{
			name:     "Negative expiration value",
			i:        item{[]byte(""), -100},
			expected: true,
		},
		{
			name:     "Expired",
			i:        item{[]byte(""), time.Now().UnixNano() - 100000},
			expected: true,
		},
		{
			name:     "Not expired",
			i:        item{[]byte(""), time.Now().UnixNano() + 100000},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.i.isExpired() != tc.expected {
				t.Errorf("Expected %v, got %v instead", tc.expected, tc.i.isExpired())
			}
		})
	}
}
