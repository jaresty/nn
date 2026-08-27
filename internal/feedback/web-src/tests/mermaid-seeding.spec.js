import { readFileSync } from "node:fs";
import { test, expect } from "@playwright/test";

const appSource = readFileSync(new URL("../src/App.jsx", import.meta.url), "utf8");

async function openMermaidSession(page, id, mermaid) {
  const drafts = [];
  const pngs = [];

  await page.route(`**/session/${id}{,/**}`, async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const base = `/session/${id}`;

    if (url.pathname === base && request.method() === "GET") {
      await route.fulfill({ json: { id, surface: "canvas", mermaid } });
      return;
    }
    if (url.pathname === `${base}/draft` && request.method() === "GET") {
      await route.fulfill({ status: 404 });
      return;
    }
    if (url.pathname === `${base}/draft` && request.method() === "PUT") {
      drafts.push(JSON.parse(request.postData()));
      await route.fulfill({ status: 204 });
      return;
    }
    if (url.pathname === `${base}/png` && request.method() === "POST") {
      pngs.push(request.postDataBuffer());
      await route.fulfill({ status: 204 });
      return;
    }
    if (url.pathname === `${base}/submit` && request.method() === "POST") {
      await route.fulfill({ status: 204 });
      return;
    }
    await route.fulfill({ status: 404 });
  });

  await page.goto(`/?session=${id}`);
  await expect(page.locator(".excalidraw")).toBeVisible();
  await expect.poll(() => drafts.length).toBeGreaterThan(0);
  return { drafts, pngs };
}

test("Mermaid seeding uses the mounted public API after fonts are ready", async ({ page }) => {
  const id = "seed-lifecycle";
  const drafts = [];

  await page.addInitScript(() => {
    let releaseFontReady;
    const fontReady = new Promise((resolve) => {
      releaseFontReady = resolve;
    });
    Object.defineProperty(document.fonts, "ready", {
      configurable: true,
      get() {
        window.__fontReadyReads = (window.__fontReadyReads ?? 0) + 1;
        return fontReady.then(() => document.fonts);
      },
    });
    window.__releaseFontReady = releaseFontReady;
  });
  await page.route(`**/session/${id}{,/**}`, async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const session = `/session/${id}`;
    if (path === session && request.method() === "GET") {
      await route.fulfill({
        json: {
          id,
          surface: "canvas",
          mermaid: "flowchart TD\n  A[Mounted] --> B[Font ready]",
        },
      });
      return;
    }
    if (path === `${session}/draft` && request.method() === "GET") {
      await route.fulfill({ status: 404 });
      return;
    }
    if (path === `${session}/draft` && request.method() === "PUT") {
      drafts.push(JSON.parse(request.postData()));
      await route.fulfill({ status: 204 });
      return;
    }
    await route.fulfill({ status: 404 });
  });

  await page.goto(`/?session=${id}`);
  await expect(page.locator(".excalidraw")).toBeVisible();
  await expect.poll(() => page.evaluate(() => window.__fontReadyReads ?? 0)).toBe(1);
  await page.waitForTimeout(500);
  expect(drafts).toEqual([]);

  await page.evaluate(() => window.__releaseFontReady());
  await expect.poll(() => drafts.at(-1)?.elements?.length ?? 0).toBeGreaterThan(0);
});

