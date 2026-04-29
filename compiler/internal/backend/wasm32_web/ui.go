package wasm32_web

import "strings"

func UIModule(uiJSONFile string) []byte {
	if uiJSONFile == "" {
		uiJSONFile = "app.ui.json"
	}
	uiJSONFile = escapeJSLiteral(uiJSONFile)
	src := strings.Join([]string{
		"const TETRA_UI_URL = new URL(\"" + uiJSONFile + "\", import.meta.url);",
		"",
		"function addLine(root, text) {",
		"  const line = document.createElement(\"div\");",
		"  line.textContent = text;",
		"  root.appendChild(line);",
		"}",
		"",
		"export async function mountTetraUI(root = document.body, uiURL = TETRA_UI_URL) {",
		"  const response = await fetch(uiURL);",
		"  if (!response.ok) {",
		"    throw new Error(\"tetra_ui: failed to fetch metadata: \" + response.status);",
		"  }",
		"  const bundle = await response.json();",
		"  if (!bundle || bundle.schema !== \"tetra.ui.v1\") {",
		"    throw new Error(\"tetra_ui: unsupported schema: \" + String(bundle && bundle.schema));",
		"  }",
		"  const host = document.createElement(\"section\");",
		"  host.setAttribute(\"data-tetra-ui\", \"v1\");",
		"  addLine(host, \"Tetra UI Shell\");",
		"  addLine(host, \"runtime: metadata-only preview (no event dispatch)\");",
		"  for (const view of (bundle.views || [])) {",
		"    addLine(host, \"view \" + view.name + \" (state: \" + view.state_type + \")\");",
		"    for (const binding of (view.bindings || [])) {",
		"      addLine(host, \"  bind \" + binding.name + \": \" + binding.type + \" <- \" + binding.source);",
		"    }",
		"    for (const event of (view.events || [])) {",
		"      addLine(host, \"  event \" + event.name + \" -> \" + event.command);",
		"    }",
		"  }",
		"  root.appendChild(host);",
		"  return bundle;",
		"}",
	}, "\n")
	return []byte(src + "\n")
}

func UIHTMLPage(wasmFileName string, loaderFileName string, uiModuleFileName string) []byte {
	if wasmFileName == "" {
		wasmFileName = "app.wasm"
	}
	if loaderFileName == "" {
		loaderFileName = "app.mjs"
	}
	if uiModuleFileName == "" {
		uiModuleFileName = "app.ui.web.mjs"
	}
	wasmFileName = escapeJSLiteral(wasmFileName)
	loaderFileName = escapeJSLiteral(loaderFileName)
	uiModuleFileName = escapeJSLiteral(uiModuleFileName)
	html := strings.Join([]string{
		"<!doctype html>",
		"<html lang=\"en\">",
		"<head>",
		"  <meta charset=\"utf-8\">",
		"  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">",
		"  <title>Tetra UI Preview</title>",
		"</head>",
		"<body>",
		"  <main id=\"app\">Loading...</main>",
		"  <script type=\"module\">",
		"    import { runTetra } from \"./" + loaderFileName + "\";",
		"    import { mountTetraUI } from \"./" + uiModuleFileName + "\";",
		"    const root = document.getElementById(\"app\");",
		"    try {",
		"      await mountTetraUI(root);",
		"      const exitCode = await runTetra(new URL(\"./" + wasmFileName + "\", import.meta.url));",
		"      const done = document.createElement(\"div\");",
		"      done.textContent = \"tetra_main exit=\" + (exitCode | 0);",
		"      root.appendChild(done);",
		"    } catch (err) {",
		"      root.textContent = String(err);",
		"    }",
		"  </script>",
		"</body>",
		"</html>",
	}, "\n")
	return []byte(html + "\n")
}
