import React, { useEffect, useRef, useState, useCallback } from "react";
import {
  Excalidraw,
  exportToBlob,
  serializeAsJSON,
  convertToExcalidrawElements,
} from "@excalidraw/excalidraw";
import { parseMermaidToExcalidraw } from "@excalidraw/mermaid-to-excalidraw";
// Excalidraw 0.17.x injects its own styles from the bundle; there is no
// separate CSS entry to import.

const params = new URLSearchParams(window.location.search);
const sessionId = params.get("session");
const base = sessionId ? `/session/${sessionId}` : null;

// Debounce helper for onChange -> PUT /draft. The returned function exposes a
// cancel() so a pending trailing call can be dropped on submit (property [12]),
// avoiding a draft PUT that races the server shutdown.
function debounce(fn, ms) {
  let t = null;
  const wrapped = (...args) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...args), ms);
  };
  wrapped.cancel = () => clearTimeout(t);
  return wrapped;
}

export default function App() {
  const [api, setApi] = useState(null);
  const [status, setStatus] = useState("Loading session…");
  // property [6]: prior draft is restored as initialData on reopen. Mermaid
  // never uses initialData: it is inserted only after the empty editor mounts.
  const [initialData, setInitialData] = useState(null);
  const [mermaidSeed, setMermaidSeed] = useState(null);
  const [ready, setReady] = useState(false);
  const [terminal, setTerminal] = useState(null);
  const doneRef = useRef(false);
  const draftRequestRef = useRef(null);
  const sceneReadyRef = useRef(false);
  const seedStartedRef = useRef(false);

  useEffect(() => {
    if (!base) {
      setStatus("No session id in URL (?session=<id> missing).");
      sceneReadyRef.current = true;
      setInitialData({});
      setReady(true);
      return;
    }
    (async () => {
      try {
        // Load the request; a prior draft (if any) is the Excalidraw scene.
        const resp = await fetch(base);
        const req = resp.ok ? await resp.json() : {};
        let scene = {};
        let haveDraft = false;
        try {
          const draftResp = await fetch(`${base}/draft`);
          if (draftResp.ok) {
            const d = await draftResp.json();
            if (d && (d.elements || d.appState)) {
              scene = d;
              haveDraft = true;
            }
          }
        } catch {
          // no draft yet — fall through to mermaid seed or blank canvas
        }
        // property [10]: seed from Mermaid only when there is no prior draft,
        // so draft recovery (property [6]) always takes precedence. A new
        // Mermaid scene starts as an empty mounted editor and is inserted by the
        // public imperative API after editor and font initialization.
        if (!haveDraft && req.mermaid) {
          setMermaidSeed(req.mermaid);
        } else {
          sceneReadyRef.current = true;
        }
        setInitialData(scene);
        setStatus(req.instructions ? `Task: ${req.instructions}` : `Session ${sessionId}`);
      } catch (e) {
        sceneReadyRef.current = true;
        setInitialData({});
        setStatus(`Load error: ${e}`);
      } finally {
        setReady(true);
      }
    })();
  }, []);

  // Excalidraw's own Mermaid interaction parses only after the editor mounts,
  // then inserts through a private API that redraws bound text (and explicitly
  // loads fonts in Safari). The public API has no equivalent insertion method,
  // so wait for the imperative API and browser fonts, preserve conversion
  // output unchanged, add helper files, update the scene, and fit exactly once.
  useEffect(() => {
    if (!api || !mermaidSeed || seedStartedRef.current) return undefined;
    seedStartedRef.current = true;
    let cancelled = false;

    (async (mermaid) => {
      try {
        // A blank canvas does not itself request the drawing font. Ensure the
        // editable default font participates in FontFaceSet readiness before
        // conversion, matching Excalidraw's Safari insertion safeguard.
        await document.fonts.load("20px Virgil");
        await document.fonts.ready;
        if (cancelled) return;
        const { elements, files = {} } = await parseMermaidToExcalidraw(mermaid);
        const converted = convertToExcalidrawElements(elements);
        if (cancelled) return;
        api.addFiles(Object.values(files));
        sceneReadyRef.current = true;
        api.updateScene({ elements: converted });
        api.scrollToContent(converted, {
          fitToViewport: true,
          viewportZoomFactor: 0.9,
          animate: false,
        });
      } catch {
        // Parse failure retains the mounted blank canvas as the fallback.
        sceneReadyRef.current = true;
      }
    })(mermaidSeed);

    return () => {
      cancelled = true;
    };
  }, [api, mermaidSeed]);

  // property [5]: onChange -> debounced PUT /draft carrying the scene.
  const putDraft = useCallback(
    debounce(async (elements, appState, files) => {
      // property [12]/[13]: once the session is submitted the server is
      // shutting down; skip the draft PUT entirely so a trailing debounced
      // call cannot race the close and surface a spurious error.
      if (!base || doneRef.current) return;
      const scene = JSON.parse(
        serializeAsJSON(elements, appState, files ?? {}, "local"),
      );
      const controller = new AbortController();
      draftRequestRef.current = controller;
      try {
        await fetch(`${base}/draft`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(scene),
          signal: controller.signal,
        });
        if (!doneRef.current) setStatus("Draft saved.");
      } catch (e) {
        // Once submit or cancel begins, an aborted/in-flight autosave must not
        // overwrite the terminal state after the session server shuts down.
        if (!doneRef.current) setStatus(`Draft error: ${e}`);
      } finally {
        if (draftRequestRef.current === controller) {
          draftRequestRef.current = null;
        }
      }
    }, 400),
    [],
  );

  const onChange = useCallback(
    (elements, appState, files) => {
      // Do not let the intentionally empty mounted editor overwrite the session
      // while Mermaid conversion is waiting for fonts.
      if (sceneReadyRef.current) putDraft(elements, appState, files);
    },
    [putDraft],
  );

  // property [5]: Done exports scene (.excalidraw JSON via PUT /draft) + png
  // (POST /png), then POST /submit.
  const onDone = useCallback(async () => {
    if (!api || !base || doneRef.current) return;
    doneRef.current = true;
    // property [12]: drop any pending debounced draft so it cannot race the
    // server shutdown that submit triggers.
    putDraft.cancel();
    draftRequestRef.current?.abort();
    setStatus("Submitting…");
    const elements = api.getSceneElements();
    const appState = api.getAppState();
    const files = api.getFiles();
    const sceneJSON = serializeAsJSON(elements, appState, files, "local");
    try {
      await fetch(`${base}/draft`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: sceneJSON,
      });
      const blob = await exportToBlob({
        elements,
        appState,
        files,
        mimeType: "image/png",
      });
      await fetch(`${base}/png`, {
        method: "POST",
        headers: { "Content-Type": "image/png" },
        body: blob,
      });
      await fetch(`${base}/submit`, { method: "POST" });
      setStatus("Submitted. Closing…");
      // The session result is captured; close the window. window.close() only
      // works for script-opened tabs, so fall back to a terminal message.
      window.close();
      setStatus("Submitted. You may close this window.");
    } catch (e) {
      doneRef.current = false;
      setStatus(`Submit error: ${e}`);
    }
  }, [api, putDraft]);

  const onCancel = useCallback(async () => {
    if (!base || doneRef.current) return;
    doneRef.current = true;
    putDraft.cancel();
    draftRequestRef.current?.abort();
    setStatus("Cancelling…");
    try {
      await fetch(`${base}/cancel`, { method: "POST" });
    } catch {
      // A server that has already observed cancellation may disappear before
      // the browser receives its response. Cancellation remains terminal.
    }
    setStatus("Cancelled. Closing…");
    window.close();
    // Browser security blocks close() for tabs not opened by script. Keep a
    // stable terminal fallback instead of leaving the live editor/autosave UI.
    setTerminal("cancelled");
    setStatus("Cancelled. You may close this window.");
  }, [putDraft]);

  if (!ready) {
    return <div style={{ padding: "1rem" }}>Loading…</div>;
  }

  return (
    <>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "0.75rem",
          padding: "0.5rem 0.75rem",
          borderBottom: "1px solid #e4e4e7",
          font: "14px system-ui, sans-serif",
        }}
      >
        <strong>nn ask</strong>
        <span style={{ color: "#555", flex: 1 }}>{status}</span>
        <button onClick={onCancel} disabled={terminal === "cancelled"}>Cancel</button>
        <button
          onClick={onDone}
          disabled={terminal === "cancelled"}
          style={{ background: "#6366f1", color: "#fff", border: "none", padding: "0.4rem 0.9rem", cursor: "pointer" }}
        >
          Done
        </button>
      </div>
      {terminal === "cancelled" ? (
        <div
          role="status"
          aria-label="Canvas session cancelled"
          style={{ flex: 1, padding: "2rem", font: "16px system-ui, sans-serif" }}
        >
          This Canvas session is cancelled. No further drafts will be saved.
        </div>
      ) : (
        <div style={{ flex: 1, minHeight: 0 }}>
          <Excalidraw
            initialData={initialData}
            onChange={onChange}
            excalidrawAPI={setApi}
          />
        </div>
      )}
    </>
  );
}
