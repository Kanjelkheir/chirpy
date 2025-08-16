package auth

import (
	"testing"
)

type test struct {
	password string
	hash     string
}

func TestHasher(t *testing.T) {
	tests := []test{
		{
			password: "Hello",
			hash:     "$2a$12$g5PQbcUSzO266O.x6JL2jeDOo9XIcF/9rsPH48Zq5U8AU0Mj0mx3q",
		},
		{
			password: "password123",
			hash:     "$2a$12$l2XsMf9jwNLQsf86u5Gnae5dCyksRzSrUXDMOBw8fe1lGNfLD95KO",
		},
	}
	for _, test := range tests {
		hash, err := HashPassword(test.password)
		if err != nil {
			t.Errorf("HashPassword function failed: %v", err)
		}
		if hash != test.hash {
			t.Errorf("Hashes don't match\nexpected: %s\nfound: %s", test.hash, hash)
		}
	}
}
