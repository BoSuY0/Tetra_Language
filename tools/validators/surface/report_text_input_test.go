package surface

import (
	"strings"
	"testing"
)

func TestValidateReportAcceptsHeadlessTextFocusInputSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessTextFocusInputSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceTextInputReportAcceptsProductionBaseline(t *testing.T) {
	raw := validSurfaceTextInputReportJSON()
	if err := ValidateTextInputReport(raw); err != nil {
		t.Fatalf("ValidateTextInputReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceTextInputReportRejectsFakeProductionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{
			name: "experimental true",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"experimental": false`, `"experimental": true`, 1)
			},
			want: "experimental",
		},
		{
			name: "production false",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"production_claim": true`, `"production_claim": false`, 1)
			},
			want: "production_claim",
		},
		{
			name: "missing utf8 validation",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"utf8_validation": true`, `"utf8_validation": false`, 1)
			},
			want: "utf8_validation",
		},
		{
			name: "missing invalid utf8 rejection",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"invalid_utf8_rejected": true`, `"invalid_utf8_rejected": false`, 1)
			},
			want: "invalid_utf8_rejected",
		},
		{
			name: "missing multiline storage",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"multiline": true`, `"multiline": false`, 1)
			},
			want: "multiline",
		},
		{
			name: "missing composition commit",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"composition_commit": true`, `"composition_commit": false`, 1)
			},
			want: "composition_commit",
		},
		{
			name: "missing clipboard write",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"clipboard_write": true`, `"clipboard_write": false`, 1)
			},
			want: "clipboard_write",
		},
		{
			name: "missing clipboard host abi",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"clipboard_host_abi": true`, `"clipboard_host_abi": false`, 1)
			},
			want: "clipboard_host_abi",
		},
		{
			name: "missing composition trace commit",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"commit":true`, `"commit":false`, 1)
			},
			want: "composition_trace.commit",
		},
		{
			name: "missing clipboard owned copy",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"clipboard_owned_copy": true`, `"clipboard_owned_copy": false`, 1)
			},
			want: "clipboard_owned_copy",
		},
		{
			name: "borrowed view storage",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"borrowed_view_storage": false`, `"borrowed_view_storage": true`, 1)
			},
			want: "borrowed_view_storage",
		},
		{
			name: "missing safe view lifetime",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"safe_view_lifetime_checked": true`, `"safe_view_lifetime_checked": false`, 1)
			},
			want: "safe_view_lifetime_checked",
		},
		{
			name: "missing target host composition trace",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"target_host_composition_trace": true`, `"target_host_composition_trace": false`, 1)
			},
			want: "target_host_composition_trace",
		},
		{
			name: "rich text production claim",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"rich_text_production_claim": false`, `"rich_text_production_claim": true`, 1)
			},
			want: "rich_text_production_claim",
		},
		{
			name: "bidi production claim",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"bidi_production_claim": false`, `"bidi_production_claim": true`, 1)
			},
			want: "bidi_production_claim",
		},
		{
			name: "missing settings reference trace",
			mutate: func(raw string) string {
				return strings.Replace(raw, `    {"source":"examples/surface_morph_settings.tetra","trace":"settings text field trace","focus":true,"selection":true,"clipboard":true,"composition":true,"multiline":true,"pass":true},
`, "", 1)
			},
			want: "examples/surface_morph_settings.tetra",
		},
		{
			name: "shaping plan claims bidi",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"bidi":"nonclaim-full-bidi-v1"`, `"bidi":"full-bidi-production-v1"`, 1)
			},
			want: "text_shaping_plan.bidi",
		},
		{
			name: "missing target evidence",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "target": "headless",`+"\n", "", 1)
			},
			want: "target",
		},
		{
			name: "missing process evidence",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke --mode headless-release-text-input","ran":true,"pass":true,"exit_code":0}
  ]`, `"processes": []`, 1)
			},
			want: "process evidence",
		},
		{
			name: "missing composition case evidence",
			mutate: func(raw string) string {
				return strings.Replace(raw, `    {"name":"release text input composition commit","kind":"positive","ran":true,"pass":true},
`, "", 1)
			},
			want: "composition commit",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.mutate(string(validSurfaceTextInputReportJSON())))
			err := ValidateTextInputReport(raw)
			if err == nil {
				t.Fatalf("expected text-input report to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsTextFocusInputMissingCaretAndDeleteEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessTextFocusInputSurfaceReportJSON()), `"caret":"1",`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"text focus input backspace delete","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected text focus input report without caret/delete evidence to fail")
	}
	for _, want := range []string{"caret", "backspace delete"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsTextFocusInputMissingTabRoutingEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessTextFocusInputSurfaceReportJSON()), `,
    {"name":"text focus input Tab changes focus","kind":"positive","ran":true,"pass":true}`, ``, 1)
	raw = strings.Replace(raw, `{"order":7,"kind":"key_down","target_component":"TextInputApp","dispatch_path":["TextInputApp"],"handled":true,"pass":true,"x":0,"y":0,"key":9`, `{"order":7,"kind":"key_down","target_component":"TextInputApp","dispatch_path":["TextInputApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected text focus input report without Tab routing evidence to fail")
	}
	for _, want := range []string{"Tab", "focus routing"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func validSurfaceTextInputReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.text-input.v1",
  "target": "headless",
  "source": "examples/surface_release_text_input.tetra",
  "level": "production-text-input-v1",
  "experimental": false,
  "production_claim": true,
  "storage": "owned-utf8-byte-buffer",
  "utf8_validation": true,
  "invalid_utf8_rejected": true,
  "caret": true,
  "selection": true,
  "selection_clipboard_transfer": true,
  "multiline": true,
  "backspace": true,
  "delete": true,
  "home_end": true,
  "arrow_left_right": true,
  "composition_events": true,
  "composition_commit": true,
  "composition_cancel": true,
  "clipboard_read": true,
  "clipboard_write": true,
  "clipboard_host_abi": true,
  "clipboard_owned_copy": true,
  "target_host_composition_trace": true,
  "composition_trace": {"start":true,"update":true,"commit":true,"cancel":true},
  "text_shaping_plan": {"quality_level":"scoped-text-shaping-plan-v1","fallback_fonts":true,"grapheme_boundaries":"byte-offset-codepoint-v1","line_breaking":"newline-storage-plus-wrap-plan-v1","bidi":"nonclaim-full-bidi-v1","rich_text":"nonclaim-rich-text-editor-v1"},
  "reference_traces": [
    {"source":"examples/surface_morph_settings.tetra","trace":"settings text field trace","focus":true,"selection":true,"clipboard":true,"composition":true,"multiline":true,"pass":true},
    {"source":"examples/surface_morph_editor_shell.tetra","trace":"editor shell text area trace","focus":true,"selection":true,"clipboard":true,"composition":true,"multiline":true,"pass":true}
  ],
  "unsupported_claims": ["full-rich-text-editor","full-bidi-shaping","grapheme-cluster-caret","ide-grade-editor"],
  "rich_text_production_claim": false,
  "bidi_production_claim": false,
  "full_editor_production_claim": false,
  "borrowed_view_storage": false,
  "safe_view_lifetime_checked": true,
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke --mode headless-release-text-input","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-release-text-input","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":4096},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":2048}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "cases": [
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"release text input ASCII insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input UTF-8 insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input invalid UTF-8 rejected","kind":"negative","ran":true,"pass":true,"expected_error":"invalid utf8 rejected"},
    {"name":"release text input multiline storage","kind":"positive","ran":true,"pass":true},
    {"name":"release text input caret home end arrows","kind":"positive","ran":true,"pass":true},
    {"name":"release text input selection replacement","kind":"positive","ran":true,"pass":true},
    {"name":"release text input selection clipboard transfer","kind":"positive","ran":true,"pass":true},
    {"name":"release text input backspace delete","kind":"positive","ran":true,"pass":true},
    {"name":"release text input clipboard owned copy transfer","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition start update","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition commit","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition cancel","kind":"positive","ran":true,"pass":true},
    {"name":"release text input shaping plan scoped","kind":"positive","ran":true,"pass":true},
    {"name":"settings reference text input trace","kind":"positive","ran":true,"pass":true},
    {"name":"editor reference text input trace","kind":"positive","ran":true,"pass":true},
    {"name":"release text input safe view lifetime checked","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`)
}
func validHeadlessTextFocusInputSurfaceReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "headless",
  "host": "linux-x64",
  "runtime": "surface-headless",
  "surface_schema": "tetra.surface.v1",
  "host_abi": "tetra.surface.host-abi.v1",
  "host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false},
  "source": "examples/surface_textbox_app.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_textbox_app.tetra -o /tmp/surface-artifacts/surface-textbox-app","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-textbox-app","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-textbox-app","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":69657},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":13015}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "components": [
    {"id":"TextInputApp","type":"examples.surface_textbox_app.TextInputApp","bounds":{"x":0,"y":0,"w":400,"h":240},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"focused_component":"SubmitButton","width":"400","height":"240","resize_count":"1","accessibility_role":"none"}},
    {"id":"TextBox","type":"examples.surface_textbox_app.TextBox","parent":"TextInputApp","bounds":{"x":32,"y":64,"w":224,"h":44},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"focused":"false","buffer":"Z","caret":"1","text_len":"1","backspace_count":"1","delete_count":"1","accessibility_role":"label"}},
    {"id":"SubmitButton","type":"examples.surface_textbox_app.ActionButton","parent":"TextInputApp","bounds":{"x":32,"y":128,"w":128,"h":44},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"focused":"true","press_count":"1","key_count":"1","accessibility_role":"button"}}
  ],
  "events": [
    {"order":1,"kind":"mouse_up","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":48,"y":96,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0],"before_state":{"TextInputApp.focused_component":"none","TextBox.focused":"false"},"after_state":{"TextInputApp.focused_component":"TextBox","TextBox.focused":"true"}},
    {"order":2,"kind":"text_input","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":320,"height":200,"timestamp_ms":1,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,1,2],"before_state":{"TextBox.buffer":"","TextBox.caret":"0","TextBox.text_len":"0"},"after_state":{"TextBox.buffer":"OK","TextBox.caret":"2","TextBox.text_len":"2"}},
    {"order":3,"kind":"key_down","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":37,"width":320,"height":200,"timestamp_ms":2,"buffer_slots":[6,0,0,0,37,320,200,2,0],"before_state":{"TextBox.buffer":"OK","TextBox.caret":"2"},"after_state":{"TextBox.buffer":"OK","TextBox.caret":"1"}},
    {"order":4,"kind":"key_down","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":8,"width":320,"height":200,"timestamp_ms":3,"buffer_slots":[6,0,0,0,8,320,200,3,0],"before_state":{"TextBox.buffer":"OK","TextBox.caret":"1"},"after_state":{"TextBox.buffer":"K","TextBox.caret":"0"}},
    {"order":5,"kind":"key_down","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":46,"width":320,"height":200,"timestamp_ms":4,"buffer_slots":[6,0,0,0,46,320,200,4,0],"before_state":{"TextBox.buffer":"K","TextBox.caret":"0"},"after_state":{"TextBox.buffer":"","TextBox.caret":"0"}},
    {"order":6,"kind":"text_input","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":320,"height":200,"timestamp_ms":5,"text_len":1,"text_bytes_hex":"5a","buffer_slots":[8,0,0,0,0,320,200,5,1],"before_state":{"TextBox.buffer":"","TextBox.caret":"0","TextBox.text_len":"0"},"after_state":{"TextBox.buffer":"Z","TextBox.caret":"1","TextBox.text_len":"1"}},
    {"order":7,"kind":"key_down","target_component":"TextInputApp","dispatch_path":["TextInputApp"],"handled":true,"pass":true,"x":0,"y":0,"key":9,"width":320,"height":200,"timestamp_ms":6,"buffer_slots":[6,0,0,0,9,320,200,6,0],"before_state":{"TextInputApp.focused_component":"TextBox","TextBox.focused":"true","SubmitButton.focused":"false"},"after_state":{"TextInputApp.focused_component":"SubmitButton","TextBox.focused":"false","SubmitButton.focused":"true"}},
    {"order":8,"kind":"key_down","target_component":"SubmitButton","dispatch_path":["TextInputApp","SubmitButton"],"handled":true,"pass":true,"x":0,"y":0,"key":32,"width":320,"height":200,"timestamp_ms":7,"buffer_slots":[6,0,0,0,32,320,200,7,0],"before_state":{"SubmitButton.press_count":"0","TextBox.buffer":"Z"},"after_state":{"SubmitButton.press_count":"1","TextBox.buffer":"Z"}},
    {"order":9,"kind":"resize","target_component":"TextInputApp","dispatch_path":["TextInputApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":8,"buffer_slots":[2,0,0,0,0,400,240,8,0],"before_state":{"TextInputApp.width":"320","TextInputApp.focused_component":"SubmitButton"},"after_state":{"TextInputApp.width":"400","TextInputApp.focused_component":"SubmitButton"}}
  ],
  "frames": [
    {"order":1,"width":320,"height":200,"stride":1280,"checksum":"1111111111111111111111111111111111111111111111111111111111111111","presented":true},
    {"order":2,"width":400,"height":240,"stride":1600,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}
  ],
  "state_transitions": [
    {"order":1,"component":"TextInputApp","field":"focused_component","before":"none","after":"TextBox","cause":"mouse_up"},
    {"order":2,"component":"TextBox","field":"buffer","before":"","after":"OK","cause":"text_input"},
    {"order":3,"component":"TextBox","field":"caret","before":"2","after":"1","cause":"key_down"},
    {"order":4,"component":"TextBox","field":"buffer","before":"OK","after":"K","cause":"backspace"},
    {"order":5,"component":"TextBox","field":"buffer","before":"K","after":"","cause":"delete"},
    {"order":6,"component":"TextBox","field":"buffer","before":"","after":"Z","cause":"text_input"},
    {"order":7,"component":"TextInputApp","field":"focused_component","before":"TextBox","after":"SubmitButton","cause":"tab"},
    {"order":8,"component":"SubmitButton","field":"press_count","before":"0","after":"1","cause":"key_down"},
    {"order":9,"component":"TextInputApp","field":"width","before":"320","after":"400","cause":"resize"}
  ],
  "cases": [
    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input click focuses TextBox","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input Tab changes focus","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input keyboard routes only focused component","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input text insertion","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input caret movement","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input backspace delete","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input resize preserves focus","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input rendered frame update","kind":"positive","ran":true,"pass":true},
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"state transition","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"},
    {"name":"headless event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"headless framebuffer checksum","kind":"positive","ran":true,"pass":true},
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}
  ]
}`)
}