test("invalid Mermaid exposes a safe editable fallback instead of an ordinary blank canvas", async ({ page }) => {
  const id = "invalid-mermaid";
  let draftPuts = 0;
  await page.route(`**/session/${id}{,/**}`, async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const session = `/session/${id}`;
    if (path === session && request.method() === "GET") {
      await route.fulfill({ json: { id, surface: "canvas", mermaid: "flowchart TD\n  A[broken" } });
      return;
    }
    if (path === `${session}/draft` && request.method() === "GET") {
      await route.fulfill({ status: 404 });
      return;
    }
    if (path === `${session}/draft` && request.method() === "PUT") {
      draftPuts++;
      await route.fulfill({ status: 204 });
      return;
    }
    await route.fulfill({ status: 404 });
  });

  await page.goto(`/?session=${id}`);
  await expect(page.locator(".excalidraw")).toBeVisible();
  await expect(page.getByText("Mermaid seed could not be rendered. Blank canvas is available.", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Done" })).toBeEnabled();
  await expect(page.getByText(/broken image/i)).toHaveCount(0);

  // Draw into the fallback so autosave runs; the conversion warning must remain
  // visible instead of being replaced by the ordinary "Draft saved" status.
  await page.keyboard.press("r");
  await page.mouse.move(320, 260);
  await page.mouse.down();
  await page.mouse.move(430, 340);
  await page.mouse.up();
  await expect.poll(() => draftPuts).toBeGreaterThan(0);
  await expect(page.getByText("Mermaid seed could not be rendered. Blank canvas is available.", { exact: true })).toBeVisible();
});

test("Mermaid insertion lifecycle keeps conversion authoritative and has no geometry mutator", () => {
  expect(appSource).not.toMatch(
    /padConvertedLabelContainers|BOUND_TEXT_PADDING|measureText|actualBoundingBox/,
  );

  const lifecycle = [
    "await document.fonts.ready",
    "parseMermaidToExcalidraw(mermaid)",
    "convertToExcalidrawElements(elements)",
    "api.addFiles(Object.values(files))",
    "api.updateScene({ elements: converted })",
    "api.scrollToContent(converted",
  ];
  const positions = lifecycle.map((token) => appSource.indexOf(token));
  expect(positions, `missing lifecycle token(s): ${JSON.stringify(lifecycle)}`).not.toContain(-1);
  expect(positions).toEqual([...positions].sort((a, b) => a - b));
});

test("graph-image Mermaid seeds retain helper files in drafts and PNG export", async ({ page }) => {
  const { drafts, pngs } = await openMermaidSession(
    page,
    "graph-image",
    `mindmap
  root((Knowledge))
    First branch
    Second branch`,
  );

  await expect.poll(() => Object.keys(drafts.at(-1)?.files ?? {}).length).toBe(1);
  const seeded = drafts.at(-1);
  const image = seeded.elements.find((element) => element.type === "image");
  expect(image).toBeTruthy();
  expect(seeded.files[image.fileId]?.dataURL).toMatch(/^data:image\/svg\+xml;base64,/);

  await page.getByRole("button", { name: "Done" }).click();
  await expect.poll(() => pngs.length).toBe(1);
  expect(pngs[0].subarray(0, 8)).toEqual(
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
  );
  expect(Object.keys(drafts.at(-1).files)).toEqual([image.fileId]);
});

test("a restored draft stays authoritative and does not rerun Mermaid seeding", async ({ page }) => {
  const id = "draft-authority";
  const drafts = [];
  let restoredDraft = null;
  let submitRequests = 0;
  let mermaid = "flowchart TD\n  ORIGINAL[Original editable seed]";

  await page.route(`**/session/${id}{,/**}`, async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const session = `/session/${id}`;
    if (path === session && request.method() === "GET") {
      await route.fulfill({ json: { id, surface: "canvas", mermaid } });
      return;
    }
    if (path === `${session}/draft` && request.method() === "GET") {
      if (restoredDraft) {
        await route.fulfill({ json: restoredDraft });
      } else {
        await route.fulfill({ status: 404 });
      }
      return;
    }
    if (path === `${session}/draft` && request.method() === "PUT") {
      restoredDraft = JSON.parse(request.postData());
      drafts.push(restoredDraft);
      await route.fulfill({ status: 204 });
      return;
    }
    if (
      (path === `${session}/png` || path === `${session}/submit`) &&
      request.method() === "POST"
    ) {
      if (path === `${session}/submit`) submitRequests += 1;
      await route.fulfill({ status: 204 });
      return;
    }
    await route.fulfill({ status: 404 });
  });

  await page.goto(`/?session=${id}`);
  await expect(page.locator(".excalidraw")).toBeVisible();
  await expect.poll(() => drafts.at(-1)?.elements?.length ?? 0).toBeGreaterThan(0);
  const originalIds = restoredDraft.elements.map((element) => element.id).sort();

  mermaid = "flowchart TD\n  REPLACEMENT[Mermaid must not replace the draft]";
  await page.reload();
  await expect(page.locator(".excalidraw")).toBeVisible();
  await page.waitForTimeout(600);
  await page.getByRole("button", { name: "Done" }).click();
  await expect.poll(() => submitRequests).toBe(1);

  const submitted = drafts.at(-1);
  expect(submitted.elements.map((element) => element.id).sort()).toEqual(originalIds);
  expect(submitted.elements.some((element) =>
    element.type === "text" && element.originalText?.includes("Original editable seed")
  )).toBe(true);
  expect(submitted.elements.some((element) =>
    element.type === "text" && element.originalText?.includes("must not replace")
  )).toBe(false);
});

