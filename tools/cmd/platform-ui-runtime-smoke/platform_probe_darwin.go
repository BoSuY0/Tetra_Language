//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func runPlatformWindowProbe(target string) (platformWindowProbeResult, error) {
	if target != "macos-x64" {
		return platformWindowProbeResult{}, fmt.Errorf("macOS platform probe cannot run target %s", target)
	}
	tmpDir, err := os.MkdirTemp("", "tetra-platform-ui-appkit-*")
	if err != nil {
		return platformWindowProbeResult{}, err
	}
	defer os.RemoveAll(tmpDir)
	sourcePath := filepath.Join(tmpDir, "probe.swift")
	binPath := filepath.Join(tmpDir, "probe")
	source := `import AppKit
import Darwin

let app = NSApplication.shared
app.setActivationPolicy(.regular)

final class Handler: NSObject {
    var clicked = false
    weak var title: NSTextField?

    @objc func save(_ sender: NSButton) {
        clicked = true
        title?.stringValue = "Saved"
    }
}

let rect = NSRect(x: 0, y: 0, width: 320, height: 200)
let window = NSWindow(contentRect: rect, styleMask: [.titled, .closable], backing: .buffered, defer: false)
window.title = "Tetra UI Runtime Smoke"
let root = NSStackView(frame: NSRect(x: 0, y: 0, width: 320, height: 200))
root.orientation = .vertical
root.alignment = .leading
root.spacing = 8
root.edgeInsets = NSEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
let title = NSTextField(labelWithString: "Editing")
let input = NSTextField(string: "tetra")
let list = NSPopUpButton(frame: .zero, pullsDown: false)
list.addItems(withTitles: ["item-1", "item-2"])
let button = NSButton(title: "Save", target: nil, action: nil)
let handler = Handler()
handler.title = title
button.target = handler
button.action = #selector(Handler.save(_:))
root.addArrangedSubview(title)
root.addArrangedSubview(input)
root.addArrangedSubview(list)
root.addArrangedSubview(button)
window.contentView = root
window.layoutIfNeeded()
window.makeKeyAndOrderFront(nil)
input.becomeFirstResponder()
input.stringValue = "tetra-ui"
list.selectItem(at: 1)
let actionSent = app.sendAction(#selector(Handler.save(_:)), to: handler, from: button)
var timerFired = false
let timer = Timer(timeInterval: 0.01, repeats: false) { _ in
    timerFired = true
    title.stringValue = "Saved after timer"
    window.contentView?.needsDisplay = true
}
RunLoop.current.add(timer, forMode: .default)
let until = Date().addingTimeInterval(1.0)
while !timerFired && Date() < until {
    if let event = app.nextEvent(matching: .any, until: Date().addingTimeInterval(0.01), inMode: .default, dequeue: true) {
        app.sendEvent(event)
    }
    RunLoop.current.run(mode: .default, before: Date().addingTimeInterval(0.01))
}
if !actionSent || !handler.clicked || !timerFired || input.stringValue != "tetra-ui" || list.titleOfSelectedItem != "item-2" {
    fputs("AppKit UI runtime probe did not dispatch expected state changes actionSent=\(actionSent) clicked=\(handler.clicked) timerFired=\(timerFired) input=\(input.stringValue) selected=\(list.titleOfSelectedItem ?? "nil")\n", stderr)
    exit(1)
}
window.close()
print("ok")
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		return platformWindowProbeResult{}, err
	}
	if out, err := exec.Command("swiftc", "-o", binPath, sourcePath).CombinedOutput(); err != nil {
		return platformWindowProbeResult{}, fmt.Errorf("swiftc AppKit probe failed: %w: %s", err, string(out))
	}
	if out, err := exec.Command(binPath).CombinedOutput(); err != nil {
		return platformWindowProbeResult{}, fmt.Errorf("AppKit probe failed: %w: %s", err, string(out))
	}
	return platformWindowProbeResult{
		API:     "cocoa-appkit",
		Markers: []string{"platform-window-create:cocoa-appkit", "platform-widget-tree:ok", "platform-event-dispatch:ok", "platform-timer:ok", "platform-redraw:ok", "platform-window-close:cocoa-appkit"},
	}, nil
}
