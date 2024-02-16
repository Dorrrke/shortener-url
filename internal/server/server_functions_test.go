package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/Dorrrke/shortener-url/internal/logger"
)

func TestGetUID(t *testing.T) {
	type want struct {
		UserID string
	}

	tests := []struct {
		name string
		UID  string
		want want
	}{
		{
			name: "Test get id from jwt #1",
			UID:  "dafsgfdas-gadsfga-fdsf",
			want: want{
				UserID: "dafsgfdas-gadsfga-fdsf",
			},
		},
		{
			name: "Test get id from jwt #2",
			UID:  "fdsh-gfsdfg-hgfh",
			want: want{
				UserID: "fdsh-gfsdfg-hgfh",
			},
		},
		{
			name: "Test get id from jwt #3",
			UID:  "262453g-fsdh545-gh63",
			want: want{
				UserID: "262453g-fsdh545-gh63",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := tt.UID
			token, err := createJWTToken(userID)
			if err != nil {
				logger.Log.Info("cannot create token", zap.Error(err))
			}
			getedUID := GetUID(token)
			assert.Equal(t, tt.want.UserID, getedUID)
		})

	}
}