test("actual multiline explanation seed keeps visible glyphs inside container interiors", async ({ page }) => {
  const { drafts } = await openMermaidSession(
    page,
    "multiline",
    `flowchart TD
  NOTE["\`Illustrative layout and inferred relationships — not literal notebook structure.
Source IDs identify evidence; only edges explicitly marked STORED are notebook edges.\`"]
  CLAIM["\`Claim: consultation must retain the complete positioned navigation frame
Source: 20260825143000-ask-as-navigation-human-consultation\`"]
  EVIDENCE["\`Evidence: selected note bodies supply sections, claims, assumptions, and implications
Provenance: 20260824101530-graph-selection-answer-composition-handoff\`"]
  CLAIM -->|"derived explanation"| EVIDENCE
  NOTE -.-> CLAIM`,
  );

  const scene = drafts.at(-1);
  const labels = scene.elements.filter(
    (element) =>
      element.type === "text" &&
      scene.elements.some(
        (container) => container.id === element.containerId && container.type === "rectangle",
      ),
  );
  const claim = labels.find((element) =>
    element.originalText.startsWith("Claim: consultation"),
  );
  expect(claim).toBeTruthy();
  expect(claim.height).toBeGreaterThan(claim.fontSize * claim.lineHeight);
  expect(labels.some((element) => element.originalText.includes("<br>"))).toBe(false);
  expect(scene.elements.some((element) => element.type === "image")).toBe(false);
  expect(scene.elements.some((element) => element.type === "arrow")).toBe(true);

  const measurements = await page.evaluate(async ({ labels }) => {
    const fontFamilies = {
      1: "Virgil",
      2: "Helvetica",
      3: "Cascadia",
      4: "Assistant",
    };
    await Promise.all(labels.map((text) =>
      document.fonts.load(
        `${text.fontSize}px ${fontFamilies[text.fontFamily] ?? "sans-serif"}`,
        text.text,
      ),
    ));
    await document.fonts.ready;
    const canvas = document.createElement("canvas");
    const context = canvas.getContext("2d");
    return labels.map((text) => {
      const family = fontFamilies[text.fontFamily] ?? "sans-serif";
      context.font = `${text.fontSize}px ${family}`;
      // `text` is Excalidraw's visible wrapped string; `originalText` retains
      // the editable Mermaid source label.
      const lines = text.text.split("\n");
      const glyphs = lines.map((line) => {
        const metrics = context.measureText(line);
        const advance = metrics.width;
        const lineX = text.textAlign === "center"
          ? text.x + (text.width - advance) / 2
          : text.textAlign === "right"
            ? text.x + text.width - advance
            : text.x;
        return {
          line,
          left: lineX - metrics.actualBoundingBoxLeft,
          right: lineX + metrics.actualBoundingBoxRight,
        };
      });
      return {
        id: text.id,
        text: text.originalText,
        textBounds: {
          left: text.x,
          top: text.y,
          right: text.x + text.width,
          bottom: text.y + text.height,
        },
        glyphBounds: {
          left: Math.min(...glyphs.map((glyph) => glyph.left)),
          right: Math.max(...glyphs.map((glyph) => glyph.right)),
        },
      };
    });
  }, { labels });

  const interiorPadding = 5;
  const violations = [];
  for (const measurement of measurements) {
    const box = scene.elements.find(
      (element) => element.id === labels.find((text) => text.id === measurement.id).containerId,
    );
    expect(box?.type).toBe("rectangle");
    expect(box.boundElements).toContainEqual({ type: "text", id: measurement.id });
    const interior = {
      left: box.x + interiorPadding,
      top: box.y + interiorPadding,
      right: box.x + box.width - interiorPadding,
      bottom: box.y + box.height - interiorPadding,
    };
    for (const [kind, bounds] of [
      ["text", measurement.textBounds],
      ["glyph", { ...measurement.glyphBounds, top: measurement.textBounds.top, bottom: measurement.textBounds.bottom }],
    ]) {
      if (
        bounds.left < interior.left ||
        bounds.top < interior.top ||
        bounds.right > interior.right ||
        bounds.bottom > interior.bottom
      ) {
        violations.push({
          label: measurement.text,
          kind,
          bounds,
          interior,
        });
      }
    }
  }
  expect(
    violations,
    `visible multiline label bounds escaped their container interiors:\n${JSON.stringify(violations, null, 2)}`,
  ).toEqual([]);

  for (const arrow of scene.elements.filter((element) => element.type === "arrow")) {
    if (arrow.startBinding) {
      expect(scene.elements.some((element) => element.id === arrow.startBinding.elementId)).toBe(true);
    }
    if (arrow.endBinding) {
      expect(scene.elements.some((element) => element.id === arrow.endBinding.elementId)).toBe(true);
    }
  }
});

