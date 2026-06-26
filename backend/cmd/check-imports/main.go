package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// allowedPrefixes lists all packages that a module may import from.
// Any import NOT matching this list will be flagged as a violation.
//
// NOTE: rbac, audit, and queue are intentionally cross-cutting modules
// shared across the system via handler injection and event dispatching.
// Strict module isolation is enforced for domain modules only.
var allowedPrefixes = []string{
	"github.com/tevoworks/corekit/backend/internal/database",
	"github.com/tevoworks/corekit/backend/internal/middleware",
	"github.com/tevoworks/corekit/backend/internal/modules/audit",
	"github.com/tevoworks/corekit/backend/internal/modules/queue",
	"github.com/tevoworks/corekit/backend/internal/modules/rbac",
	"github.com/tevoworks/corekit/backend/internal/modules/settings",
	"github.com/tevoworks/corekit/backend/internal/modules/permregistry",
	"github.com/tevoworks/corekit/backend/internal/config",
	"github.com/tevoworks/corekit/backend/internal/redisstore",
	"github.com/tevoworks/corekit/backend/internal/authverify",
	"github.com/tevoworks/corekit/backend/internal/container",
	"github.com/tevoworks/corekit/backend/internal/validation",
	"github.com/tevoworks/corekit/backend/pkg/",
}

// nonDomainPackages is the set of infra packages that don't need checking.
var nonDomainPackages = map[string]bool{
	"github.com/tevoworks/corekit/backend/internal/middleware":    true,
	"github.com/tevoworks/corekit/backend/internal/config":        true,
	"github.com/tevoworks/corekit/backend/internal/database":      true,
	"github.com/tevoworks/corekit/backend/internal/redisstore":    true,
	"github.com/tevoworks/corekit/backend/internal/authverify":    true,
	"github.com/tevoworks/corekit/backend/internal/container":     true,
	"github.com/tevoworks/corekit/backend/internal/modules/audit": true,
	"github.com/tevoworks/corekit/backend/internal/modules/queue": true,
}

type pkgInfo struct {
	ImportPath  string   `json:"ImportPath"`
	Imports     []string `json:"Imports"`
	TestImports []string `json:"TestImports"`
}

func main() {
	root := flag.String("root", ".", "module root directory")
	flag.Parse()

	cmd := exec.Command("go", "list", "-json", "./internal/modules/...")
	cmd.Dir = *root
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "go list failed: %v\n", err)
		os.Exit(1)
	}

	violations := 0
	dec := json.NewDecoder(strings.NewReader(string(out)))
	for dec.More() {
		var p pkgInfo
		if err := dec.Decode(&p); err != nil {
			break
		}

		// Skip non-domain packages (infrastructure, not modules)
		if nonDomainPackages[p.ImportPath] {
			continue
		}

		// Only check packages under internal/modules/<name>
		if !strings.HasPrefix(p.ImportPath, "github.com/tevoworks/corekit/backend/internal/modules/") {
			continue
		}

		moduleName := strings.TrimPrefix(p.ImportPath, "github.com/tevoworks/corekit/backend/internal/modules/")
		parts := strings.SplitN(moduleName, "/", 2)
		moduleName = parts[0]

		// Check all imports
		allImports := append([]string{}, p.Imports...)
		allImports = append(allImports, p.TestImports...)

		for _, imp := range allImports {
			if !isInternal(imp) {
				continue
			}
			if imp == p.ImportPath {
				continue // own package
			}
			if allowed(imp) {
				continue
			}

			fmt.Printf("VIOLATION: %s imports '%s' — not in allowlist\n", p.ImportPath, imp)
			violations++
		}
	}

	if violations > 0 {
		fmt.Fprintf(os.Stderr, "\nFAIL: %d cross-module import violation(s) found.\n", violations)
		fmt.Fprintf(os.Stderr, "Fix: Move shared logic to internal/pkg/ or inject via interface.\n")
		os.Exit(1)
	}

	fmt.Println("OK: All module boundaries respected.")
}

func isInternal(imp string) bool {
	return strings.HasPrefix(imp, "github.com/tevoworks/corekit/backend/internal/")
}

func allowed(imp string) bool {
	for _, prefix := range allowedPrefixes {
		if imp == prefix || strings.HasPrefix(imp, prefix+"/") || strings.HasPrefix(imp, prefix+".") {
			return true
		}
	}
	return false
}
