package ulid

import "testing"

func TestNewULIDParses(t *testing.T) {
	id := NewULID()
	if len(id) != EncodedSize {
		t.Fatalf("encoded length = %d, want %d", len(id), EncodedSize)
	}
	parsed, err := ParseULID(id)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.String() != id {
		t.Fatalf("round trip = %q, want %q", parsed.String(), id)
	}
}

func TestParseULIDKnownValue(t *testing.T) {
	parsed, err := ParseULID("01arz3ndektsv4rrffq69g5fav")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.String() != "01arz3ndektsv4rrffq69g5fav" {
		t.Fatalf("string = %q", parsed.String())
	}
	if parsed.Time() != 1469922850259 {
		t.Fatalf("time = %d", parsed.Time())
	}
}

func TestParseULIDRejectsUppercase(t *testing.T) {
	if _, err := ParseULID("01ARZ3NDEKTSV4RRFFQ69G5FAV"); err == nil {
		t.Fatal("expected uppercase ULID to be rejected")
	}
}
