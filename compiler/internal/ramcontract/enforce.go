package ramcontract

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const DiagnosticCode = "TETRA4100"

type EnforcementOptions struct {
	FailIfHeap        bool
	FailIfCopy        bool
	FailIfUnbounded   bool
	MemoryBudgetBytes int64
	ContractFile      string
}

type ContractFile struct {
	SchemaVersion     string      `json:"schema_version"`
	MaxGrade          MemoryGrade `json:"max_grade,omitempty"`
	FailIfHeap        bool        `json:"fail_if_heap,omitempty"`
	FailIfCopy        bool        `json:"fail_if_copy,omitempty"`
	FailIfUnbounded   bool        `json:"fail_if_unbounded,omitempty"`
	MemoryBudgetBytes int64       `json:"memory_budget_bytes,omitempty"`
}

type EnforcementError struct {
	Rule     string
	SiteID   string
	Function string
	Grade    MemoryGrade
	Blockers []string
	Message  string
}

func (e EnforcementError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	parts := []string{e.Rule}
	if e.Function != "" {
		parts = append(parts, "function="+e.Function)
	}
	if e.SiteID != "" {
		parts = append(parts, "site="+e.SiteID)
	}
	if e.Grade != "" {
		parts = append(parts, "grade="+string(e.Grade))
	}
	if len(e.Blockers) > 0 {
		parts = append(parts, "blockers="+strings.Join(e.Blockers, ","))
	}
	return strings.Join(parts, " ")
}

func (e EnforcementError) DiagnosticCode() string {
	return DiagnosticCode
}

func Enforce(report Report, opt EnforcementOptions) error {
	if strings.TrimSpace(opt.ContractFile) != "" {
		contract, err := ReadContractFile(opt.ContractFile)
		if err != nil {
			return err
		}
		if contract.FailIfHeap {
			opt.FailIfHeap = true
		}
		if contract.FailIfCopy {
			opt.FailIfCopy = true
		}
		if contract.FailIfUnbounded {
			opt.FailIfUnbounded = true
		}
		if contract.MemoryBudgetBytes > 0 {
			opt.MemoryBudgetBytes = contract.MemoryBudgetBytes
		}
		if contract.MaxGrade != "" && gradeRank(report.Summary.ArtifactGrade) > gradeRank(contract.MaxGrade) {
			return EnforcementError{
				Rule:    "RAM_CONTRACT_GRADE",
				Grade:   report.Summary.ArtifactGrade,
				Message: fmt.Sprintf("RAM_CONTRACT_GRADE artifact grade %s exceeds contract max_grade %s", report.Summary.ArtifactGrade, contract.MaxGrade),
			}
		}
	}
	if err := ValidateReport(report); err != nil {
		return err
	}
	if opt.MemoryBudgetBytes > 0 && report.Summary.BudgetBytes > opt.MemoryBudgetBytes {
		return EnforcementError{
			Rule:    "RAM_CONTRACT_BUDGET",
			Grade:   report.Summary.ArtifactGrade,
			Message: fmt.Sprintf("RAM_CONTRACT_BUDGET budget_bytes %d exceeds memory_budget %d", report.Summary.BudgetBytes, opt.MemoryBudgetBytes),
		}
	}
	for _, row := range report.Rows {
		if opt.FailIfHeap && isHeapPlacement(row.Placement) {
			return EnforcementError{Rule: "RAM_CONTRACT_HEAP", SiteID: row.SiteID, Function: row.Function, Grade: row.ContractGrade, Blockers: row.Blockers}
		}
		if opt.FailIfCopy && isCopyIntent(row.Intent) {
			return EnforcementError{Rule: "RAM_CONTRACT_COPY", SiteID: row.SiteID, Function: row.Function, Grade: row.ContractGrade, Blockers: append(row.Blockers, row.CopyReason)}
		}
		if opt.FailIfUnbounded && (row.Placement == PlacementHeapUnbounded || row.ContractGrade == GradeM5 || row.ContractGrade == GradeM6) {
			return EnforcementError{Rule: "RAM_CONTRACT_UNBOUNDED", SiteID: row.SiteID, Function: row.Function, Grade: row.ContractGrade, Blockers: row.Blockers}
		}
	}
	return nil
}

func ReadContractFile(path string) (ContractFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ContractFile{}, err
	}
	var contract ContractFile
	if err := json.Unmarshal(raw, &contract); err != nil {
		return ContractFile{}, err
	}
	if contract.SchemaVersion != "" && contract.SchemaVersion != "tetra.ram-contract-file.v1" {
		return ContractFile{}, fmt.Errorf("ram contract file schema_version is %q, want tetra.ram-contract-file.v1", contract.SchemaVersion)
	}
	if contract.MaxGrade != "" && !knownGrade(contract.MaxGrade) {
		return ContractFile{}, fmt.Errorf("ram contract file max_grade %q is unknown", contract.MaxGrade)
	}
	if contract.MemoryBudgetBytes < 0 {
		return ContractFile{}, errors.New("ram contract file memory_budget_bytes must not be negative")
	}
	return contract, nil
}
