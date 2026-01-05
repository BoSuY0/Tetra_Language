package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type smokeCaseReport struct {
	Name         string `json:"name"`
	SrcPath      string `json:"src_path"`
	ExpectedExit int    `json:"expected_exit"`
	ActualExit   *int   `json:"actual_exit,omitempty"`
	Ran          bool   `json:"ran"`
	Pass         bool   `json:"pass"`
	Error        string `json:"error,omitempty"`
}

type smokeReport struct {
	Timestamp    string            `json:"timestamp"`
	Target       string            `json:"target"`
	Host         string            `json:"host"`
	Version      string            `json:"version"`
	GitHead      string            `json:"git_head,omitempty"`
	IslandsDebug bool              `json:"islands_debug"`
	Cases        []smokeCaseReport `json:"cases"`
}

func sectionHeadingForTarget(target string) (string, error) {
	switch target {
	case "windows-x64":
		return "## Windows x64", nil
	case "macos-x64":
		return "## macOS x64", nil
	case "linux-x64":
		return "## Linux x64 (sanity)", nil
	default:
		return "", fmt.Errorf("unsupported target %q", target)
	}
}

func setHeaderField(md string, key string, value string) string {
	lines := strings.Split(md, "\n")
	prefix := key + ":"
	for i := range lines {
		if strings.HasPrefix(lines[i], prefix) {
			if value == "" {
				lines[i] = prefix
			} else {
				lines[i] = prefix + " " + value
			}
			break
		}
	}
	return strings.Join(lines, "\n")
}

func extractSection(md string, heading string) (before string, section string, after string, err error) {
	idx := strings.Index(md, heading+"\n")
	if idx == -1 {
		return "", "", "", fmt.Errorf("missing heading %q", heading)
	}
	before = md[:idx]
	rest := md[idx:]
	nextIdx := strings.Index(rest[len(heading)+1:], "\n## ")
	if nextIdx == -1 {
		return before, rest, "", nil
	}
	nextIdx += len(heading) + 1
	section = rest[:nextIdx]
	after = rest[nextIdx:]
	return before, section, after, nil
}

func setCheckboxState(section string, contains string, checked bool) (string, bool) {
	lines := strings.Split(section, "\n")
	changed := false
	want := "- [ ]"
	if checked {
		want = "- [x]"
	}
	for i := range lines {
		if !strings.Contains(lines[i], contains) {
			continue
		}
		if strings.Contains(lines[i], "- [ ]") {
			if checked {
				lines[i] = strings.Replace(lines[i], "- [ ]", want, 1)
				changed = true
			}
			continue
		}
		if strings.Contains(lines[i], "- [x]") {
			if !checked {
				lines[i] = strings.Replace(lines[i], "- [x]", want, 1)
				changed = true
			}
			continue
		}
	}
	if changed {
		return strings.Join(lines, "\n"), true
	}
	return section, false
}

type checkboxUpdate struct {
	Contains string
	Checked  bool
}

func applyToChecklist(path string, report *smokeReport, updates []checkboxUpdate) error {
	if report == nil {
		return fmt.Errorf("missing report")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	md := string(data)

	ts := report.Timestamp
	if ts == "" {
		ts = time.Now().UTC().Format(time.RFC3339)
	}
	date := strings.SplitN(ts, "T", 2)[0]

	md = setHeaderField(md, "Date", date)
	md = setHeaderField(md, "Target version", report.Target)
	md = setHeaderField(md, "Git HEAD", report.GitHead)
	md = setHeaderField(md, "Compiler version (compilerVersion)", report.Version)

	heading, err := sectionHeadingForTarget(report.Target)
	if err != nil {
		return err
	}
	before, section, after, err := extractSection(md, heading)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	for _, u := range updates {
		updated, ok := setCheckboxState(section, u.Contains, u.Checked)
		if ok {
			section = updated
		}
	}

	out := before + section + after
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func main() {
	var reportPath string
	var islandsChecklist string
	var actorsChecklist string

	flag.StringVar(&reportPath, "report", "", "path to tetra smoke JSON report")
	flag.StringVar(&islandsChecklist, "islands-checklist", filepath.FromSlash("docs/checklists/islands_platform_smoke.md"), "path to islands platform checklist")
	flag.StringVar(&actorsChecklist, "actors-checklist", filepath.FromSlash("docs/checklists/actors_platform_smoke.md"), "path to actors platform checklist")
	flag.Parse()

	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var report smokeReport
	if err := json.Unmarshal(raw, &report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	passed := make(map[string]bool, len(report.Cases))
	ran := make(map[string]bool, len(report.Cases))
	for _, c := range report.Cases {
		passed[c.Name] = c.Pass
		ran[c.Name] = c.Ran
	}

	var islandsUpdates []checkboxUpdate
	for _, name := range []string{"islands_hello", "islands_i32", "islands_overflow", "mmio_smoke", "cap_mem_smoke", "memset_smoke"} {
		if _, ok := passed[name]; !ok {
			continue
		}
		islandsUpdates = append(islandsUpdates, checkboxUpdate{
			Contains: fmt.Sprintf("examples/%s.tetra", name),
			Checked:  passed[name],
		})
		islandsUpdates = append(islandsUpdates, checkboxUpdate{
			Contains: fmt.Sprintf("./%s", name),
			Checked:  passed[name] && ran[name],
		})
	}
	if len(islandsUpdates) > 0 {
		if err := applyToChecklist(islandsChecklist, &report, islandsUpdates); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	var actorsUpdates []checkboxUpdate
	if _, ok := passed["actors_pingpong"]; ok {
		actorsUpdates = append(actorsUpdates, checkboxUpdate{
			Contains: "examples/actors_pingpong.tetra",
			Checked:  passed["actors_pingpong"],
		})
		actorsUpdates = append(actorsUpdates, checkboxUpdate{
			Contains: "./actors_pingpong",
			Checked:  passed["actors_pingpong"] && ran["actors_pingpong"],
		})
	}
	if len(actorsUpdates) > 0 {
		if err := applyToChecklist(actorsChecklist, &report, actorsUpdates); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
