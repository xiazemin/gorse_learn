package crawler

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
)

func TestGetStared(t *testing.T) {
	tests := []struct {
		name  string
		wantR []map[string]interface{}
	}{
		// TODO: Add test cases.
		{
			name: "case1",

			wantR: []map[string]interface{}{}, //[]map[string]interface{},
		},
	}
	for _, tt := range tests {
		tt := tt
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		t.Run(tt.name, func(t *testing.T) {
			if gotR := GetStared(); reflect.DeepEqual(gotR, tt.wantR) {
				t.Errorf("GetStared() = %v, want %v", gotR, tt.wantR)
			}
		})
	}
}
