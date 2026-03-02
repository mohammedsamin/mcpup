package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmDefaultNoOnEmptyInput(t *testing.T) {
	SetTTY(true)
	t.Cleanup(func() { SetTTY(false) })

	in := strings.NewReader("\n")
	var out bytes.Buffer

	ok, err := Confirm(in, &out, "Enable on clients now?", false)
	if err != nil {
		t.Fatalf("confirm returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected default false on empty input")
	}
}

func TestConfirmRepromptsOnInvalidInput(t *testing.T) {
	SetTTY(true)
	t.Cleanup(func() { SetTTY(false) })

	in := strings.NewReader("maybe\nn\n")
	var out bytes.Buffer

	ok, err := Confirm(in, &out, "Enable on clients now?", true)
	if err != nil {
		t.Fatalf("confirm returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected explicit no to return false")
	}
	if !strings.Contains(out.String(), "Please answer y or n.") {
		t.Fatalf("expected invalid-input hint in prompt output")
	}
}
