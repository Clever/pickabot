package github

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenIsExpired(t *testing.T) {
	tests := []struct {
		name       string
		expiration time.Time
		want       bool
	}{
		{
			name:       "future expiration",
			expiration: time.Now().Add(2 * time.Hour),
			want:       false,
		},
		{
			name:       "past expiration",
			expiration: time.Now().Add(-2 * time.Hour),
			want:       true,
		},
		{
			name:       "current expiration",
			expiration: time.Now().Add(expirationTimeBufferDuration - time.Second),
			want:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := Token{
				Expiration: tt.expiration,
			}
			got := token.IsExpired()
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}
