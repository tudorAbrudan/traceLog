package upgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestParseChecksum(t *testing.T) {
	data := []byte(`abc123 *tracelog_linux_amd64.tar.gz
def456  other.txt
`)
	h, err := parseChecksum(data, "tracelog_linux_amd64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if h != "abc123" {
		t.Fatalf("hash: %q", h)
	}
}

func TestExtractTracelogBinary(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: "tracelog", Mode: 0755, Size: 4, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte{1, 2, 3, 4})
	tw.Close()
	gw.Close()

	out, err := extractTracelogBinary(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string([]byte{1, 2, 3, 4}) {
		t.Fatalf("bad payload: %v", out)
	}
}

func TestParseChecksum_NoMatch(t *testing.T) {
	_, err := parseChecksum([]byte("foo bar\n"), "missing.tar.gz")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no checksum") {
		t.Fatal(err)
	}
}
