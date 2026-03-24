package auth

import (
	"testing"
)

func HashPasswordTest(t *testing.T) {
	var testCase struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}

	testCases := []testCase{}
}
