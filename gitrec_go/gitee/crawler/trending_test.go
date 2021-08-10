package crawler

import (
	"testing"

	"github.com/golang/mock/gomock"
)

func TestGetTrending(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
		{
			name: "case1",
		},
	}
	for _, tt := range tests {
		tt := tt
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		t.Run(tt.name, func(t *testing.T) {
			GetTrending()
		})
	}
}
