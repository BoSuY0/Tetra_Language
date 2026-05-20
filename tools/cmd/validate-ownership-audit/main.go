package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	auditPath := flag.String("audit", "docs/release/ownership_production_audit.md", "ownership production audit Markdown")
	expectedStatus := flag.String("expected-status", "not-achieved", "expected audit status: not-achieved or achieved")
	flag.Parse()

	raw, err := os.ReadFile(*auditPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate-ownership-audit: read audit: %v\n", err)
		os.Exit(2)
	}
	if err := validateOwnershipAudit(raw, ownershipAuditOptions{ExpectedStatus: *expectedStatus}); err != nil {
		fmt.Fprintf(os.Stderr, "validate-ownership-audit: %v\n", err)
		os.Exit(1)
	}
}
