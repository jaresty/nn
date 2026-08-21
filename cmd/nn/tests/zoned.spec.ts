import { test, expect, Page } from '@playwright/test';
import { startServer, stopServer, BASE_URL } from './fixture';

test.beforeAll(async () => { await startServer(); });
test.afterAll(() => stopServer());

type Placed = { id: string; zone: string; x: number; y: number };

// Run the real applyZonedLayout over a synthetic dataset via the test hook.
async function layout(page: Page, nodes: Array<{ id: string; zone: string }>, egoID?: string): Promise<Placed[]> {
  await page.goto(BASE_URL);
  await page.waitForFunction(() => typeof (window as any).__testApplyZonedLayout === 'function', { timeout: 8000 });
  return page.evaluate(([ns, e]) => (window as any).__testApplyZonedLayout(ns, e), [nodes, egoID] as const);
}

const angle = (p: Placed) => Math.atan2(p.y, p.x);
const quadrantOf = (p: Placed): string => {
  // Which cardinal wedge does this position fall in? (dominant axis)
  if (Math.abs(p.x) < 1e-9 && Math.abs(p.y) < 1e-9) return 'none';
  if (Math.abs(p.y) >= Math.abs(p.x)) return p.y < 0 ? 'top' : 'bottom';
  return p.x < 0 ? 'left' : 'right';
};

// property 2: same-zone nodes stay in their zone's quadrant even when the bucket
// is large. Straight-line placement lets outer nodes cross into the perpendicular
// quadrant, so this fails until arc-fan constrains them to a wedge.
test('[Z2] all top-zone nodes remain in the top quadrant with many same-zone nodes', async ({ page }) => {
  const placed = await layout(page, [
    { id: 'e', zone: '' },
    { id: 't1', zone: 'top' }, { id: 't2', zone: 'top' }, { id: 't3', zone: 'top' },
    { id: 't4', zone: 'top' }, { id: 't5', zone: 'top' },
  ]);
  const tops = placed.filter(p => p.zone === 'top');
  expect(tops.length).toBe(5);
  for (const p of tops) {
    expect(quadrantOf(p)).toBe('top');
  }
});

// property 1: distinct edge angle per same-zone node.
test('[Z1] two same-zone nodes leave the ego at distinct angles', async ({ page }) => {
  const placed = await layout(page, [
    { id: 'e', zone: '' },
    { id: 't1', zone: 'top' }, { id: 't2', zone: 'top' },
  ]);
  const tops = placed.filter(p => p.zone === 'top');
  expect(tops.length).toBe(2);
  expect(Math.abs(angle(tops[0]) - angle(tops[1]))).toBeGreaterThan(1e-3);
});

// property 3: a lone zone node sits on the exact cardinal axis (top = -pi/2).
test('[Z3] a lone top node sits on the top cardinal axis', async ({ page }) => {
  const placed = await layout(page, [
    { id: 'e', zone: '' },
    { id: 't1', zone: 'top' },
  ]);
  const t = placed.find(p => p.zone === 'top')!;
  expect(Math.abs(t.x)).toBeLessThan(1e-6); // on the vertical axis
  expect(t.y).toBeLessThan(0);              // above the ego
});

// property 4: none-bucket nodes (incl. ego) stay at the origin.
test('[Z4] none-zone node stays at the origin', async ({ page }) => {
  const placed = await layout(page, [
    { id: 'e', zone: '' },
    { id: 't1', zone: 'top' },
  ]);
  const e = placed.find(p => p.id === 'e')!;
  expect(Math.abs(e.x)).toBeLessThan(1e-6);
  expect(Math.abs(e.y)).toBeLessThan(1e-6);
});

// ── micro-legend content: each axis names its relationships ──────────────────
// (Replaces the old corner-box order check.) Each per-axis micro-legend lists
// the relationship types that place a node in that zone.
test('[Z5] each axis micro-legend names its zone relationships', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g circle').first().click();
  await page.waitForTimeout(1300);
  await expect(page.locator('#mlegend-top')).toContainText('grounded-by');
  await expect(page.locator('#mlegend-bottom')).toContainText('governs');
  await expect(page.locator('#mlegend-left')).toContainText('contradicts');
  await expect(page.locator('#mlegend-right')).toContainText('source-of');
});

// ── wedge half-angle <= 30deg (property 2) ───────────────────────────────────
// property 2: with many same-zone nodes, no node deviates > 30deg from its
// cardinal axis, so a wider gutter sits between adjacent zones.
test('[Z6] no same-zone node exceeds 30deg from its cardinal axis', async ({ page }) => {
  const placed = await layout(page, [
    { id: 'e', zone: '' },
    ...Array.from({ length: 8 }, (_, i) => ({ id: 'r' + i, zone: 'right' })),
  ]);
  const cardinal: Record<string, number> = { top: -Math.PI / 2, bottom: Math.PI / 2, left: Math.PI, right: 0 };
  const rights = placed.filter(p => p.zone === 'right');
  expect(rights.length).toBe(8);
  const maxDev = Math.max(...rights.map(p => Math.abs(angle(p) - cardinal.right)));
  expect(maxDev).toBeLessThanOrEqual(Math.PI / 6 + 1e-6); // 30deg
});

