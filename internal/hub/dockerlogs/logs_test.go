package dockerlogs

import (
	"context"
	"testing"
)

func TestFetch_InvalidContainer(t *testing.T) {
	_, err := Fetch(context.Background(), "; rm -rf /", 100)
	if err == nil {
		t.Fatal("expected error for malicious container name")
	}
	_, err = Fetch(context.Background(), "", 100)
	if err == nil {
		t.Fatal("expected error for empty container")
	}
}
