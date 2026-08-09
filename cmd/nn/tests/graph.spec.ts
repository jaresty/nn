import { test, expect, Page } from '@playwright/test';
import { startServer, stopServer, BASE_URL } from './fixture';

test.beforeAll(async () => { await startServer(); });
test.afterAll(() => stopServer());

// Helper: load the graph page and wait for nodes to render
async function loadGraph(page: Page) {
  await page.goto(BASE_URL);
  // Wait for at least one node circle to appear
  await page.waitForSelector('g circle', { timeout: 8000 });
}

// Helper: shift-click the first node
async function shiftClickNode(page: Page, index = 0) {
  const nodes = page.locator('g.node, svg g[data-id], #graph-svg g > g').nth(index);
  // Use the SVG g elements that contain circles
  const circles = page.locator('svg g circle');
  const circle = circles.nth(index);
  await circle.click({ modifiers: ['Shift'] });
}

// ── [P1a/1b] tray-selected CSS class on shift-click toggle ──────────────────

test('[P1a] shift-click adds tray-selected class to node g element', async ({ page }) => {
  await loadGraph(page);
  const circle = page.locator('svg g circle').first();
  await circle.click({ modifiers: ['Shift'] });
  // The parent g of the circle should have the tray-selected class
  const hasTraySelected = await circle.evaluate(el => el.parentElement?.classList.contains('tray-selected'));
  expect(hasTraySelected).toBe(true);
});

test('[P1b] second shift-click removes tray-selected class', async ({ page }) => {
  await loadGraph(page);
  const circle = page.locator('svg g circle').first();
  await circle.click({ modifiers: ['Shift'] });
  const after1 = await circle.evaluate(el => el.parentElement?.classList.contains('tray-selected'));
  expect(after1).toBe(true);
  await circle.click({ modifiers: ['Shift'] });
  const after2 = await circle.evaluate(el => el.parentElement?.classList.contains('tray-selected'));
  expect(after2).toBe(false);
});

// ── [P2a/2b] #tray-count label ───────────────────────────────────────────────

test('[P2a] tray-count shows "0 selected" when tray is empty', async ({ page }) => {
  await loadGraph(page);
  await expect(page.locator('#tray-count')).toHaveText('0 selected');
});

test('[P2b] tray-count shows note titles when notes are selected', async ({ page }) => {
  await loadGraph(page);
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  const text = await page.locator('#tray-count').textContent();
  // Should contain a title (not just a number)
  expect(text).not.toMatch(/^\d+ selected$/);
  expect(text!.length).toBeGreaterThan(0);
});

// ── [P3a/3b] #btn-export opens #brief-modal ─────────────────────────────────

test('[P3b] #brief-modal is not open on page load', async ({ page }) => {
  await loadGraph(page);
  const modal = page.locator('#brief-modal');
  await expect(modal).not.toHaveAttribute('open');
});

test('[P3a] clicking Review & Export opens #brief-modal', async ({ page }) => {
  await loadGraph(page);
  // Select a node first
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  await page.locator('#btn-export').click();
  await expect(page.locator('#brief-modal')).toHaveAttribute('open');
});

// ── [P4a/4b] modal contains per-note title, ID, textarea, checkbox ───────────

test('[P4a] modal contains note title and ID for each selected note', async ({ page }) => {
  await loadGraph(page);
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  await page.locator('#btn-export').click();
  await expect(page.locator('#brief-modal')).toHaveAttribute('open');
  const modal = page.locator('#brief-modal');
  // Should contain a title element
  const titleEl = modal.locator('.brief-note-title').first();
  await expect(titleEl).toBeVisible();
  // Should contain an ID element
  const idEl = modal.locator('.brief-note-id').first();
  await expect(idEl).toBeVisible();
});

test('[P4b] modal contains annotation textarea and include-body checkbox per note', async ({ page }) => {
  await loadGraph(page);
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  await page.locator('#btn-export').click();
  const modal = page.locator('#brief-modal');
  await expect(modal.locator('textarea.brief-ann').first()).toBeVisible();
  await expect(modal.locator('input[type=checkbox].brief-body-toggle').first()).toBeVisible();
});

// ── [P5a/5b] export text content ────────────────────────────────────────────