// ── adjacent same-zone label radii differ (property 3) ───────────────────────
// property 3: consecutive same-zone nodes sit at different radii so their
// below-node labels do not share a baseline.
test('[Z7] adjacent same-zone nodes have different radii', async ({ page }) => {
  const placed = await layout(page, [
    { id: 'e', zone: '' },
    { id: 'r0', zone: 'right' }, { id: 'r1', zone: 'right' },
    { id: 'r2', zone: 'right' }, { id: 'r3', zone: 'right' },
  ]);
  const rights = placed.filter(p => p.zone === 'right');
  const radius = (p: Placed) => Math.hypot(p.x, p.y);
  for (let i = 0; i + 1 < rights.length; i++) {
    expect(Math.abs(radius(rights[i]) - radius(rights[i + 1]))).toBeGreaterThan(1e-3);
  }
});

// ── ego alone at origin; other none-nodes spread (property 4a/4b) ─────────────
// property 4a: only the true ego sits at (0,0). property 4b: other zoneless
// nodes are pushed off the origin so their labels stop stacking on the ego.
test('[Z8] only the ego sits at the origin; other none-zone nodes are off-origin', async ({ page }) => {
  const placed = await layout(page, [
    { id: 'ego', zone: '' },
    { id: 'n1', zone: '' }, { id: 'n2', zone: '' }, { id: 'n3', zone: '' },
    { id: 't1', zone: 'top' },
  ], 'ego');
  const atOrigin = (p: Placed) => Math.abs(p.x) < 1e-6 && Math.abs(p.y) < 1e-6;
  const ego = placed.find(p => p.id === 'ego')!;
  expect(atOrigin(ego)).toBe(true);
  for (const id of ['n1', 'n2', 'n3']) {
    const n = placed.find(p => p.id === id)!;
    expect(atOrigin(n)).toBe(false);
  }
});

// ── R1: zoned recenter uses depth=1 (no zoneless none bucket) ─────────────────
// property R1: recentering in zoned mode fetches depth=1, so every rendered
// neighbor has a direct edge to the ego and therefore a zone — no 'none' pile.
test('[Z9] zoned recenter fetches depth=1', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  const btn = page.locator('#btn-zoned');
  if (!(await btn.evaluate((el: HTMLElement) => el.classList.contains('active')))) await btn.click();
  const urls: string[] = [];
  page.on('request', r => { if (r.url().includes('/graph?focus=')) urls.push(r.url()); });
  await page.locator('svg g circle').first().click();
  await page.waitForTimeout(800);
  expect(urls.length).toBeGreaterThan(0);
  expect(urls.some(u => u.includes('depth=1'))).toBe(true);
  expect(urls.some(u => u.includes('depth=2'))).toBe(false);
});

// ── R3: zoned mode keeps labels visible below the zoom threshold ──────────────
// property R3: the zone layout is meaningless without labels, so in zoned mode
// labels stay displayed even when zoomed out past LABEL_ZOOM_THRESHOLD.
test('[Z10] focused zoned view keeps node labels visible when zoomed out', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g.node text', { timeout: 8000 });
  await page.locator('#btn-zoned').click(); // enter zoned mode
  await page.locator('svg g circle').first().click(); // focus a node (armed -> focused)
  await page.waitForTimeout(1000);
  await page.evaluate(() => (window as any).__testSetZoom(0.1)); // well below threshold
  await page.waitForTimeout(200);
  const display = await page.locator('svg g.node text').first().evaluate(
    el => window.getComputedStyle(el).display
  );
  expect(display).not.toBe('none');
});

// ── R2: zoomFit is panel-aware (centers in the panel-clear region) ────────────
// property R2b: when the detail panel is open, the fitted content is centered in
// the panel-clear region so the ego is not pushed under the panel. We assert the
// graph's rendered center sits left of the panel's left edge.
test('[Z11] with panel open, fitted graph center is left of the panel', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g.node circle', { timeout: 8000 });
  const btn = page.locator('#btn-zoned');
  if (!(await btn.evaluate((el: HTMLElement) => el.classList.contains('active')))) await btn.click();
  await page.locator('svg g circle').first().click(); // recenter + open panel
  await page.waitForTimeout(1000);
  const panelLeft = await page.locator('#right-col').evaluate((el: HTMLElement) => el.getBoundingClientRect().left);
  // centroid of visible node circles in viewport x
  const cx = await page.locator('svg g.node circle').evaluateAll((els: SVGCircleElement[]) => {
    const vis = els.filter(el => window.getComputedStyle(el.parentElement as Element).display !== 'none');
    const xs = vis.map(el => el.getBoundingClientRect().left + el.getBoundingClientRect().width / 2);
    return xs.reduce((a, b) => a + b, 0) / xs.length;
  });
  expect(cx).toBeLessThan(panelLeft);
});

// (Superseded) The reset affordance is now the #focus-crumb breadcrumb, not a
// back button — its appearance-on-drill-in property is covered by [Z15].

// ── B2: settled recenter fit is legible (not the mid-glide tiny scale) ────────
// property B2: after the recenter glide settles, zoomFit has fitted the final
// zoned positions, so the zoom scale is legible (k >= 0.5), not the pre-glide
// k~0.18 produced by fitting mid-transition bounds.
test('[Z13] zoned recenter settles to a legible zoom scale', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  const btn = page.locator('#btn-zoned');
  if (!(await btn.evaluate((el: HTMLElement) => el.classList.contains('active')))) await btn.click();
  await page.locator('svg g circle').first().click();
  await page.waitForTimeout(1200); // allow glide + delayed fit to settle
  const k = await page.evaluate(() => {
    const svg = document.querySelector('svg')!;
    const t = (window as any).d3.zoomTransform(svg);
    return t.k;
  });
  expect(k).toBeGreaterThanOrEqual(0.5);
});

