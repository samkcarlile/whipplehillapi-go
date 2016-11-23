package whapi

import (
	"testing"
)

func TestFull(t *testing.T) {
	wac := NewWhipplehillAPIClient("https://fwcd.myschoolapp.com")

	err := wac.SignIn("sam.carlile", "spicysausage")
	if err != nil {
		t.Error(err)
	}
}
