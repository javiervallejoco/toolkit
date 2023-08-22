package toolkit

import (
	"testing"

	"github.com/javiervallejoco/toolkit"
)

func TestTools_RandomString(t *testing.T) {
	var tools toolkit.Tools
	s := tools.RandomString(10)
	if len(s) != 10 {
		t.Errorf("wrong len, expected: %d, gotted: %d\n", 10, len(s))
	}
}
