package auth

import (
	"testing"
)

func HashPasswordTest(t *testing.T) {
	type testCase struct {
		name     string
		password string
		wantErr  bool
	}

	testCases := []testCase{
		{
			name:     "Happy Path",
			password: "Password123",
			wantErr:  false,
		},
	}
}
