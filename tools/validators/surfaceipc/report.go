package surfaceipc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	SchemaV1            = "tetra.surface.ipc-lifecycle-report.v1"
	LevelIPCLifecycleV1 = "surface-ipc-lifecycle-v1"
)

type Report struct {
	Schema         string         `json:"schema"`
	Status         string         `json:"status"`
	Level          string         `json:"level"`
	Scope          string         `json:"scope"`
	ReleaseScope   string         `json:"release_scope"`
	Producer       string         `json:"producer,omitempty"`
	GitHead        string         `json:"git_head,omitempty"`
	Version        string         `json:"version,omitempty"`
	App            AppModel       `json:"app"`
	Messages       []Message      `json:"messages"`
	UIUpdates      []UIUpdate     `json:"ui_updates"`
	CrashIsolation CrashIsolation `json:"crash_isolation"`
	Operations     []Operation    `json:"operations"`
	NegativeGuards NegativeGuards `json:"negative_guards"`
	NonClaims      []string       `json:"nonclaims"`
	Cases          []CaseReport   `json:"cases"`
}

type AppModel struct {
	Main               string          `json:"main"`
	UIIsolate          string          `json:"ui_isolate"`
	UIThreadPolicy     string          `json:"ui_thread_policy"`
	BackgroundServices []string        `json:"background_services"`
	Lifecycle          []LifecycleStep `json:"lifecycle"`
}

type LifecycleStep struct {
	Name             string `json:"name"`
	Phase            string `json:"phase"`
	UIThread         bool   `json:"ui_thread"`
	DispatcherRouted bool   `json:"dispatcher_routed"`
	Evidence         string `json:"evidence"`
}

type Message struct {
	Name                  string `json:"name"`
	Direction             string `json:"direction"`
	PayloadKind           string `json:"payload_kind"`
	OwnedData             bool   `json:"owned_data"`
	ContainsSurfaceHandle bool   `json:"contains_surface_handle"`
	ContainsSurfaceFrame  bool   `json:"contains_surface_frame"`
	ContainsSurfaceEvent  bool   `json:"contains_surface_event"`
	DispatcherRouted      bool   `json:"dispatcher_routed"`
	Typed                 bool   `json:"typed"`
	Accepted              bool   `json:"accepted"`
	Evidence              string `json:"evidence"`
}

type UIUpdate struct {
	Name             string `json:"name"`
	Source           string `json:"source"`
	Target           string `json:"target"`
	MutatesUI        bool   `json:"mutates_ui"`
	DispatcherRouted bool   `json:"dispatcher_routed"`
	Allowed          bool   `json:"allowed"`
	Evidence         string `json:"evidence"`
}

type CrashIsolation struct {
	Strategy                 string `json:"strategy"`
	UIStatePreserved         bool   `json:"ui_state_preserved"`
	BackgroundServiceRestart bool   `json:"background_service_restart"`
	CrashReport              bool   `json:"crash_report"`
	Evidence                 string `json:"evidence"`
}

type Operation struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

type NegativeGuards struct {
	SurfaceHandleActorTransferRejected            bool `json:"surface_handle_actor_transfer_rejected"`
	SurfaceFrameActorMessageRejected              bool `json:"surface_frame_actor_message_rejected"`
	SurfaceEventActorMessageRejected              bool `json:"surface_event_actor_message_rejected"`
	BackgroundUIMutationWithoutDispatcherRejected bool `json:"background_ui_mutation_without_dispatcher_rejected"`
	BorrowedPayloadRejected                       bool `json:"borrowed_payload_rejected"`
	UntypedChannelRejected                        bool `json:"untyped_channel_rejected"`
	CrashIsolationRequired                        bool `json:"crash_isolation_required"`
}

type CaseReport struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

func ValidateReport(raw []byte) error {
	report, err := decodeReport(raw)
	if err != nil {
		return err
	}
	return Validate(report)
}

func Validate(report Report) error {
	var issues []string
	issues = append(issues, validateIdentity(report)...)
	issues = append(issues, validateApp(report.App)...)
	issues = append(issues, validateMessages(report.Messages)...)
	issues = append(issues, validateUIUpdates(report.UIUpdates)...)
	issues = append(issues, validateCrashIsolation(report.CrashIsolation)...)
	issues = append(issues, validateOperations(report.Operations)...)
	issues = append(issues, validateNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateCases(report.Cases)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func decodeReport(raw []byte) (Report, error) {
	var report Report
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err != nil {
		return Report{}, err
	}
	if err := ensureJSONEOF(dec); err != nil {
		return Report{}, err
	}
	return report, nil
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("unexpected trailing JSON payload")
}

func validateIdentity(report Report) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Level != LevelIPCLifecycleV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelIPCLifecycleV1))
	}
	if report.Scope != "surface-v1-scoped-linux-web-ipc-lifecycle" {
		issues = append(issues, fmt.Sprintf("scope is %q, want surface-v1-scoped-linux-web-ipc-lifecycle", report.Scope))
	}
	if report.ReleaseScope != "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want PROD_STABLE_SCOPED_LINUX_WEB_APP_UI", report.ReleaseScope))
	}
	return issues
}