test('[P5a] copy text contains note title, ID, and annotation', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await loadGraph(page);
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  await page.locator('#btn-export').click();
  // Fill in annotation
  await page.locator('textarea.brief-ann').first().fill('my annotation');
  await page.locator('#btn-copy').click();
  const clipText = await page.evaluate(() => navigator.clipboard.readText());
  expect(clipText).toContain('my annotation');
  // Should contain an ID-like string
  expect(clipText).toMatch(/[a-z]+-\d+|[0-9]{13}-[0-9]{4}/);
});

test('[P5b] copy text does not contain note body when Include body unchecked', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await loadGraph(page);
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  await page.locator('#btn-export').click();
  // Ensure checkbox is unchecked
  const checkbox = page.locator('input[type=checkbox].brief-body-toggle').first();
  if (await checkbox.isChecked()) await checkbox.uncheck();
  await page.locator('#btn-copy').click();
  const clipText = await page.evaluate(() => navigator.clipboard.readText());
  expect(clipText).not.toContain('Body of alpha note.');
});

// ── [P6a/6b] clipboard failure fallback ─────────────────────────────────────

test('[P6a] #brief-fallback becomes visible when clipboard throws', async ({ page }) => {
  await loadGraph(page);
  // Override clipboard to throw
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: () => Promise.reject(new Error('denied')) },
      configurable: true,
    });
  });
  await page.reload();
  await page.waitForSelector('g circle', { timeout: 8000 });
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  await page.locator('#btn-export').click();
  await page.locator('#btn-copy').click();
  await expect(page.locator('#brief-fallback')).toBeVisible();
});

test('[P6b] #brief-fallback text is pre-selected when shown', async ({ page }) => {
  await loadGraph(page);
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: () => Promise.reject(new Error('denied')) },
      configurable: true,
    });
  });
  await page.reload();
  await page.waitForSelector('g circle', { timeout: 8000 });
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  await page.locator('#btn-export').click();
  await page.locator('#btn-copy').click();
  await expect(page.locator('#brief-fallback')).toBeVisible();
  const selectionLength = await page.locator('#brief-fallback').evaluate((el: HTMLTextAreaElement) =>
    el.selectionEnd - el.selectionStart
  );
  expect(selectionLength).toBeGreaterThan(0);
});

// ── [P7a/7b] modal dismiss ───────────────────────────────────────────────────

test('[P7a] Escape key closes #brief-modal', async ({ page }) => {
  await loadGraph(page);
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  await page.locator('#btn-export').click();
  await expect(page.locator('#brief-modal')).toHaveAttribute('open');
  await page.keyboard.press('Escape');
  await expect(page.locator('#brief-modal')).not.toHaveAttribute('open');
});

test('[P7b] clicking dialog backdrop closes #brief-modal', async ({ page }) => {
  await loadGraph(page);
  await page.locator('svg g circle').first().click({ modifiers: ['Shift'] });
  await page.locator('#btn-export').click();
  await expect(page.locator('#brief-modal')).toHaveAttribute('open');
  // Dispatch a click event directly on the dialog element (simulates backdrop click)
  await page.locator('#brief-modal').evaluate((el: HTMLDialogElement) => {
    el.dispatchEvent(new MouseEvent('click', { bubbles: true, target: el } as MouseEventInit));
  });
  await expect(page.locator('#brief-modal')).not.toHaveAttribute('open');
});

// ── page load / DOM structure ─────────────────────────────────────────────────

test('page loads and renders node circles', async ({ page }) => {
  await loadGraph(page);
  const count = await page.locator('svg g circle').count();
  expect(count).toBeGreaterThan(0);
});

test('side panel is present in DOM', async ({ page }) => {
  await loadGraph(page);
  await expect(page.locator('#panel')).toBeAttached();
});

test('side panel opens on node click', async ({ page }) => {
  await loadGraph(page);
  await page.locator('svg g circle').first().click();
  await expect(page.locator('#right-col')).toHaveClass(/open/);
  await expect(page.locator('#panel-title')).not.toBeEmpty();
});

test('tray-count and btn-export present on page load', async ({ page }) => {
  await loadGraph(page);
  await expect(page.locator('#tray-count')).toBeAttached();
  await expect(page.locator('#btn-export')).toBeAttached();
});

test('btn-export is disabled on page load (empty tray)', async ({ page }) => {
  await loadGraph(page);
  await expect(page.locator('#btn-export')).toBeDisabled();
});

test('no chat elements in DOM', async ({ page }) => {
  await loadGraph(page);
  await expect(page.locator('#msg-panel')).not.toBeAttached();
  await expect(page.locator('#msg-input')).not.toBeAttached();
  await expect(page.locator('#msg-send')).not.toBeAttached();
});