// ── B3a/B3b: labels are truncated; the full title is available via the datum ──
// property B3a: rendered label text is capped so long titles do not overrun.
// property B3b: the full untruncated title is available on the node datum (which
// the #tooltip shows on hover) — no native <title> is used (it would compete).
test('[Z14] labels are truncated; the full title lives on the node datum', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g.node text', { timeout: 8000 });
  const info = await page.locator('svg g.node').evaluateAll((gs: SVGGElement[]) => {
    return gs.map(g => {
      const label = g.querySelector('text');
      const full = (g as any).__data__ ? (g as any).__data__.title : null;
      return { text: label ? label.textContent || '' : '', full };
    });
  });
  expect(info.length).toBeGreaterThan(0);
  // B3a: no rendered label exceeds the cap (allow ellipsis char)
  const MAXLEN = 28; // code truncates to 26 chars + ellipsis
  for (const n of info) expect(n.text.length).toBeLessThanOrEqual(MAXLEN);
  // B3b: the full title is available on the datum (shown by #tooltip on hover)
  for (const n of info) expect(n.full).toBeTruthy();
  // At least one full title is longer than its truncated label (truncation happened)
  expect(info.some(n => (n.full || '').length > n.text.length)).toBe(true);
});

// ── P1: drill-in shows a titled breadcrumb chip ──────────────────────────────
// property P1: after recentering in zoned mode, a #focus-crumb appears whose
// text names the focused note — a navigation indicator, not a bare toggle.
test('[Z15] zoned drill-in shows a breadcrumb naming the focused note', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  const btn = page.locator('#btn-zoned');
  if (!(await btn.evaluate((el: HTMLElement) => el.classList.contains('active')))) await btn.click();
  const before = await page.locator('#focus-crumb').evaluate(el => getComputedStyle(el).display).catch(() => 'missing');
  expect(before === 'none' || before === 'missing').toBe(true);
  // click a node that has neighbors so the ego subgraph is non-trivial
  const target = page.locator('svg g.node').filter({ hasText: 'Alpha' }).locator('circle').first();
  await (await target.count() ? target : page.locator('svg g circle').first()).click();
  await page.waitForTimeout(1000);
  const crumb = page.locator('#focus-crumb');
  await expect(crumb).toBeVisible();
  const text = await crumb.textContent();
  expect(text && text.length).toBeGreaterThan(0);
  // names a real note title fragment (fixture titles contain 'Note')
  expect(text).toMatch(/Note/);
});

// ── P2a/P2b: dismissing the breadcrumb restores the full graph ───────────────
// property P2a: after dismiss, the full graph is back (node count matches root).
// property P2b: after dismiss, the breadcrumb is hidden.
test('[Z16] dismissing the breadcrumb restores the full graph and hides the chip', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  const rootCount = await page.locator('svg g.node').count();
  const btn = page.locator('#btn-zoned');
  if (!(await btn.evaluate((el: HTMLElement) => el.classList.contains('active')))) await btn.click();
  await page.locator('svg g circle').first().click();
  await page.waitForTimeout(1000);
  await expect(page.locator('#focus-crumb')).toBeVisible();
  // dismiss via the chip itself
  await page.locator('#focus-crumb').click();
  await page.waitForTimeout(1000);
  const afterCount = await page.locator('svg g.node').count();
  expect(afterCount).toBe(rootCount);
  const disp = await page.locator('#focus-crumb').evaluate(el => getComputedStyle(el).display);
  expect(disp).toBe('none');
});

// ── P3: while focused, the toggle row is disabled; enabled otherwise ─────────
// property P3: drilling in disables the Grouped/Zoned/All toggles (reinforcing
// that you are in a navigated state); dismissing re-enables them.
test('[Z17] toggle buttons are disabled while focused and re-enabled after dismiss', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  const ids = ['#btn-layout', '#btn-zoned', '#btn-status'];
  const btn = page.locator('#btn-zoned');
  if (!(await btn.evaluate((el: HTMLElement) => el.classList.contains('active')))) await btn.click();
  await page.locator('svg g circle').first().click();
  await page.waitForTimeout(1000);
  for (const id of ids) expect(await page.locator(id).isDisabled()).toBe(true);
  await page.locator('#focus-crumb').click();
  await page.waitForTimeout(1000);
  for (const id of ids) expect(await page.locator(id).isDisabled()).toBe(false);
});

// ── D1: zoned layout renders only when a node is focused ──────────────────────
// property D1: zones are defined relative to a focus, so entering zoned mode
// with nothing focused must NOT apply the zoned fixed-position layout (which
// would dump every zoneless node into one ball). The force layout stays.
test('[Z18] entering zoned mode with no focus does not apply the zoned layout', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g.node circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click(); // zoned on, but nothing focused
  await page.waitForTimeout(600);
  // The tell-tale of the zoned layout is fixed fx/fy on nodes. In the armed
  // (unfocused) state no node should be pinned by the zoned layout.
  const anyPinned = await page.locator('svg g.node').evaluateAll((gs: SVGGElement[]) =>
    gs.some(g => { const d = (g as any).__data__; return d && d.fx != null && d.fy != null; })
  );
  expect(anyPinned).toBe(false);
});

// ── D2: armed hint is shown when zoned mode is on but nothing is focused ──────
// property D2: the user must be told zoned mode is waiting for a focus.
test('[Z19] zoned mode with no focus shows a "pick a node" hint', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g.node circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.waitForTimeout(300);
  await expect(page.locator('#zoned-hint')).toBeVisible();
});