func validateApp(app AppModel) []string {
	var issues []string
	if strings.TrimSpace(app.Main) == "" {
		issues = append(issues, "app main entry is required")
	}
	if strings.TrimSpace(app.UIIsolate) == "" {
		issues = append(issues, "app UI isolate is required")
	}
	if app.UIThreadPolicy != "single-owner-ui-dispatcher-v1" {
		issues = append(issues, fmt.Sprintf("ui_thread_policy is %q, want single-owner-ui-dispatcher-v1", app.UIThreadPolicy))
	}
	if len(app.BackgroundServices) == 0 {
		issues = append(issues, "background services are required")
	}
	for i, service := range app.BackgroundServices {
		if strings.TrimSpace(service) == "" {
			issues = append(issues, fmt.Sprintf("background_services[%d] is empty", i))
		}
	}
	if len(app.Lifecycle) == 0 {
		issues = append(issues, "lifecycle steps are required")
		return issues
	}
	required := map[string]bool{"start": false, "suspend": false, "stop": false}
	for i, step := range app.Lifecycle {
		prefix := fmt.Sprintf("lifecycle[%d]", i)
		if strings.TrimSpace(step.Name) == "" || strings.TrimSpace(step.Phase) == "" {
			issues = append(issues, prefix+" requires name and phase")
		}
		if strings.TrimSpace(step.Evidence) == "" {
			issues = append(issues, prefix+" evidence is required")
		}
		if _, ok := required[step.Phase]; ok {
			required[step.Phase] = true
		}
		if step.Phase == "start" && !step.UIThread {
			issues = append(issues, "lifecycle start must establish the UI isolate on the UI thread")
		}
		if !step.UIThread && !step.DispatcherRouted {
			issues = append(issues, fmt.Sprintf("lifecycle step %s crosses background boundary without dispatcher routing", step.Name))
		}
	}
	for phase, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("lifecycle phase %q is required", phase))
		}
	}
	return issues
}

func validateMessages(messages []Message) []string {
	if len(messages) == 0 {
		return []string{"IPC messages are required"}
	}
	var issues []string
	acceptedOwnedDispatcher := false
	rejectedSurfaceTransfer := false
	for i, msg := range messages {
		prefix := fmt.Sprintf("messages[%d]", i)
		if strings.TrimSpace(msg.Name) == "" || strings.TrimSpace(msg.Direction) == "" || strings.TrimSpace(msg.PayloadKind) == "" {
			issues = append(issues, prefix+" requires name, direction, and payload_kind")
		}
		if strings.TrimSpace(msg.Evidence) == "" {
			issues = append(issues, prefix+" evidence is required")
		}
		if msg.ContainsSurfaceHandle || strings.Contains(strings.ToLower(msg.PayloadKind), "surface-handle") {
			if msg.Accepted {
				issues = append(issues, fmt.Sprintf("message %s surface handle cannot cross actor/task boundary", msg.Name))
			} else {
				rejectedSurfaceTransfer = true
			}
		}
		if msg.ContainsSurfaceFrame || strings.Contains(strings.ToLower(msg.PayloadKind), "surface-frame") {
			if msg.Accepted {
				issues = append(issues, fmt.Sprintf("message %s surface frame cannot cross actor/task boundary", msg.Name))
			} else {
				rejectedSurfaceTransfer = true
			}
		}
		if msg.ContainsSurfaceEvent || strings.Contains(strings.ToLower(msg.PayloadKind), "surface-event") {
			if msg.Accepted {
				issues = append(issues, fmt.Sprintf("message %s surface event cannot cross actor/task boundary", msg.Name))
			} else {
				rejectedSurfaceTransfer = true
			}
		}
		if msg.Accepted {
			if !msg.Typed {
				issues = append(issues, fmt.Sprintf("message %s accepted untyped IPC channel", msg.Name))
			}
			if !msg.OwnedData || strings.Contains(strings.ToLower(msg.PayloadKind), "borrow") {
				issues = append(issues, fmt.Sprintf("message %s accepted payload without owned data", msg.Name))
			}
			if msg.Direction == "background-to-ui" && !msg.DispatcherRouted {
				issues = append(issues, fmt.Sprintf("message %s background-to-ui IPC requires dispatcher routing", msg.Name))
			}
			if msg.OwnedData && msg.Typed && msg.DispatcherRouted && msg.Direction == "background-to-ui" {
				acceptedOwnedDispatcher = true
			}
		}
	}
	if !acceptedOwnedDispatcher {
		issues = append(issues, "owned background message dispatch evidence is required")
	}
	if !rejectedSurfaceTransfer {
		issues = append(issues, "rejected Surface handle/frame/event actor transfer evidence is required")
	}
	return issues
}

