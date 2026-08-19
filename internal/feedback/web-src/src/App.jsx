import React, { useEffect, useRef, useState, useCallback } from "react";
import { Excalidraw, exportToBlob, serializeAsJSON } from "@excalidraw/excalidraw";
// Excalidraw 0.17.x injects its own styles from the bundle; there is no
// separate CSS entry to import.

const params = new URLSearchParams(window.location.search);
const sessionId = params.get("session");
const base = sessionId ? `/session/${sessionId}` : null;

// Debounce helper for onChange -> PUT /draft.
function debounce(fn, ms) {
  let t = null;
  return (...args) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...args), ms);
  };
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
        try {
          const draftResp = await fetch(`${base}/draft`);
          if (draftResp.ok) {
            const d = await draftResp.json();
            if (d && (d.elements || d.appState)) scene = d;
          }
        } catch {
          // no draft yet — start empty
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
      if (!base) return;
      const scene = JSON.parse(
        serializeAsJSON(elements, appState, {}, "local"),
      );
      try {
        await fetch(`${base}/draft`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(scene),
        });
        setStatus("Draft saved.");
      } catch (e) {
        setStatus(`Draft error: ${e}`);
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
      setStatus("Submitted. You may close this window.");
    } catch (e) {
      doneRef.current = false;
      setStatus(`Submit error: ${e}`);
    }
  }, [api]);

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