// ── D3: exiting focus returns to the armed (unfocused, force) state ───────────
// property D3: dismissing the breadcrumb clears the focus; combined with D1 the
// zoned layout is no longer applied — no broken whole-graph zoned ball.
test('[Z20] exiting focus clears the focus and does not leave a zoned whole-graph render', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g.node circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g circle').first().click(); // focus
  await page.waitForTimeout(1000);
  await page.locator('#focus-crumb').click();          // exit focus
  await page.waitForTimeout(1000);
  const anyPinned = await page.locator('svg g.node').evaluateAll((gs: SVGGElement[]) =>
    gs.some(g => { const d = (g as any).__data__; return d && d.fx != null && d.fy != null; })
  );
  expect(anyPinned).toBe(false);
  // Exiting returns to the prior view with zoned OFF (not the armed state), so
  // the armed hint is hidden and the Zoned button is inactive.
  await expect(page.locator('#btn-zoned')).not.toHaveClass(/active/);
  const hint = await page.locator('#zoned-hint').evaluate(el => getComputedStyle(el).display !== 'none');
  expect(hint).toBe(false);
});

// ── B1a/B1b: tension is a halo, not a fill override ──────────────────────────
// The observation color; must not be overridden by the warn/protocol red for a
// left-zone (contradicts) node. Left-zone nodes carry a tension-halo class.
test('[Z21] left-zone node keeps its type fill and gets a tension halo (not red fill)', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  // focus alpha (beta contradicts it -> beta is in the LEFT zone)
  const alpha = page.locator('svg g.node').filter({ hasText: 'Alpha' }).locator('circle').first();
  await alpha.click();
  await page.waitForTimeout(1200);
  // find beta's circle and its data
  const beta = await page.locator('svg g.node').filter({ hasText: 'Beta' }).first().evaluate((g: SVGGElement) => {
    const c = g.querySelector('circle')!;
    const d = (g as any).__data__;
    return { zone: d.zone, type: d.type, fill: c.getAttribute('fill'), hasHalo: c.classList.contains('tension-halo') || g.classList.contains('tension-halo') };
  });
  expect(beta.zone).toBe('left');
  // B1a: fill is the type color (observation = #f7e07e), NOT the warn/protocol red (#f77e7e)
  expect(beta.fill!.toLowerCase()).not.toBe('#f77e7e');
  // B1b: it carries the tension halo marker
  expect(beta.hasHalo).toBe(true);
});

// ── A1: after a focused zoned fit, no node is occluded by an overlay ──────────
// property A1: the fit region excludes the legends, breadcrumb, and panel, so no
// node circle overlaps any visible overlay.
test('[Z22] focused zoned fit does not place nodes under the info overlays', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g.node').filter({ hasText: 'Alpha' }).locator('circle').first().click();
  await page.waitForTimeout(1400); // recenter + settle + delayed fit
  const overlapCount = await page.evaluate(() => {
    const overlays = ['legend', 'zone-legend', 'right-col', 'focus-crumb', 'search-bar']
      .map(id => document.getElementById(id))
      .filter(e => e && getComputedStyle(e).display !== 'none')
      .map(e => e!.getBoundingClientRect());
    const circles = [...document.querySelectorAll('svg g.node circle')]
      .filter(c => getComputedStyle(c.parentElement as Element).display !== 'none')
      .map(c => c.getBoundingClientRect());
    const hit = (a: DOMRect, b: DOMRect) => a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top;
    let n = 0;
    for (const c of circles) for (const o of overlays) if (hit(c, o)) { n++; break; }
    return n;
  });
  expect(overlapCount).toBe(0);
});

// ── C1: the breadcrumb is a top-center banner, clear of the corner legend ─────
// property C1: while focused, the #focus-crumb is horizontally centered (not
// pinned to the top-left corner where it collided with the color #legend). The
// legend on the fixture is too short to overlap, so we assert the placement
// property directly: the crumb's center is near the viewport center and it does
// not intersect the legend.
test('[Z23] focus breadcrumb is a centered banner, not overlapping the legend', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g circle').first().click(); // focus -> crumb appears
  await page.waitForTimeout(1000);
  const info = await page.evaluate(() => {
    const b = (id: string) => { const e = document.getElementById(id)!; const r = e.getBoundingClientRect(); return { l: r.left, t: r.top, r: r.right, b: r.bottom, cx: (r.left + r.right) / 2 }; };
    return { crumb: b('focus-crumb'), legend: b('legend'), vw: window.innerWidth };
  });
  const { crumb, legend, vw } = info;
  // centered: crumb center within 15% of viewport center (was left:14 -> far left)
  expect(Math.abs(crumb.cx - vw / 2)).toBeLessThan(vw * 0.15);
  // and does not intersect the legend
  const intersects = crumb.l < legend.r && crumb.r > legend.l && crumb.t < legend.b && crumb.b > legend.t;
  expect(intersects).toBe(false);
});

// ── P2a/P2b: rim nodes show a +N hidden-connections badge ────────────────────
// property P2a: a badge appears iff a rim node has hidden direct connections.
// property P2b: the badge text is "+N" where N = total degree - visible edges.
test('[Z24] rim node shows a +N badge counting its hidden direct connections', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  // focus alpha: beta (deg 2) is a rim node with 1 hidden edge (gamma->beta)
  await page.locator('svg g.node').filter({ hasText: 'Alpha' }).locator('circle').first().click();
  await page.waitForTimeout(1200);
  const info = await page.locator('svg g.node').evaluateAll((gs: SVGGElement[]) =>
    gs.map(g => {
      const d = (g as any).__data__;
      const badge = g.querySelector('.hidden-badge');
      return { title: (d.title || '').slice(0, 10), badge: badge ? badge.textContent : null };
    })
  );
  const beta = info.find(n => n.title.startsWith('Beta'))!;
  const alpha = info.find(n => n.title.startsWith('Alpha'))!;
  // P2b: beta hides 1 connection -> "+1"
  expect(beta.badge).toBe('+1');
  // P2a: the focused ego (alpha) has all its connections visible -> no badge
  expect(alpha.badge).toBeNull();
});

