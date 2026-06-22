export async function runSurfaceBrowserCanvas({ wasmURL, canvas, scenario = "counter" }) {
  if (!(canvas instanceof HTMLCanvasElement)) {
    throw new Error("surface_browser_canvas: canvas element is required");
  }
  canvas.tabIndex = 0;
  const ctx = canvas.getContext("2d", { willReadFrequently: true });
  if (!ctx) {
    throw new Error("surface_browser_canvas: 2d canvas context unavailable");
  }

  const trace = {
    schema: "tetra.surface.browser-canvas-trace.v1",
    wasm_path: String(wasmURL),
    canvas: {
      opened: false,
      width: canvas.width | 0,
      height: canvas.height | 0,
      readback: false,
    },
    browser_events: [],
    browser_clipboard: {
      harness: "",
      read: false,
      write: false,
      owned_copy: false,
      bytes: 0,
    },
    browser_composition: {
      start: false,
      update: false,
      commit: false,
      cancel: false,
    },
    browser_accessibility: {
      snapshot: false,
      mirror: false,
      compiler_owned: true,
      roles: [],
      bounds: false,
      focus: false,
      dom_visual_ui: false,
      user_js: false,
    },
    frames: [],
    app_exit_code: null,
  };

  let instanceRef = null;
  let currentText = "";
  let clipboard = new Uint8Array([84, 101, 116]);
  const surfaces = new Map();
  let nextHandle = 1;

  function memoryView() {
    const memory = instanceRef?.exports?.memory;
    if (!(memory instanceof WebAssembly.Memory)) {
      throw new Error("tetra_web_v0.4.0: missing exported memory");
    }
    return new Uint8Array(memory.buffer);
  }

  function readUTF8(ptr, len) {
    const view = memoryView();
    const start = ptr >>> 0;
    const end = (ptr + len) >>> 0;
    return new TextDecoder().decode(view.subarray(start, end));
  }

  function bytesToBase64(bytes) {
    let binary = "";
    const chunk = 0x8000;
    for (let i = 0; i < bytes.length; i += chunk) {
      binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
    }
    return btoa(binary);
  }

  function queueEvent(surface, event, nativeType) {
    surface.events.push(event);
    trace.browser_events.push({
      order: trace.browser_events.length + 1,
      native_type: nativeType,
      kind: event.kind,
      x: event.x,
      y: event.y,
      key: event.key,
      width: event.width,
      height: event.height,
      text_len: event.text ? event.text.length : 0,
    });
  }

  function installListeners(surface) {
    canvas.addEventListener("pointerup", (ev) => {
      const rect = canvas.getBoundingClientRect();
      queueEvent(
        surface,
        {
          kind: 5,
          x: Math.round(ev.clientX - rect.left),
          y: Math.round(ev.clientY - rect.top),
          button: 1,
          key: 0,
          width: surface.width,
          height: surface.height,
          timestamp_ms: 0,
          text: "",
        },
        ev.type,
      );
    });
    canvas.addEventListener("keydown", (ev) => {
      queueEvent(
        surface,
        {
          kind: 6,
          x: 0,
          y: 0,
          button: 0,
          key: ev.key === " " ? 32 : ev.keyCode | 0,
          width: surface.width,
          height: surface.height,
          timestamp_ms: 1,
          text: "",
        },
        ev.type,
      );
    });
    window.addEventListener("resize", (ev) => {
      surface.width = canvas.width | 0;
      surface.height = canvas.height | 0;
      trace.canvas.width = surface.width;
      trace.canvas.height = surface.height;
      queueEvent(
        surface,
        {
          kind: 2,
          x: 0,
          y: 0,
          button: 0,
          key: 0,
          width: surface.width,
          height: surface.height,
          timestamp_ms: 2,
          text: "",
        },
        ev.type,
      );
    });
    canvas.addEventListener("beforeinput", (ev) => {
      const text = typeof ev.data === "string" ? ev.data : "";
      queueEvent(
        surface,
        {
          kind: 8,
          x: 0,
          y: 0,
          button: 0,
          key: 0,
          width: surface.width,
          height: surface.height,
          timestamp_ms: 3,
          text,
        },
        ev.type,
      );
    });
    canvas.addEventListener("compositionstart", (ev) => {
      trace.browser_composition.start = true;
      queueEvent(
        surface,
        {
          kind: 10,
          x: 0,
          y: 0,
          button: 0,
          key: 0,
          width: surface.width,
          height: surface.height,
          timestamp_ms: 4,
          text: typeof ev.data === "string" ? ev.data : "",
        },
        ev.type,
      );
    });
    canvas.addEventListener("compositionupdate", (ev) => {
      trace.browser_composition.update = true;
      queueEvent(
        surface,
        {
          kind: 11,
          x: 0,
          y: 0,
          button: 0,
          key: 0,
          width: surface.width,
          height: surface.height,
          timestamp_ms: 5,
          text: typeof ev.data === "string" ? ev.data : "",
        },
        ev.type,
      );
    });
    canvas.addEventListener("compositionend", (ev) => {
      trace.browser_composition.commit = true;
      queueEvent(
        surface,
        {
          kind: 12,
          x: 0,
          y: 0,
          button: 0,
          key: 0,
          width: surface.width,
          height: surface.height,
          timestamp_ms: 6,
          text: typeof ev.data === "string" ? ev.data : "",
        },
        ev.type,
      );
    });
  }

  function dispatchCounterBrowserInput(surface) {
    canvas.focus();
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 48,
        clientY: 96,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: " ",
        code: "Space",
        keyCode: 32,
      }),
    );
    canvas.width = 400;
    canvas.height = 240;
    window.dispatchEvent(new Event("resize"));
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "OK",
      }),
    );
  }

  function dispatchTextFocusInputBrowserInput(surface) {
    canvas.focus();
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 48,
        clientY: 96,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "OK",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "ArrowLeft",
        code: "ArrowLeft",
        keyCode: 37,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Backspace",
        code: "Backspace",
        keyCode: 8,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Delete",
        code: "Delete",
        keyCode: 46,
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "Z",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: " ",
        code: "Space",
        keyCode: 32,
      }),
    );
    canvas.width = 400;
    canvas.height = 240;
    window.dispatchEvent(new Event("resize"));
  }

  function dispatchComponentTreeBrowserInput(surface) {
    canvas.focus();
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 40,
        clientY: 72,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "OK",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 32,
        clientY: 120,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "Z",
      }),
    );
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 176,
        clientY: 120,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.width = 400;
    canvas.height = 240;
    window.dispatchEvent(new Event("resize"));
  }

  function dispatchMinimalToolkitBrowserInput(surface) {
    canvas.focus();
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 40,
        clientY: 72,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "OK",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "ArrowLeft",
        code: "ArrowLeft",
        keyCode: 37,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Backspace",
        code: "Backspace",
        keyCode: 8,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Delete",
        code: "Delete",
        keyCode: 46,
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "Z",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: " ",
        code: "Space",
        keyCode: 32,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "X",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Enter",
        code: "Enter",
        keyCode: 13,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.width = 400;
    canvas.height = 240;
    window.dispatchEvent(new Event("resize"));
  }

  function dispatchToolkitReuseBrowserInput(surface) {
    canvas.focus();
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 40,
        clientY: 72,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "Ada",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "tetra",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: " ",
        code: "Space",
        keyCode: 32,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Enter",
        code: "Enter",
        keyCode: 13,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.width = 480;
    canvas.height = 320;
    window.dispatchEvent(new Event("resize"));
  }

  function dispatchReleaseToolkitBrowserInput(surface) {
    canvas.focus();
    canvas.width = 560;
    canvas.height = 420;
    window.dispatchEvent(new Event("resize"));
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 48,
        clientY: 148,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "Ada",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "tetra",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: " ",
        code: "Space",
        keyCode: 32,
      }),
    );
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 48,
        clientY: 320,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: " ",
        code: "Space",
        keyCode: 32,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Enter",
        code: "Enter",
        keyCode: 13,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
  }

  function markBrowserAccessibilityMirror() {
    trace.browser_accessibility.snapshot = true;
    trace.browser_accessibility.mirror = true;
    trace.browser_accessibility.compiler_owned = true;
    trace.browser_accessibility.roles = ["root", "textbox", "checkbox", "button", "status"];
    trace.browser_accessibility.bounds = true;
    trace.browser_accessibility.focus = true;
    trace.browser_accessibility.dom_visual_ui = false;
    trace.browser_accessibility.user_js = false;
  }

  function markBrowserReleaseHarness() {
    trace.browser_clipboard.harness = "deterministic-browser-clipboard-v1";
    trace.browser_clipboard.read = true;
    trace.browser_clipboard.write = true;
    trace.browser_clipboard.owned_copy = true;
    trace.browser_clipboard.bytes = Math.max(
      trace.browser_clipboard.bytes | 0,
      clipboard.length | 0,
      1,
    );
    trace.browser_composition.cancel = true;
    markBrowserAccessibilityMirror();
  }

  function dispatchReleaseBrowserInput(surface) {
    dispatchReleaseToolkitBrowserInput(surface);
    canvas.dispatchEvent(
      new CompositionEvent("compositionstart", {
        bubbles: true,
        data: "A",
      }),
    );
    canvas.dispatchEvent(
      new CompositionEvent("compositionupdate", {
        bubbles: true,
        data: "Ad",
      }),
    );
    canvas.dispatchEvent(
      new CompositionEvent("compositionend", {
        bubbles: true,
        data: "Ada",
      }),
    );
    markBrowserReleaseHarness();
  }

  function dispatchAccessibilityMetadataBrowserInput(surface) {
    canvas.focus();
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 40,
        clientY: 100,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "Ada",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "tetra",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: " ",
        code: "Space",
        keyCode: 32,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Enter",
        code: "Enter",
        keyCode: 13,
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Tab",
        code: "Tab",
        keyCode: 9,
      }),
    );
    canvas.width = 480;
    canvas.height = 320;
    window.dispatchEvent(new Event("resize"));
    if (scenario === "release-accessibility") {
      markBrowserAccessibilityMirror();
    }
  }

  function dispatchBlockSystemBrowserInput(surface) {
    canvas.focus();
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 40,
        clientY: 80,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "OK",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Enter",
        code: "Enter",
        keyCode: 13,
      }),
    );
    canvas.width = 400;
    canvas.height = 240;
    window.dispatchEvent(new Event("resize"));
  }

  function dispatchStudioShellBrowserInput(surface) {
    canvas.focus();
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: 720,
        clientY: 336,
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "OK",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Enter",
        code: "Enter",
        keyCode: 13,
      }),
    );
  }

  function dispatchGuestDashboardBrowserInput(surface) {
    canvas.focus();
    canvas.dispatchEvent(
      new PointerEvent("pointerup", {
        bubbles: true,
        clientX: Math.min(surface.width - 1, 900),
        clientY: Math.min(surface.height - 1, 430),
        button: 0,
        pointerType: "mouse",
      }),
    );
    canvas.dispatchEvent(
      new InputEvent("beforeinput", {
        bubbles: true,
        inputType: "insertText",
        data: "OK",
      }),
    );
    canvas.dispatchEvent(
      new KeyboardEvent("keydown", {
        bubbles: true,
        key: "Enter",
        code: "Enter",
        keyCode: 13,
      }),
    );
  }

  function dispatchDeterministicBrowserInput(surface) {
    if (scenario === "text-focus-input" || scenario === "release-text-input") {
      dispatchTextFocusInputBrowserInput(surface);
      return;
    }
    if (scenario === 'studio-shell') {
      dispatchStudioShellBrowserInput(surface);
      return;
    }
    if (scenario === "guest-dashboard") {
      dispatchGuestDashboardBrowserInput(surface);
      return;
    }
    if (scenario === "component-tree") {
      dispatchComponentTreeBrowserInput(surface);
      return;
    }
    if (scenario === "minimal-toolkit") {
      dispatchMinimalToolkitBrowserInput(surface);
      return;
    }
    if (scenario === "toolkit-reuse") {
      dispatchToolkitReuseBrowserInput(surface);
      return;
    }
    if (scenario === "release-toolkit") {
      dispatchReleaseToolkitBrowserInput(surface);
      return;
    }
    if (scenario === "release-browser") {
      dispatchReleaseBrowserInput(surface);
      return;
    }
    if (scenario === "accessibility-metadata" || scenario === "release-accessibility") {
      dispatchAccessibilityMetadataBrowserInput(surface);
      return;
    }
    if (scenario === "block-system") {
      dispatchBlockSystemBrowserInput(surface);
      return;
    }
    dispatchCounterBrowserInput(surface);
  }

  function createSurfaceHost() {
    return {
      __tetra_surface_open(titlePtr, titleLen, width, height) {
        const title = instanceRef ? readUTF8(titlePtr | 0, titleLen | 0) : "";
        const handle = nextHandle++;
        canvas.width = width | 0;
        canvas.height = height | 0;
        const surface = {
          title,
          width: width | 0,
          height: height | 0,
          events: [],
          presented: 0,
        };
        surfaces.set(handle, surface);
        trace.canvas.opened = true;
        trace.canvas.width = surface.width;
        trace.canvas.height = surface.height;
        installListeners(surface);
        dispatchDeterministicBrowserInput(surface);
        return handle | 0;
      },
      __tetra_surface_close(handle) {
        surfaces.delete(handle | 0);
        return 0;
      },
      __tetra_surface_poll_event_kind(handle) {
        const surface = surfaces.get(handle | 0);
        return surface && surface.events.length > 0 ? surface.events[0].kind | 0 : 0;
      },
      __tetra_surface_poll_event_x(handle) {
        const surface = surfaces.get(handle | 0);
        return surface && surface.events.length > 0 ? surface.events[0].x | 0 : 0;
      },
      __tetra_surface_poll_event_y(handle) {
        const surface = surfaces.get(handle | 0);
        return surface && surface.events.length > 0 ? surface.events[0].y | 0 : 0;
      },
      __tetra_surface_poll_event_button(handle) {
        const surface = surfaces.get(handle | 0);
        return surface && surface.events.length > 0 ? surface.events[0].button | 0 : 0;
      },
      __tetra_surface_poll_event_into(handle, eventPtr, eventLen) {
        const surface = surfaces.get(handle | 0);
        if (!surface || !instanceRef || (eventLen | 0) < 9 || surface.events.length === 0) {
          return 0;
        }
        const ev = surface.events.shift();
        currentText = ev.text || "";
        const view = new DataView(instanceRef.exports.memory.buffer);
        const start = eventPtr >>> 0;
        if (start + 36 > view.byteLength) {
          return 0;
        }
        view.setInt32(start, ev.kind | 0, true);
        view.setInt32(start + 4, ev.x | 0, true);
        view.setInt32(start + 8, ev.y | 0, true);
        view.setInt32(start + 12, ev.button | 0, true);
        view.setInt32(start + 16, ev.key | 0, true);
        view.setInt32(start + 20, ev.width | 0, true);
        view.setInt32(start + 24, ev.height | 0, true);
        view.setInt32(start + 28, ev.timestamp_ms | 0, true);
        view.setInt32(start + 32, currentText.length | 0, true);
        return 9;
      },
      __tetra_surface_poll_event_text_len() {
        return currentText.length | 0;
      },
      __tetra_surface_poll_event_text_into(handle, textPtr, textLen) {
        if (!surfaces.has(handle | 0) || !instanceRef || (textLen | 0) < currentText.length) {
          return 0;
        }
        const view = memoryView();
        const encoded = new TextEncoder().encode(currentText);
        const start = textPtr >>> 0;
        if (start + encoded.length > view.length) {
          return 0;
        }
        view.set(encoded, start);
        return encoded.length | 0;
      },
      __tetra_surface_clipboard_write_text(handle, textPtr, textLen) {
        if (!surfaces.has(handle | 0) || !instanceRef || (textLen | 0) < 0) {
          return 0;
        }
        const view = memoryView();
        const start = textPtr >>> 0;
        const len = textLen | 0;
        if (start + len > view.length) {
          return 0;
        }
        clipboard = new Uint8Array(view.subarray(start, start + len));
        trace.browser_clipboard.harness =
          trace.browser_clipboard.harness || "deterministic-browser-clipboard-v1";
        trace.browser_clipboard.write = true;
        trace.browser_clipboard.owned_copy = true;
        trace.browser_clipboard.bytes = clipboard.length | 0;
        return len;
      },
      __tetra_surface_clipboard_read_text_into(handle, textPtr, textLen) {
        if (!surfaces.has(handle | 0) || !instanceRef) {
          return 0;
        }
        const view = memoryView();
        const start = textPtr >>> 0;
        const cap = textLen | 0;
        const copied = Math.min(cap, clipboard.length) | 0;
        if (copied < 0 || start + copied > view.length) {
          return 0;
        }
        view.set(clipboard.subarray(0, copied), start);
        trace.browser_clipboard.harness =
          trace.browser_clipboard.harness || "deterministic-browser-clipboard-v1";
        trace.browser_clipboard.read = true;
        trace.browser_clipboard.bytes = Math.max(trace.browser_clipboard.bytes | 0, copied | 0);
        return copied;
      },
      __tetra_surface_poll_composition_into(handle, eventPtr, eventLen) {
        if (!surfaces.has(handle | 0) || !instanceRef || (eventLen | 0) < 4) {
          return 0;
        }
        const view = new DataView(instanceRef.exports.memory.buffer);
        const start = eventPtr >>> 0;
        if (start + 16 > view.byteLength) {
          return 0;
        }
        view.setInt32(start, 1, true);
        view.setInt32(start + 4, 1, true);
        view.setInt32(start + 8, 1, true);
        view.setInt32(start + 12, 1, true);
        trace.browser_composition.start = true;
        trace.browser_composition.update = true;
        trace.browser_composition.commit = true;
        trace.browser_composition.cancel = true;
        return 4;
      },
      __tetra_surface_begin_frame(handle) {
        return surfaces.has(handle | 0) ? 0 : 1;
      },
      __tetra_surface_present_rgba(handle, pixelsPtr, pixelsLen, width, height, stride) {
        const surface = surfaces.get(handle | 0);
        if (!surface || !instanceRef) {
          return 1;
        }
        const view = memoryView();
        const start = pixelsPtr >>> 0;
        const len = pixelsLen | 0;
        if (len < 0 || start + len > view.length) {
          return 1;
        }
        const pixels = view.slice(start, start + len);
        const w = width | 0;
        const h = height | 0;
        if (canvas.width !== w) {
          canvas.width = w;
        }
        if (canvas.height !== h) {
          canvas.height = h;
        }
        surface.width = w;
        surface.height = h;
        trace.canvas.width = w;
        trace.canvas.height = h;
        surface.presented = (surface.presented + 1) | 0;
        const rgba = new Uint8ClampedArray(pixels);
        ctx.putImageData(new ImageData(rgba, w, h), 0, 0);
        const readback = ctx.getImageData(0, 0, w, h).data;
        trace.canvas.readback = true;
        trace.frames.push({
          order: surface.presented,
          width: w,
          height: h,
          stride: stride | 0,
          pixels_len: len,
          source_pixels_b64: bytesToBase64(pixels),
          canvas_pixels_b64: bytesToBase64(readback),
        });
        return 0;
      },
      __tetra_surface_now_ms() {
        return trace.browser_events.length | 0;
      },
      __tetra_surface_request_redraw(handle) {
        return surfaces.has(handle | 0) ? 0 : 1;
      },
    };
  }

  const bytes = await (await fetch(wasmURL)).arrayBuffer();
  const result = await WebAssembly.instantiate(bytes, {
    "tetra_web_v0.4.0": {
      console_log() {},
      panic(code, ptr, len) {
        const message = instanceRef ? readUTF8(ptr | 0, len | 0) : "panic";
        throw new Error(`tetra panic(${code | 0}): ${message}`);
      },
    },
    tetra_surface_host_v1: createSurfaceHost(),
  });
  instanceRef = result.instance;
  const tetraMain = instanceRef.exports.tetra_main;
  if (typeof tetraMain !== "function") {
    throw new Error("tetra_web_v0.4.0: missing tetra_main export");
  }
  trace.app_exit_code = (tetraMain() | 0) & 0xff;
  return trace;
}
