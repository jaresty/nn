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

// ── legend order (property 1) ────────────────────────────────────────────────
// property 1: zone legend rows read Top, Left, Right, Bottom.
test('[Z5] zone legend rows are ordered Top, Left, Right, Bottom', async ({ page }) => {
  await page.goto(BASE_URL);
  const dirs = await page.locator('#zone-legend .zl-row .zl-dir').allTextContents();
  const words = dirs.map(d => d.replace(/[^A-Za-z]/g, ''));
  expect(words).toEqual(['Top', 'Left', 'Right', 'Bottom']);
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

// ── B3a/B3b: labels are truncated and carry the full title as a tooltip ───────
// property B3a: rendered label text is capped so long titles do not overrun.
// property B3b: each node exposes its full title via an SVG <title> element.
test('[Z14] labels are truncated with the full title in an SVG <title>', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g.node text', { timeout: 8000 });
  const info = await page.locator('svg g.node').evaluateAll((gs: SVGGElement[]) => {
    return gs.map(g => {
      const label = g.querySelector('text');
      const title = g.querySelector('title');
      return { text: label ? label.textContent || '' : '', title: title ? title.textContent || '' : null };
    });
  });
  expect(info.length).toBeGreaterThan(0);
  // B3a: no rendered label exceeds the cap (allow ellipsis char)
  const MAXLEN = 28; // code truncates to 26 chars + ellipsis
  for (const n of info) expect(n.text.length).toBeLessThanOrEqual(MAXLEN);
  // B3b: every node has a <title> element
  for (const n of info) expect(n.title).not.toBeNull();
  // At least one title is longer than its truncated label (proves truncation happened on a long title)
  expect(info.some(n => (n.title || '').length > n.text.length)).toBe(true);
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
  await expect(page.locator('#zoned-hint')).toBeVisible();
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
  // the type title. They share data order, so zip by index.
  const strokes = await page.evaluate(() => {
    const vis = [...document.querySelectorAll('svg line.link')];
    const hit = [...document.querySelectorAll('svg line.link-hit')];
    return vis.map((l, i) => ({
      type: hit[i]?.querySelector('title')?.textContent ?? null,
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

// ── H1: each edge has a WIDE hover target carrying its link type ──────────────
// property H1: a 1px visible line is unhittable, so the tooltip must live on a
// transparent hit-line with a wide stroke (>= 8px). We assert both the width and
// the type title on the same element.
test('[Z28] each edge has a wide hit-line carrying its link-type tooltip', async ({ page }) => {
  await page.goto(BASE_URL);
  await page.waitForSelector('svg g circle', { timeout: 8000 });
  await page.locator('#btn-zoned').click();
  await page.locator('svg g.node').filter({ hasText: 'Beta' }).locator('circle').first().click();
  await page.waitForTimeout(1200);
  // Find hit-lines: lines whose <title> names a type and whose stroke-width is wide.
  const hits = await page.locator('svg line.link-hit').evaluateAll((ls: SVGLineElement[]) =>
    ls.map(l => ({
      type: l.querySelector('title')?.textContent ?? null,
      width: parseFloat(getComputedStyle(l).strokeWidth),
    }))
  );
  expect(hits.length).toBeGreaterThan(0);
  // every hit-line is wide enough to hover
  for (const h of hits) expect(h.width).toBeGreaterThanOrEqual(8);
  // the contradicts edge's type is reachable via a hit-line
  expect(hits.map(h => h.type)).toContain('contradicts');
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