// ── A1: entering Zoned adopts the currently selected node as the focus ────────
// property A1: if a node was clicked (selected) in force mode, toggling Zoned on
// focuses that node immediately instead of showing the armed hint.
test('[Z25] toggling Zoned on with a node selected focuses that node', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  // select a node in force mode (opens panel, sets selection) WITHOUT zoned on
  await page.locator('svg g.node').filter({ hasText: 'Alpha' }).locator('circle').first().click();
  await page.waitForTimeout(400);
  // now turn on Zoned — it should adopt the selection
  await page.locator('#btn-zoned').click();
  await page.waitForTimeout(1200);
  // breadcrumb shows the selected note; armed hint is not shown
  await expect(page.locator('#focus-crumb')).toBeVisible();
  const crumbText = await page.locator('#focus-crumb').textContent();
  expect(crumbText).toMatch(/Alpha/);
  const hintVisible = await page.locator('#zoned-hint').evaluate(el => getComputedStyle(el).display !== 'none');
  expect(hintVisible).toBe(false);
});

// ── A2: entering Zoned with nothing selected stays in the armed state ─────────
test('[Z26] toggling Zoned on with no selection shows the armed hint', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click(); // nothing selected
  await page.waitForTimeout(400);
  await expect(page.locator('#zoned-hint')).toBeVisible();
  const crumbVisible = await page.locator('#focus-crumb').evaluate(el => getComputedStyle(el).display !== 'none');
  expect(crumbVisible).toBe(false);
});

// ── E1: edges are colored by relationship family in focus mode ───────────────
// property E1: a contradicts/questions edge (tension) gets a distinct color
// from a structural (refines/etc.) edge.
test('[Z27] tension edges are colored distinctly from structural edges', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  // focus beta: beta->alpha is contradicts (tension), gamma->beta is refines (structural)
  await page.locator('svg g.node').filter({ hasText: 'Beta' }).locator('circle').first().click();
  await page.waitForTimeout(1200);
  // Visible .link lines carry the family color; the paired .link-hit lines carry
  // the edge datum (with its type). They share data order, so zip by index.
  const strokes = await page.evaluate(() => {
    const vis = [...document.querySelectorAll('svg path.link')];
    const hit = [...document.querySelectorAll('svg path.link-hit')];
    return vis.map((l, i) => ({
      type: (hit[i] as any)?.__data__?.type ?? null,
      stroke: l.getAttribute('stroke'),
    }));
  });
  const tension = strokes.find(s => s.type === 'contradicts');
  const structural = strokes.find(s => s.type === 'refines');
  expect(tension).toBeTruthy();
  expect(structural).toBeTruthy();
  // tension and structural strokes must differ (colored by family)
  expect(tension!.stroke).not.toBe(structural!.stroke);
});

// ── H1: each edge has a WIDE (hoverable) hit-line ────────────────────────────
// property H1: a 1px visible line is unhittable, so each edge has a transparent
// hit-line with a wide stroke (>= 8px). The type is shown on hover (see [Z31]).
test('[Z28] each edge has a wide hit-line for hovering', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g.node').filter({ hasText: 'Beta' }).locator('circle').first().click();
  await page.waitForTimeout(1200);
  const widths = await page.locator('svg path.link-hit').evaluateAll((ls: SVGLineElement[]) =>
    ls.map(l => parseFloat(getComputedStyle(l).strokeWidth))
  );
  expect(widths.length).toBeGreaterThan(0);
  for (const w of widths) expect(w).toBeGreaterThanOrEqual(8);
});

// ── PN1: the detail panel stays open after a zoned recenter ──────────────────
// property PN1: recentering opens the panel for the focused note and it stays
// open (previously openPanel ran before the async rebuild's closePanel, so the
// panel flashed then closed).
test('[Z29] detail panel stays open after a zoned recenter', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g.node').filter({ hasText: 'Beta' }).locator('circle').first().click();
  await page.waitForTimeout(1400); // recenter + rebuild + settle
  await expect(page.locator('#right-col')).toHaveClass(/open/);
  await expect(page.locator('#panel-title')).not.toBeEmpty();
});

// ── C1: the focused ego stays centered even when zones are empty ──────────────
// property C1: focusing alpha (only a LEFT-zone neighbor) leaves top/right/bottom
// empty; the ego must still sit at the center of the clear region, not be pushed
// toward the populated side by a bbox-centered fit.
test('[Z30] focused ego stays at the clear-region center with empty axes', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g.node').filter({ hasText: 'Alpha' }).locator('circle').first().click();
  await page.waitForTimeout(1500); // recenter + settle + delayed fit
  const info = await page.evaluate(() => {
    // ego is the focused node; find its screen center
    const gs = [...document.querySelectorAll('svg g.node')];
    const egoG = gs.find(g => { const d = (g as any).__data__; return d && Math.abs(d.fx) < 1 && Math.abs(d.fy) < 1; });
    const egoRect = egoG!.querySelector('circle')!.getBoundingClientRect();
    const egoCx = egoRect.left + egoRect.width / 2;
    const egoCy = egoRect.top + egoRect.height / 2;
    // clear region = svg minus visible overlays (approx: subtract panel on right, legends)
    const svg = document.querySelector('svg')!.getBoundingClientRect();
    const panel = document.getElementById('right-col')!;
    const panelOpen = panel.classList.contains('open');
    const panelW = panelOpen ? panel.getBoundingClientRect().width : 0;
    const zl = document.getElementById('mlegend-bottom')!.getBoundingClientRect();
    const clearCx = (0 + (svg.width - panelW)) / 2;
    const clearCy = (0 + Math.min(svg.height, zl.top)) / 2; // above the bottom micro-legend
    return { egoCx, egoCy, clearCx, clearCy, svgW: svg.width };
  });
  // ego within 12% of viewport width from the clear-region center
  expect(Math.abs(info.egoCx - info.clearCx)).toBeLessThan(info.svgW * 0.12);
  expect(Math.abs(info.egoCy - info.clearCy)).toBeLessThan(info.svgW * 0.12);
});

