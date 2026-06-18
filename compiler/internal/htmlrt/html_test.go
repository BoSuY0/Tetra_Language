package htmlrt

import (
	"strings"
	"testing"
)

func TestAppendEscapedEscapesHTMLSpecialsAndPreservesUTF8(t *testing.T) {
	got := string(AppendEscaped(nil, `<tag attr="Tom & Jerry's">日本語</tag>`))
	want := `&lt;tag attr=&quot;Tom &amp; Jerry&apos;s&quot;&gt;日本語&lt;/tag&gt;`
	if got != want {
		t.Fatalf("AppendEscaped() = %q, want %q", got, want)
	}
}

func TestRenderFortunesSortsByMessageAndEscapesMessages(t *testing.T) {
	body := string(RenderFortunes(nil, []Fortune{
		{ID: 7, Message: "Zulu"},
		{ID: 11, Message: `<script>alert("x");</script>`},
		{ID: 5, Message: "Alpha & Beta"},
	}))

	if !strings.HasPrefix(
		body,
		"<!DOCTYPE html><html><head><title>Fortunes</title></head><body><table>",
	) {
		t.Fatalf("RenderFortunes missing compact HTML prefix:\n%s", body)
	}
	if strings.Contains(body, `<script>alert("x");</script>`) {
		t.Fatalf("RenderFortunes left an unescaped script tag:\n%s", body)
	}
	for _, want := range []string{
		"<tr><th>id</th><th>message</th></tr>",
		"<tr><td>11</td><td>&lt;script&gt;alert(&quot;x&quot;);&lt;/script&gt;</td></tr>",
		"<tr><td>5</td><td>Alpha &amp; Beta</td></tr>",
		"<tr><td>7</td><td>Zulu</td></tr>",
		"</table></body></html>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("RenderFortunes missing %q:\n%s", want, body)
		}
	}
	if !(strings.Index(body, "&lt;script&gt;") < strings.Index(body, "Alpha &amp; Beta") &&
		strings.Index(body, "Alpha &amp; Beta") < strings.Index(body, "Zulu")) {
		t.Fatalf("RenderFortunes did not sort by raw message:\n%s", body)
	}
}

func FuzzAppendEscapedRemovesRawHTMLSpecials(f *testing.F) {
	for _, seed := range []string{
		`<script>alert("x");</script>`,
		"Tom & Jerry's",
		"plain",
		"日本語",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		got := string(AppendEscaped(nil, input))
		if strings.ContainsAny(got, "<>\"'") {
			t.Fatalf("escaped output contains raw HTML special: input=%q output=%q", input, got)
		}
	})
}
