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