// ── layout toggle ─────────────────────────────────────────────────────────────

test('layout toggle button present', async ({ page }) => {
  await loadGraph(page);
  await expect(page.locator('#btn-layout')).toBeAttached();
});

test('layout toggle activates grouped mode', async ({ page }) => {
  await loadGraph(page);
  await expect(page.locator('#btn-layout')).not.toHaveClass(/active/);
  await page.locator('#btn-layout').click();
  await expect(page.locator('#btn-layout')).toHaveClass(/active/);
});

// ── search interaction ────────────────────────────────────────────────────────

test('search dims non-matching nodes', async ({ page }) => {
  await loadGraph(page);
  await page.locator('#search-input').fill('xyzzynosuchterm');
  await page.waitForTimeout(400); // debounce
  const dimmed = await page.locator('g.node.search-dim').count();
  const total  = await page.locator('g.node').count();
  expect(dimmed).toBeGreaterThan(0);
  expect(dimmed).toBeLessThanOrEqual(total);
});

test('node click replaces search dim with click dim', async ({ page }) => {
  await loadGraph(page);
  await page.locator('#search-input').fill('xyzzynosuchterm');
  await page.waitForTimeout(400);
  const dimmedBefore = await page.locator('g.node.search-dim').count();
  expect(dimmedBefore).toBeGreaterThan(0);
  await page.locator('svg g circle').first().click();
  // After click, clicked node and its neighbors are undimmed; others remain dimmed
  // Clicked node itself must not be dimmed
  const clickedHasDim = await page.locator('svg g circle').first().evaluate(
    el => el.parentElement?.classList.contains('search-dim')
  );
  expect(clickedHasDim).toBe(false);
});

// ── node size ─────────────────────────────────────────────────────────────────

test('nodes have non-zero radius', async ({ page }) => {
  await loadGraph(page);
  const r = await page.locator('svg g circle').first().getAttribute('r');
  expect(Number(r)).toBeGreaterThan(0);
});

// ── status toggle ─────────────────────────────────────────────────────────────

test('status toggle button present', async ({ page }) => {
  await loadGraph(page);
  await expect(page.locator('#btn-status')).toBeAttached();
  await expect(page.locator('#btn-status')).toHaveText('All');
});

test('status toggle cycles through labels', async ({ page }) => {
  await loadGraph(page);
  const btn = page.locator('#btn-status');
  await btn.click(); await expect(btn).toHaveText('Draft');
  await btn.click(); await expect(btn).toHaveText('Reviewed');
  await btn.click(); await expect(btn).toHaveText('Permanent');
  await btn.click(); await expect(btn).toHaveText('All');
});

test('status toggle dims non-matching nodes', async ({ page }) => {
  await loadGraph(page);
  // fixture notes are draft; clicking to Reviewed should dim all of them
  await page.locator('#btn-status').click(); // → Draft
  await page.locator('#btn-status').click(); // → Reviewed
  const dimmed = await page.locator('g.node.search-dim').count();
  const total  = await page.locator('g.node').count();
  expect(dimmed).toBe(total);
});

test('status toggle All clears dim', async ({ page }) => {
  await loadGraph(page);
  await page.locator('#btn-status').click(); // Draft
  await page.locator('#btn-status').click(); // Reviewed
  await page.locator('#btn-status').click(); // Permanent
  await page.locator('#btn-status').click(); // All
  const dimmed = await page.locator('g.node.search-dim').count();
  expect(dimmed).toBe(0);
});

// ── label visibility at zoom ──────────────────────────────────────────────────

test('[PA] node labels hidden when zoomed out below threshold', async ({ page }) => {
  await loadGraph(page);
  // Force zoom to 0.1 (well below threshold)
  await page.evaluate(() => {
    (window as any).__testSetZoom(0.1);
  });
  const display = await page.locator('svg g.node text').first().evaluate(
    el => window.getComputedStyle(el).display
  );
  expect(display).toBe('none');
});

test('[PA] node labels visible when zoomed in above threshold', async ({ page }) => {
  await loadGraph(page);
  await page.evaluate(() => {
    (window as any).__testSetZoom(1.5);
  });
  const display = await page.locator('svg g.node text').first().evaluate(
    el => window.getComputedStyle(el).display
  );
  expect(display).not.toBe('none');
});