// ── HV1/HV2: hovering an edge shows its link type in the #tooltip ─────────────
// property HV1: mouseover a hit-line makes #tooltip visible with the link type.
// property HV2: mouseleave hides it. Replaces the flaky native <title>.
test('[Z31] hovering an edge shows its link type in the tooltip', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g.node').filter({ hasText: 'Beta' }).locator('circle').first().click();
  await page.waitForTimeout(1400);
  // Hover at the edge's MIDPOINT via elementFromPoint — this respects z-order, so
  // it fails if the visible thin line (or a node) sits on top of the hit-line.
  const shown = await page.evaluate(() => {
    const hit = [...document.querySelectorAll('svg path.link-hit')]
      .find(l => (l as any).__data__ && (l as any).__data__.type === 'contradicts') as SVGPathElement | undefined;
    if (!hit) return { found: false };
    // A curved path's bbox center is off the path, so take a point ON the path
    // (via getPointAtLength) and convert to screen coords through the CTM.
    const pt = hit.getPointAtLength(hit.getTotalLength() / 2);
    const svg = hit.ownerSVGElement!;
    const dompt = svg.createSVGPoint(); dompt.x = pt.x; dompt.y = pt.y;
    const scr = dompt.matrixTransform(hit.getScreenCTM()!);
    const midX = scr.x, midY = scr.y;
    const target = document.elementFromPoint(midX, midY);
    if (!target) return { found: true, onTop: 'none' };
    target.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, clientX: midX, clientY: midY }));
    target.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, clientX: midX, clientY: midY }));
    const tip = document.getElementById('tooltip')!;
    const visible = getComputedStyle(tip).display !== 'none';
    const text = tip.textContent || '';
    const hitType = (target as any).__data__ ? (target as any).__data__.type : null;
    target.dispatchEvent(new MouseEvent('mouseleave', { bubbles: true }));
    const hiddenAfter = getComputedStyle(tip).display === 'none';
    return { found: true, onTop: (target as Element).getAttribute('class'), visible, text, hitType, hiddenAfter };
  });
  expect(shown.found).toBe(true);
  // the element actually under the cursor at the edge must be the hoverable hit-line
  expect(shown.onTop).toContain('link-hit');
  expect(shown.visible).toBe(true);       // HV1a
  // HV1b: the tooltip shows the type of whichever edge was actually hovered
  expect(shown.hitType).toBeTruthy();
  expect(shown.text).toContain(shown.hitType!);
  expect(shown.hiddenAfter).toBe(true);   // HV2
});

// ── Z1/Z2: zoneless neighbors sit on an outer diagonal ring, off the cardinals ─
// property Z1: a non-ego zoneless node's angle avoids the four cardinal axes
// (0/90/180/270) so it can't collide with a left/right/top/bottom zone node.
// property Z2: it sits beyond the zone SPREAD (outer 'other' ring).
test('[Z32] zoneless neighbors avoid the cardinal axes and sit on an outer ring', async ({ page }) => {
  const placed = await layout(page, [
    { id: 'ego', zone: '' },
    { id: 'z1', zone: '' }, { id: 'z2', zone: '' }, { id: 'z3', zone: '' },
    { id: 'r1', zone: 'right' },
  ], 'ego');
  const zoneless = placed.filter(p => (p.id === 'z1' || p.id === 'z2' || p.id === 'z3'));
  expect(zoneless.length).toBe(3);
  const SPREAD = 240;
  const cardinals = [0, Math.PI / 2, Math.PI, -Math.PI / 2];
  // robust signed-angle distance, always in [0, PI]
  const angDist = (a: number, b: number) => Math.abs(Math.atan2(Math.sin(a - b), Math.cos(a - b)));
  for (const p of zoneless) {
    const ang = Math.atan2(p.y, p.x);
    const nearest = Math.min(...cardinals.map(c => angDist(ang, c)));
    expect(nearest).toBeGreaterThanOrEqual(Math.PI / 9);      // Z1: >= 20deg off any cardinal
    expect(Math.hypot(p.x, p.y)).toBeGreaterThan(SPREAD);     // Z2: outer ring
  }
});