test("Cancel attempts close, stops autosave, and survives server shutdown", async ({ page }) => {
  const id = "cancel-lifecycle";
  let serverUp = true;
  let cancelRequests = 0;
  let draftPutsAfterCancel = 0;

  await page.addInitScript(() => {
    window.__closeAttempts = 0;
    window.close = () => {
      window.__closeAttempts += 1;
    };
  });
  await page.route(`**/session/${id}{,/**}`, async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const session = `/session/${id}`;
    if (path === session && request.method() === "GET") {
      await route.fulfill({ json: { id, surface: "canvas" } });
      return;
    }
    if (path === `${session}/draft` && request.method() === "GET") {
      await route.fulfill({ status: 404 });
      return;
    }
    if (path === `${session}/cancel` && request.method() === "POST") {
      cancelRequests += 1;
      await route.fulfill({ status: 200 });
      serverUp = false;
      return;
    }
    if (path === `${session}/draft` && request.method() === "PUT") {
      if (!serverUp) {
        draftPutsAfterCancel += 1;
        await route.abort("connectionrefused");
      } else {
        await route.fulfill({ status: 204 });
      }
      return;
    }
    await route.fulfill({ status: 404 });
  });

  await page.goto(`/?session=${id}`);
  await expect(page.locator(".excalidraw")).toBeVisible();
  await page.getByRole("button", { name: "Cancel" }).click();
  await expect.poll(() => cancelRequests).toBe(1);
  await expect.poll(() => page.evaluate(() => window.__closeAttempts)).toBe(1);
  await expect(page.getByText("Cancelled. You may close this window.")).toBeVisible();
  await expect(page.getByRole("status", { name: "Canvas session cancelled" })).toBeVisible();
  await expect(page.locator(".excalidraw")).toHaveCount(0);
  await page.waitForTimeout(600);
  expect(draftPutsAfterCancel).toBe(0);
  await expect(page.getByText(/Draft error:/)).toHaveCount(0);
});

test("initial camera fits every element in a complete wide Mermaid seed", async ({ page }) => {
  const labels = Array.from({ length: 10 }, (_, i) =>
    `N${i}["\`Stage ${i + 1}\neditable detail\`"]`,
  );
  const declaration = labels.join(" --> ");
  const { drafts } = await openMermaidSession(
    page,
    "camera-fit",
    `flowchart LR\n  ${declaration}`,
  );

  await expect.poll(async () => {
    const elements = drafts.at(-1)?.elements?.filter((element) => !element.isDeleted) ?? [];
    const encodedCamera = await page.getByTestId("canvas-surface").getAttribute("data-camera");
    const camera = encodedCamera ? JSON.parse(encodedCamera) : null;
    const zoom = camera?.zoom;
    if (!elements.length || !zoom || !camera.width || !camera.height) return false;
    const left = -camera.scrollX;
    const top = -camera.scrollY;
    const right = left + camera.width / zoom;
    const bottom = top + camera.height / zoom;
    return elements.every((element) =>
      element.x >= left - 2 &&
      element.y >= top - 2 &&
      element.x + element.width <= right + 2 &&
      element.y + element.height <= bottom + 2
    );
  }).toBe(true);
});
