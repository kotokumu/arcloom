package buildcheck_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestPackageDependencyBoundaries(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller could not locate repository")
	}
	repository := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	tests := []struct {
		name             string
		packageDirectory string
		allowed          map[string]bool
	}{
		{
			name:             "reconciliation control is target independent",
			packageDirectory: "reconciliationcontrol",
			allowed:          map[string]bool{"context": true, "errors": true, "fmt": true, "strings": true, "time": true},
		},
		{
			name:             "plan reconciliation has narrow dependencies",
			packageDirectory: "planreconciliation",
			allowed: map[string]bool{
				"context": true, "errors": true, "fmt": true,
				"github.com/kotokumu/arcloom/plancontrol":           true,
				"github.com/kotokumu/arcloom/plansnapshot":          true,
				"github.com/kotokumu/arcloom/reconciliationcontrol": true,
			},
		},
		{
			name:             "plan representation owns its result",
			packageDirectory: "planrepresentation",
			allowed: map[string]bool{
				"context": true, "errors": true, "fmt": true, "sort": true,
				"github.com/kotokumu/arcloom/plan": true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filenames, err := filepath.Glob(filepath.Join(repository, tt.packageDirectory, "*.go"))
			if err != nil {
				t.Fatalf("find package files: %v", err)
			}
			for _, filename := range filenames {
				if strings.HasSuffix(filename, "_test.go") {
					continue
				}
				file, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ImportsOnly)
				if err != nil {
					t.Fatalf("parse %s: %v", filename, err)
				}
				for _, imported := range file.Imports {
					path, err := strconv.Unquote(imported.Path.Value)
					if err != nil {
						t.Fatalf("unquote import in %s: %v", filename, err)
					}
					if !tt.allowed[path] {
						t.Errorf("%s imports disallowed dependency %q", filename, path)
					}
				}
			}
		})
	}
}