// ── EH1/EH2: hovering an edge highlights (thickens) its visible line ──────────
test('[Z33] hovering an edge thickens it and returns to idle on leave', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g.node').filter({ hasText: 'Beta' }).locator('circle').first().click();
  await page.waitForTimeout(1400);
  const res = await page.evaluate(() => {
    const hit = [...document.querySelectorAll('svg path.link-hit')]
      .find(h => (h as any).__data__ && (h as any).__data__.type === 'contradicts');
    if (!hit) return { found: false };
    hit.dispatchEvent(new MouseEvent('mousemove', { bubbles: true }));
    // measure the visible line that actually got the hover class
    const hovered = document.querySelector('svg path.link.edge-hover') as SVGLineElement | null;
    const wHover = hovered ? parseFloat(getComputedStyle(hovered).strokeWidth) : 0;
    hit.dispatchEvent(new MouseEvent('mouseleave', { bubbles: true }));
    const stillHovered = document.querySelectorAll('svg path.link.edge-hover').length;
    const wIdle = hovered ? parseFloat(getComputedStyle(hovered).strokeWidth) : 0;
    return { found: true, wHover, wIdle, stillHovered };
  });
  expect(res.found).toBe(true);
  expect(res.wHover).toBeGreaterThanOrEqual(2);   // EH1: a line is thickened on hover
  expect(res.stillHovered).toBe(0);                // EH2: cleared on leave
  expect(res.wIdle).toBeLessThanOrEqual(1.5);      // EH2: returns to idle
});

// ── NP1: hovering a node in focus mode previews a body snippet ───────────────
test('[Z34] focused node hover shows a body description snippet', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g.node').filter({ hasText: 'Alpha' }).locator('circle').first().click();
  await page.waitForTimeout(1400);
  const text = await page.evaluate(() => {
    // hover a neighbor node (Beta has body "Body of beta note.")
    const g = [...document.querySelectorAll('svg g.node')].find(el => {
      const d = (el as any).__data__; return d && (d.title || '').startsWith('Beta');
    });
    if (!g) return null;
    g.dispatchEvent(new MouseEvent('mousemove', { bubbles: true }));
    return document.getElementById('tooltip')!.textContent || '';
  });
  expect(text).not.toBeNull();
  // the body of the beta note is previewed
  expect(text).toContain('Body of beta');
});

// ── M1..M4: four per-axis micro-legends replace the corner box ────────────────
test('[Z35] focus mode shows four per-axis micro-legends including related, no corner box', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g circle').first().click(); // focus
  await page.waitForTimeout(1300);
  // M1: four micro-legends visible
  for (const axis of ['top', 'left', 'right', 'bottom']) {
    await expect(page.locator(`#mlegend-${axis}`)).toBeVisible();
  }
  // M2: RIGHT lists 'related'
  await expect(page.locator('#mlegend-right')).toContainText('related');
  // M3: old corner box gone (not present, or not visible)
  const cornerVisible = await page.locator('#zone-legend').count()
    ? await page.locator('#zone-legend').evaluate(el => getComputedStyle(el).display !== 'none').catch(() => false)
    : false;
  expect(cornerVisible).toBe(false);
  // M4: breadcrumb carries the recenter hint
  await expect(page.locator('#focus-crumb')).toContainText('recenter');
});

// ── B1/B2: bidirectional edges bow apart; single edges stay straight ─────────
test('[Z36] bidirectional edges curve to opposite sides; single edges are straight', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  // focus alpha: alpha<->beta is bidirectional (supports + contradicts); gamma->beta is single
  await page.locator('svg g.node').filter({ hasText: 'Alpha' }).locator('circle').first().click();
  await page.waitForTimeout(1400);
  const res = await page.evaluate(() => {
    const paths = [...document.querySelectorAll('svg path.link')] as SVGPathElement[];
    const pathOf = (type: string) => paths.find(p => (p as any).__data__.type === type);
    const ab = pathOf('supports');      // alpha->beta
    const ba = pathOf('contradicts');   // beta->alpha
    const single = pathOf('refines');   // gamma->beta (single)
    if (!ab || !ba) return { ok: false };
    // ONE shared world chord: use alpha->beta's endpoints for both.
    const d = (ab as any).__data__;
    const sx = d.source.x, sy = d.source.y, tx = d.target.x, ty = d.target.y;
    const dx = tx - sx, dy = ty - sy, len = Math.hypot(dx, dy) || 1;
    const signedAt = (p: SVGPathElement, f: number) => {
      const pt = p.getPointAtLength(p.getTotalLength() * f);
      return ((pt.x - sx) * (-dy) + (pt.y - sy) * dx) / len;
    };
    // Sample interior points. For ba (reverse direction), mirror f.
    const abS = [0.25, 0.5, 0.75].map(f => signedAt(ab, f));
    const baS = [0.75, 0.5, 0.25].map(f => signedAt(ba, f)); // mirrored to align chord position
    // single edge midpoint offset from its own chord
    let singleOff = 0;
    if (single) {
      const sd = (single as any).__data__;
      const s2x = sd.source.x, s2y = sd.source.y, t2x = sd.target.x, t2y = sd.target.y;
      const d2x = t2x - s2x, d2y = t2y - s2y, l2 = Math.hypot(d2x, d2y) || 1;
      const m = single.getPointAtLength(single.getTotalLength() / 2);
      singleOff = ((m.x - s2x) * (-d2y) + (m.y - s2y) * d2x) / l2;
    }
    return { ok: true, abS, baS, singleOff, hasSingle: !!single };
  });
  expect(res.ok).toBe(true);
  // NX1: at every interior sample, the two arcs are on OPPOSITE sides of the
  // shared chord (product < 0) and non-trivial — so they never cross.
  for (let i = 0; i < res.abS!.length; i++) {
    expect(res.abS![i] * res.baS![i]).toBeLessThan(0);
    expect(Math.abs(res.abS![i])).toBeGreaterThan(2);
    expect(Math.abs(res.baS![i])).toBeGreaterThan(2);
  }
  // NX2: single edge stays straight
  if (res.hasSingle) expect(Math.abs(res.singleOff!)).toBeLessThan(2);
});

