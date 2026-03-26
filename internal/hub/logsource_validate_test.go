package hub

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tudorAbrudan/tracelog/internal/hub/store"
)

const sampleNginxAccess = `127.0.0.1 - - [25/Dec/2024:10:00:00 +0000] "GET / HTTP/1.1" 200 1234 "-" "curl/8.0"
`

const sampleApacheCommon = `127.0.0.1 - frank [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326
`

func TestValidateLogSourceRecord(t *testing.T) {
	dir := t.TempDir()
	plainFile := filepath.Join(dir, "plain.log")
	if err := os.WriteFile(plainFile, []byte("just some application output\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nginxFile := filepath.Join(dir, "access.log")
	if err := os.WriteFile(nginxFile, []byte(sampleNginxAccess), 0o600); err != nil {
		t.Fatal(err)
	}
	apacheFile := filepath.Join(dir, "apache.log")
	if err := os.WriteFile(apacheFile, []byte(sampleApacheCommon), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		ls      store.LogSourceRecord
		wantErr string
	}{
		{"missing name", store.LogSourceRecord{Type: "file", Path: plainFile, Format: "plain"}, "name is required"},
		{"missing type", store.LogSourceRecord{Name: "x", Path: plainFile, Format: "plain"}, "type is required"},
		{"bad type", store.LogSourceRecord{Name: "x", Type: "docker", Path: plainFile, Format: "plain"}, "unsupported log source type"},
		{"missing path", store.LogSourceRecord{Name: "x", Type: "file", Format: "plain"}, "path is required"},
		{"nonexistent", store.LogSourceRecord{Name: "x", Type: "file", Path: filepath.Join(dir, "nope"), Format: "plain"}, "path does not exist"},
		{"directory", store.LogSourceRecord{Name: "x", Type: "file", Path: dir, Format: "plain"}, "regular file"},
		{"bad format", store.LogSourceRecord{Name: "x", Type: "file", Path: plainFile, Format: "json"}, "unsupported format"},
		{"ok plain", store.LogSourceRecord{Name: "x", Type: "file", Path: plainFile, Format: "plain"}, ""},
		{"ok nginx", store.LogSourceRecord{Name: "x", Type: "file", Path: nginxFile, Format: "nginx"}, ""},
		{"ok apache common", store.LogSourceRecord{Name: "x", Type: "file", Path: apacheFile, Format: "apache"}, ""},
		{"nginx on plain content", store.LogSourceRecord{Name: "x", Type: "file", Path: plainFile, Format: "nginx"}, "does not match"},
		{"apache on plain content", store.LogSourceRecord{Name: "x", Type: "file", Path: plainFile, Format: "apache"}, "does not match"},
		{"default format empty", store.LogSourceRecord{Name: "x", Type: "file", Path: plainFile, Format: ""}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := tt.ls
			normalizeLogSource(&ls)
			err := validateLogSourceRecord(&ls)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileMatchesFormat_apacheMatchesNginxStyleLine(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.log")
	if err := os.WriteFile(p, []byte(sampleNginxAccess), 0o600); err != nil {
		t.Fatal(err)
	}
	// Typical nginx access lines also satisfy the apache access pattern (prefix).
	if err := validateFileMatchesFormat(p, "apache"); err != nil {
		t.Fatal(err)
	}
}