// ── node radius range ─────────────────────────────────────────────────────────

test('[PB] all nodes have radius >= 5', async ({ page }) => {
  await loadGraph(page);
  const radii = await page.locator('svg g.node circle').evaluateAll(
    (els: SVGCircleElement[]) => els.map(el => Number(el.getAttribute('r')))
  );
  expect(radii.length).toBeGreaterThan(0);
  expect(Math.min(...radii)).toBeGreaterThanOrEqual(5);
});

// ── zoom-to-search ────────────────────────────────────────────────────────────

// Helper: get SVG <g> transform string (the D3 zoom target)
async function getSvgTransform(page: any): Promise<string> {
  return page.locator('svg > g').first().getAttribute('transform').then((t: string | null) => t ?? '');
}

// Helper: get bounding rect of matched nodes in viewport coordinates
async function getMatchedNodeViewportBounds(page: any) {
  return page.locator('svg g.node:not(.search-dim) circle').evaluateAll((els: SVGCircleElement[]) => {
    return els.map(el => {
      const rect = el.getBoundingClientRect();
      return { left: rect.left, right: rect.right, top: rect.top, bottom: rect.bottom };
    });
  });
}

test('[PC1a] matched nodes are horizontally within viewport after search', async ({ page }) => {
  await loadGraph(page);
  await page.locator('#search-input').fill('Alpha');
  await page.waitForTimeout(600); // debounce + zoom animation
  const W = await page.evaluate(() => window.innerWidth);
  const pad = 40;
  const rects = await getMatchedNodeViewportBounds(page);
  expect(rects.length).toBeGreaterThan(0);
  for (const r of rects) {
    expect(r.left).toBeGreaterThanOrEqual(pad);
    expect(r.right).toBeLessThanOrEqual(W - pad);
  }
});

test('[PC1b] matched nodes are vertically within viewport after search', async ({ page }) => {
  await loadGraph(page);
  await page.locator('#search-input').fill('Alpha');
  await page.waitForTimeout(600);
  const H = await page.evaluate(() => window.innerHeight);
  const pad = 40;
  const rects = await getMatchedNodeViewportBounds(page);
  expect(rects.length).toBeGreaterThan(0);
  for (const r of rects) {
    expect(r.top).toBeGreaterThanOrEqual(pad);
    expect(r.bottom).toBeLessThanOrEqual(H - pad);
  }
});

test('[PC2] zoom transform changes when search produces matches', async ({ page }) => {
  await loadGraph(page);
  // Capture transform before search (zoom-to-fit initial state)
  await page.waitForTimeout(600); // let simulation settle
  const t0 = await getSvgTransform(page);
  // Search for a term that matches one note
  await page.locator('#search-input').fill('Alpha');
  await page.waitForTimeout(600);
  const t1 = await getSvgTransform(page);
  // Transform must have changed — zoom-to-fit on search result differs from initial
  expect(t1).not.toBe(t0);
});

// ── click-dim: dim non-neighbors on node click ────────────────────────────────

test('[PD1a] clicking a node dims nodes outside its 2-hop neighborhood', async ({ page }) => {
  await loadGraph(page);
  // fixture: alpha-0001 and beta-0002 have no edges — each is outside the other's neighborhood
  await page.locator('svg g circle').first().click();
  // The second node (not clicked) should have search-dim
  const secondHasDim = await page.locator('svg g circle').nth(1).evaluate(
    el => el.parentElement?.classList.contains('search-dim')
  );
  expect(secondHasDim).toBe(true);
});

test('[PD1b] clicking a node does not dim itself', async ({ page }) => {
  await loadGraph(page);
  await page.locator('svg g circle').first().click();
  const firstHasDim = await page.locator('svg g circle').first().evaluate(
    el => el.parentElement?.classList.contains('search-dim')
  );
  expect(firstHasDim).toBe(false);
});

test('[PD2] clicking SVG background clears click-dim', async ({ page }) => {
  await loadGraph(page);
  await page.locator('svg g circle').first().click();
  // Confirm dim was applied
  const dimmedBefore = await page.locator('g.node.search-dim').count();
  expect(dimmedBefore).toBeGreaterThan(0);
  // Click background (SVG element itself)
  await page.locator('svg').click({ position: { x: 10, y: 10 } });
  const dimmedAfter = await page.locator('g.node.search-dim').count();
  expect(dimmedAfter).toBe(0);
});