// ── UA1: edge hover shows the annotation (description) as well as the type ────
test('[Z37] hovering an edge shows its annotation alongside the type', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g.node').filter({ hasText: 'Alpha' }).locator('circle').first().click();
  await page.waitForTimeout(1400);
  const res = await page.evaluate(() => {
    const hit = [...document.querySelectorAll('svg path.link-hit')]
      .find(l => (l as any).__data__ && (l as any).__data__.annotation) as SVGPathElement | undefined;
    if (!hit) return { found: false };
    const d = (hit as any).__data__;
    hit.dispatchEvent(new MouseEvent('mousemove', { bubbles: true }));
    const text = document.getElementById('tooltip')!.textContent || '';
    return { found: true, annotation: d.annotation, type: d.type, text };
  });
  expect(res.found).toBe(true);
  expect(res.text).toContain(res.annotation!);   // UA1a
  expect(res.text).toContain(res.type!);          // UA1b
});

// ── the top micro-legend must not overlap the breadcrumb (both top-center) ────
test('[Z38] top micro-legend does not overlap the focus breadcrumb', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g circle').first().click();
  await page.waitForTimeout(1300);
  const b = await page.evaluate(() => {
    const r = (id: string) => { const e = document.getElementById(id)!; const x = e.getBoundingClientRect(); return { l: x.left, t: x.top, r: x.right, b: x.bottom }; };
    return { ml: r('mlegend-top'), crumb: r('focus-crumb') };
  });
  const overlaps = b.ml.l < b.crumb.r && b.ml.r > b.crumb.l && b.ml.t < b.crumb.b && b.ml.b > b.crumb.t;
  expect(overlaps).toBe(false);
});

// ── T1: nodes have no native <title> (the custom #tooltip is the sole tooltip) ─
// A native <title> competes with the JS #tooltip, popping up simultaneously.
test('[Z39] graph nodes have no native <title> element', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g.node', { timeout: 8000 });
  const titleCount = await page.locator('svg g.node title').count();
  expect(titleCount).toBe(0);
});

// ── S1: searching while focused exits to the full graph (search spans the notebook) ─
test('[Z40] searching while focused restores the full graph', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  const rootCount = await page.locator('svg g.node').count();
  await page.locator('#btn-zoned').click();
  await page.locator('svg g circle').first().click(); // focus -> subgraph
  await page.waitForTimeout(1200);
  const focusedCount = await page.locator('svg g.node').count();
  expect(focusedCount).toBeLessThan(rootCount); // confirm we narrowed
  // now search — should exit focus back to the full graph
  await page.locator('#search-input').fill('note');
  await page.waitForTimeout(700); // debounce + rebuild
  const afterSearch = await page.locator('svg g.node').count();
  expect(afterSearch).toBe(rootCount);
});

// ── D1/D2: in a dense zone, always-on labels are hidden (revealed on hover) ───
// The fixture is too small to make a dense zone, so we drive applyZonedLayout via
// a synthetic dataset through a test hook that returns each node's label-hidden
// decision (zone bucket > threshold).
test('[Z41] dense-zone nodes hide their always-on label; sparse-zone nodes keep it', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForFunction(() => typeof (window as any).__testLabelHidden === 'function', { timeout: 8000 });
  const res = await page.evaluate(() => {
    // 6 right-zone nodes (dense), 1 top-zone node (sparse), plus ego
    const nodes = [
      { id: 'ego', zone: '' },
      { id: 't1', zone: 'top' },
      ...Array.from({ length: 6 }, (_, i) => ({ id: 'r' + i, zone: 'right' })),
    ];
    return (window as any).__testLabelHidden(nodes, 'ego');
  });
  // dense right-zone nodes are hidden
  for (let i = 0; i < 6; i++) expect(res['r' + i]).toBe(true);
  // sparse top-zone node is visible
  expect(res['t1']).toBe(false);
});

// ── G1: clicking Grouped while zoned-armed hides the armed hint ───────────────
test('[Z42] switching to Grouped while zoned-armed hides the armed hint', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();            // zoned on -> armed hint shows
  await expect(page.locator('#zoned-hint')).toBeVisible();
  await page.locator('#btn-layout').click();           // Grouped -> turns zoned off
  await page.waitForTimeout(300);
  const hintShown = await page.locator('#zoned-hint').evaluate(el => getComputedStyle(el).display !== 'none');
  expect(hintShown).toBe(false);
});

// ── X1: exiting a focused zoned view returns to the prior layout (Grouped) ────
test('[Z43] Grouped -> select -> Zoned -> exit returns to Grouped, zoned off', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-layout').click();               // Grouped on
  await expect(page.locator('#btn-layout')).toHaveClass(/active/);
  await page.locator('svg g circle').first().click();      // select a node in Grouped
  await page.waitForTimeout(400);
  await page.locator('#btn-zoned').click();                // Zoned adopts selection -> focus
  await page.waitForTimeout(1200);
  await expect(page.locator('#focus-crumb')).toBeVisible();
  await page.locator('#focus-crumb').click();              // exit focus
  await page.waitForTimeout(900);
  // back in Grouped, zoned off, hint hidden
  await expect(page.locator('#btn-layout')).toHaveClass(/active/);   // X1a
  await expect(page.locator('#btn-zoned')).not.toHaveClass(/active/); // X1b
  const hint = await page.locator('#zoned-hint').evaluate(el => getComputedStyle(el).display !== 'none');
  expect(hint).toBe(false);
});