func validateUIUpdates(updates []UIUpdate) []string {
	if len(updates) == 0 {
		return []string{"UI update records are required"}
	}
	var issues []string
	allowedDispatcherUpdate := false
	rejectedDirectBackgroundUpdate := false
	for i, update := range updates {
		prefix := fmt.Sprintf("ui_updates[%d]", i)
		if strings.TrimSpace(update.Name) == "" || strings.TrimSpace(update.Source) == "" || strings.TrimSpace(update.Target) == "" {
			issues = append(issues, prefix+" requires name, source, and target")
		}
		if strings.TrimSpace(update.Evidence) == "" {
			issues = append(issues, prefix+" evidence is required")
		}
		background := strings.Contains(update.Source, "background") || strings.Contains(update.Source, "task") || strings.Contains(update.Source, "service")
		if update.Allowed && update.MutatesUI && !update.DispatcherRouted {
			issues = append(issues, fmt.Sprintf("UI update %s mutates UI without dispatcher routing", update.Name))
		}
		if update.Allowed && update.MutatesUI && background && update.DispatcherRouted {
			allowedDispatcherUpdate = true
		}
		if !update.Allowed && update.MutatesUI && background && !update.DispatcherRouted {
			rejectedDirectBackgroundUpdate = true
		}
	}
	if !allowedDispatcherUpdate {
		issues = append(issues, "dispatcher-routed background UI update evidence is required")
	}
	if !rejectedDirectBackgroundUpdate {
		issues = append(issues, "background UI mutation without dispatcher rejection evidence is required")
	}
	return issues
}

func validateCrashIsolation(crash CrashIsolation) []string {
	var issues []string
	if crash.Strategy != "supervised-background-services-v1" {
		issues = append(issues, fmt.Sprintf("crash isolation strategy is %q, want supervised-background-services-v1", crash.Strategy))
	}
	if !crash.UIStatePreserved {
		issues = append(issues, "crash isolation requires UI state preservation evidence")
	}
	if !crash.BackgroundServiceRestart {
		issues = append(issues, "crash isolation requires background service restart evidence")
	}
	if !crash.CrashReport {
		issues = append(issues, "crash isolation requires crash report evidence")
	}
	if strings.TrimSpace(crash.Evidence) == "" {
		issues = append(issues, "crash isolation evidence is required")
	}
	return issues
}

func validateOperations(operations []Operation) []string {
	required := map[string]bool{
		"app lifecycle validated":            false,
		"owned message passing validated":    false,
		"dispatcher UI updates validated":    false,
		"crash isolation strategy validated": false,
	}
	var issues []string
	for i, op := range operations {
		if strings.TrimSpace(op.Name) == "" || strings.TrimSpace(op.Kind) == "" {
			issues = append(issues, fmt.Sprintf("operations[%d] requires name and kind", i))
		}
		if !op.Ran || !op.Pass {
			issues = append(issues, fmt.Sprintf("operation %q must run and pass", op.Name))
		}
		if _, ok := required[op.Name]; ok {
			required[op.Name] = true
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("operation %q is required", name))
		}
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	checks := map[string]bool{
		"surface handle actor transfer rejection":             guards.SurfaceHandleActorTransferRejected,
		"surface frame actor message rejection":               guards.SurfaceFrameActorMessageRejected,
		"surface event actor message rejection":               guards.SurfaceEventActorMessageRejected,
		"background UI mutation without dispatcher rejection": guards.BackgroundUIMutationWithoutDispatcherRejected,
		"borrowed payload rejection":                          guards.BorrowedPayloadRejected,
		"untyped IPC channel rejection":                       guards.UntypedChannelRejected,
		"crash isolation requirement":                         guards.CrashIsolationRequired,
	}
	var issues []string
	for name, ok := range checks {
		if !ok {
			issues = append(issues, name+" guard is required")
		}
	}
	return issues
}

func validateNonClaims(nonClaims []string) []string {
	if len(nonClaims) == 0 {
		return []string{"IPC/lifecycle nonclaims are required"}
	}
	joined := strings.ToLower(strings.Join(nonClaims, "\n"))
	var issues []string
	for _, required := range []string{"surface handles", "electron", "process sandbox", "crash"} {
		if !strings.Contains(joined, required) {
			issues = append(issues, fmt.Sprintf("IPC/lifecycle nonclaims must mention %s boundary", required))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"owned background message dispatch":                  false,
		"surface handle actor transfer rejected":             false,
		"surface frame actor message rejected":               false,
		"surface event actor message rejected":               false,
		"background UI mutation without dispatcher rejected": false,
		"borrowed payload rejected":                          false,
		"untyped IPC channel rejected":                       false,
	}
	var issues []string
	for i, c := range cases {
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Kind) == "" {
			issues = append(issues, fmt.Sprintf("cases[%d] requires name and kind", i))
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %q must run and pass", c.Name))
		}
		if _, ok := required[c.Name]; ok {
			required[c.Name] = true
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("case %q is required", name))
		}
	}
	return issues
}
