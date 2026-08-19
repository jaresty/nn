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

// mermaidToScene converts a mermaid diagram string into an Excalidraw scene
// usable as initialData. Returns null on parse failure so the caller can fall
// back to a blank canvas.
async function mermaidToScene(mermaid) {
  try {
    const { elements } = await parseMermaidToExcalidraw(mermaid);
    return { elements: convertToExcalidrawElements(elements) };
  } catch {
    return null;
  }
}

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
  // property [6]: prior draft is restored as initialData on reopen.
  const [initialData, setInitialData] = useState(null);
  const [ready, setReady] = useState(false);
  const doneRef = useRef(false);

  useEffect(() => {
    if (!base) {
      setStatus("No session id in URL (?session=<id> missing).");
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
        // property [10]: seed from mermaid only when there is no prior draft,
        // so draft recovery (property [6]) always takes precedence.
        if (!haveDraft && req.mermaid) {
          const seeded = await mermaidToScene(req.mermaid);
          if (seeded) scene = seeded;
        }
        setInitialData(scene);
        setStatus(req.instructions ? `Task: ${req.instructions}` : `Session ${sessionId}`);
      } catch (e) {
        setInitialData({});
        setStatus(`Load error: ${e}`);
      } finally {
        setReady(true);
      }
    })();
  }, []);

  // property [5]: onChange -> debounced PUT /draft carrying the scene.
  const putDraft = useCallback(
    debounce(async (elements, appState) => {
      // property [12]/[13]: once the session is submitted the server is
      // shutting down; skip the draft PUT entirely so a trailing debounced
      // call cannot race the close and surface a spurious error.
      if (!base || doneRef.current) return;
      const scene = JSON.parse(
        serializeAsJSON(elements, appState, {}, "local"),
      );
      try {
        await fetch(`${base}/draft`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(scene),
        });
        if (!doneRef.current) setStatus("Draft saved.");
      } catch (e) {
        // property [13]: never overwrite the terminal submitted status.
        if (!doneRef.current) setStatus(`Draft error: ${e}`);
      }
    }, 400),
    [],
  );

  const onChange = useCallback(
    (elements, appState) => putDraft(elements, appState),
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
    if (!base) return;
    await fetch(`${base}/cancel`, { method: "POST" });
    setStatus("Cancelled. You may close this window.");
  }, []);

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
        <button onClick={onCancel}>Cancel</button>
        <button
          onClick={onDone}
          style={{ background: "#6366f1", color: "#fff", border: "none", padding: "0.4rem 0.9rem", cursor: "pointer" }}
        >
          Done
        </button>
      </div>
      <div style={{ flex: 1, minHeight: 0 }}>
        <Excalidraw
          initialData={initialData}
          onChange={onChange}
          excalidrawAPI={(a) => setApi(a)}
        />
      </div>
    </>
  );
}
