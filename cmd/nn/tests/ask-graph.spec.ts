import { test, expect, Locator, Page } from '@playwright/test';
import { execSync, spawn, ChildProcess } from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';

// This spec drives the graph feedback surface (nn ask --surface graph) end to
// end: it launches the real binary in print-url mode, opens the surface it
// hosts, and exercises the scoping bound + inline commenting + Done/Send flow.

let proc: ChildProcess | null = null;
let notebookDir = '';
let sessionDir = '';
let surfaceURL = '';
let cfgDir = '';

// The four notes: ego + two direct neighbors (in scope) + one depth-2 node
// (out of scope, must never render on the surface).
const EGO = 'ego-0001';
const NBR_OUT = 'nbrout-0002';
const NBR_IN = 'nbrin-0003';
const FAR = 'far-0004';

function writeNote(dir: string, id: string, title: string, links: Array<{ to: string; type: string }> = []) {
  const linksSection = links.length
    ? `\n## Links\n\n` + links.map(l => `- [[${l.to}]] [${l.type}] — test link`).join('\n') + '\n'
    : '';
  fs.writeFileSync(path.join(dir, `${id}.md`), `---
id: ${id}
title: ${title}
type: concept
status: draft
tags: []
created: 2026-01-01T00:00:00Z
modified: 2026-01-01T00:00:00Z
---

Body of ${title}.
${linksSection}`);
}

// Setup does a go build plus launches a session, which can exceed the default
// 15s hook budget on a cold build.
test.beforeAll(async () => {
  test.setTimeout(60000);
  notebookDir = fs.mkdtempSync(path.join(os.tmpdir(), 'nn-askg-'));
  execSync('git init && git config user.email "t@t.com" && git config user.name "T"', { cwd: notebookDir, stdio: 'ignore' });
  writeNote(notebookDir, EGO, 'Ego Note', [{ to: NBR_OUT, type: 'refines' }]);
  writeNote(notebookDir, NBR_OUT, 'Outbound Neighbor');
  writeNote(notebookDir, NBR_IN, 'Inbound Neighbor', [{ to: EGO, type: 'supports' }]);
  writeNote(notebookDir, FAR, 'Far Node', [{ to: NBR_OUT, type: 'supports' }]); // depth-2 from ego
  execSync('git add . && git commit -m init', { cwd: notebookDir, stdio: 'ignore' });

  const repoRoot = path.resolve(__dirname, '../../..');
  execSync('go build -o /tmp/nn-askg ./cmd/nn', { cwd: repoRoot, stdio: 'ignore' });

  cfgDir = fs.mkdtempSync(path.join(os.tmpdir(), 'nn-askg-cfg-'));
  fs.writeFileSync(path.join(cfgDir, 'config.toml'),
    `[notebooks]\ndefault = "test"\n\n[notebooks.test]\npath = ${JSON.stringify(notebookDir)}\nbackend = "gitlocal"\n`);

  // A shared, read-only session for the non-submitting tests. `nn ask` is
  // one-shot — submitting ends its server — so tests that Send launch their own
  // session via launchSession() rather than consuming this shared one.
  const s = await launchSession();
  proc = s.proc; surfaceURL = s.url; sessionDir = s.dir;
});

// launchSession spawns a fresh `nn ask --surface graph` and returns its URL,
// session dir, and process. Each submitting test gets its own so the shared
// server survives for the read-only tests regardless of run order.
async function launchSession(): Promise<{ url: string; dir: string; proc: ChildProcess }> {
  const p = spawn('/tmp/nn-askg', ['ask', '--surface', 'graph', '--focus', EGO, '--instructions', 'React to this neighborhood.'],
    { env: { ...process.env, NN_CONFIG_DIR: cfgDir, NN_ASK_PRINT_URL_ONLY: '1' } });
  const url = await new Promise<string>((resolve, reject) => {
    let buf = '';
    const to = setTimeout(() => reject(new Error('no ASK_SURFACE_URL within 8s: ' + buf)), 8000);
    p.stdout!.on('data', d => {
      buf += d.toString();
      const m = buf.match(/ASK_SURFACE_URL (\S+)/);
      if (m) { clearTimeout(to); resolve(m[1]); }
    });
  });
  const sid = new URL(url).searchParams.get('session')!;
  return { url, dir: path.join(cfgDir, 'feedback', sid), proc: p };
}

test.afterAll(() => {
  proc?.kill();
  if (notebookDir) fs.rmSync(notebookDir, { recursive: true, force: true });
});

async function nodeIds(page: Page): Promise<string[]> {
  return page.evaluate(async () => {
    const r = await fetch('/graph');
    const g = await r.json();
    return g.nodes.map((n: any) => n.id);
  });
}

function expectCanonicalGraphSelectionShape(selection: any) {
  for (const key of ['selected', 'annotations', 'edges', 'answer'])
    expect.soft(selection, `obsolete top-level graph-selection key "${key}" must be absent`).not.toHaveProperty(key);
  expect.soft(selection, 'obsolete top-level graph-selection key "explain_on_canvas" must be absent')
    .not.toHaveProperty('explain_on_canvas');
  expect.soft(selection, 'obsolete top-level graph-selection key "handoffs" must be absent')
    .not.toHaveProperty('handoffs');
  expect.soft(Object.keys(selection).sort(), 'graph-selection top-level keys must be exactly canonical')
    .toEqual(['groups', 'handoff', 'overall_comment']);
}

async function realPointerClick(page: Page, button: Locator) {
  const box = await button.boundingBox();
  if (!box) throw new Error('terminal action button has no pointer target');
  await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
}

test('scope is bounded to the focus ego neighborhood — the depth-2 node is never served', async ({ page }) => {
  await page.goto(surfaceURL);
  const ids = await nodeIds(page);
  expect(ids).toContain(EGO);
  expect(ids).toContain(NBR_OUT);
  expect(ids).toContain(NBR_IN);
  expect(ids).not.toContain(FAR); // load-bearing: agent scope bounds what the human sees
});

test('R2: a persistent feedback panel is docked and visible without a modal', async ({ page }) => {
  await page.goto(surfaceURL);
  await expect(page.locator('#fbpanel')).toBeVisible();
  await expect(page.locator('#fbpanel-send')).toBeVisible();
  await expect(page.locator('#fbpanel-answer')).toBeVisible();
  const send = page.getByRole('button', { name: 'Send', exact: true });
  const canvas = page.getByRole('button', { name: 'Send to Canvas', exact: true });
  const document = page.getByRole('button', { name: 'Send to Document', exact: true });
  await expect(send).toBeVisible();
  await expect(canvas).toBeVisible();
  await expect(document).toBeVisible();
  await expect(send).not.toHaveAttribute('aria-pressed');
  await expect(canvas).not.toHaveAttribute('aria-pressed');
  await expect(document).not.toHaveAttribute('aria-pressed');
  await expect(page.locator('#handoff-options')).toHaveCount(0);
});

test('R4/S: layout toggle buttons hidden but the search box stays visible', async ({ page }) => {
  await page.goto(surfaceURL);
  // S1: search remains usable to navigate the scoped graph.
  await expect(page.locator('#search-input')).toBeVisible();
  // S2: the layout toggles are hidden (locked to zoned).
  await expect(page.locator('#btn-layout')).toBeHidden();
  await expect(page.locator('#btn-zoned')).toBeHidden();
  await expect(page.locator('#btn-status')).toBeHidden();
});

test('O: no top-area overlays overlap each other (banner/nav/crumb/search/legend)', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  const overlaps = await page.evaluate(() => {
    const ids = ['feedback-nav', 'feedback-modes', 'focus-crumb', 'feedback-banner', 'search-bar', 'legend'];
    const rects: Record<string, DOMRect> = {};
    ids.forEach(id => {
      const el = document.getElementById(id);
      if (el && getComputedStyle(el).display !== 'none') rects[id] = el.getBoundingClientRect();
    });
    const inter = (a: DOMRect, b: DOMRect) => !(a.right < b.left || a.left > b.right || a.bottom < b.top || a.top > b.bottom);
    const keys = Object.keys(rects);
    const pairs: string[] = [];
    for (let i = 0; i < keys.length; i++)
      for (let j = i + 1; j < keys.length; j++)
        if (inter(rects[keys[i]], rects[keys[j]])) pairs.push(keys[i] + ' × ' + keys[j]);
    return pairs;
  });
  expect(overlaps).toEqual([]);
});

test('R5: restore-view and back controls are present', async ({ page }) => {
  await page.goto(surfaceURL);
  await expect(page.locator('#btn-restore')).toBeVisible();
  await expect(page.locator('#btn-back')).toBeVisible();
});

test('the surface opens in Zoned spatial layout focused on the ego', async ({ page }) => {
  await page.goto(surfaceURL);
  await expect(page.locator('#btn-zoned')).toHaveClass(/active/);
  await expect(page.locator('#focus-crumb')).toBeVisible();
});

test('R3+R6: clicking a node views it (no dismiss on re-click of empty canvas)', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  await expect(page.locator('#panel-comment')).toBeVisible();
  // Clicking empty canvas must NOT dismiss the card in feedback mode.
  await page.locator('svg').click({ position: { x: 5, y: 5 } });
  await expect(page.locator('#panel-comment')).toBeVisible();
});

test('R7: the panel renders markdown, not raw text', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  // Body "Body of Ego Note." is plain, but the panel container should hold
  // rendered elements (a <p>), not a raw text node with leading newlines.
  await expect(page.locator('#panel-body p').first()).toBeVisible();
});

test('R1: commenting on a note adds it to the feedback panel; clearing removes it', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  await page.locator('#panel-comment').fill('load-bearing note');
  await expect(page.locator('.fbitem')).toHaveCount(1);
  await expect(page.locator('#fbpanel-list')).toContainText('load-bearing note');
  await page.locator('#panel-comment').fill('');
  await expect(page.locator('.fbitem')).toHaveCount(0);
});

async function clickNode(page: Page, id: string, modifier = false) {
  await page.waitForSelector('g.node');
  await page.locator('g.node').evaluateAll((nodes: any[], args: { wanted: string; modifier: boolean }) => {
    const node = nodes.find(n => n.__data__?.id === args.wanted);
    if (!node) throw new Error(`node ${args.wanted} not rendered`);
    node.dispatchEvent(new MouseEvent('click', { bubbles: true, shiftKey: args.modifier }));
  }, { wanted: id, modifier });
}

async function modifierClickNode(page: Page, id: string) {
  await clickNode(page, id, true);
}

async function clickFirstEdge(page: Page, modifier = false) {
  await page.waitForSelector('path.link-hit', { state: 'attached' });
  await page.evaluate((withModifier: boolean) => {
    const edge = document.querySelector('path.link-hit') as SVGElement;
    edge.dispatchEvent(new MouseEvent('click', { bubbles: true, shiftKey: withModifier }));
  }, modifier);
}

async function modifierClickFirstEdge(page: Page) {
  await clickFirstEdge(page, true);
}

type LassoBox = { left: number; top: number; right: number; bottom: number };

async function settleInLassoMode(page: Page, url: string) {
  await page.goto(url);
  await page.waitForSelector('g.node circle');
  await page.waitForTimeout(1000); // zoned placement + zoomFit transition
  const button = page.getByRole('button', { name: 'Lasso', exact: true });
  if (await button.getAttribute('aria-pressed') !== 'true') await button.click();
  await expect(button).toHaveAttribute('aria-pressed', 'true');
}

// Drive the browser's actual pointer pipeline. The gesture must start on the SVG
// background because production deliberately reserves node/edge pointer targets.
async function pointerLasso(page: Page, box: LassoBox) {
  const drag = await page.evaluate((candidate: LassoBox) => {
    const svg = document.querySelector('svg')!;
    const corners = [
      { x: candidate.left, y: candidate.top, endX: candidate.right, endY: candidate.bottom },
      { x: candidate.right, y: candidate.top, endX: candidate.left, endY: candidate.bottom },
      { x: candidate.right, y: candidate.bottom, endX: candidate.left, endY: candidate.top },
      { x: candidate.left, y: candidate.bottom, endX: candidate.right, endY: candidate.top },
    ];
    const clear = corners.find(p => document.elementFromPoint(p.x, p.y) === svg);
    if (!clear) {
      const tops = corners.map(p => document.elementFromPoint(p.x, p.y)?.getAttribute('class') ||
        document.elementFromPoint(p.x, p.y)?.tagName || 'none');
      throw new Error(`lasso has no clear SVG start corner: ${JSON.stringify({ candidate, tops })}`);
    }
    return clear;
  }, box);
  await page.mouse.move(drag.x, drag.y);
  await page.mouse.down();
  await page.mouse.move(drag.endX, drag.endY, { steps: 5 });
  await page.mouse.up();
}

async function nodeLassoBox(page: Page, id: string): Promise<LassoBox> {
  return page.locator('g.node').evaluateAll((nodes: any[], wanted: string) => {
    const node = nodes.find(n => n.__data__?.id === wanted);
    if (!node) throw new Error(`node ${wanted} not rendered`);
    const r = node.querySelector('circle').getBoundingClientRect();
    return { left: r.left - 4, top: r.top - 4, right: r.right + 4, bottom: r.bottom + 4 };
  }, id);
}

async function edgePointLassoBox(page: Page, source: string, target: string, fraction: number): Promise<LassoBox> {
  return page.locator('path.link').evaluateAll((paths: any[], args: { source: string; target: string; fraction: number }) => {
    const id = (value: any) => typeof value === 'object' ? value.id : value;
    const path = paths.find(p => id(p.__data__?.source) === args.source && id(p.__data__?.target) === args.target);
    if (!path) throw new Error(`edge ${args.source}->${args.target} not rendered`);
    const local = path.getPointAtLength(path.getTotalLength() * args.fraction);
    const point = new DOMPoint(local.x, local.y).matrixTransform(path.getScreenCTM());
    const half = 14;
    return { left: point.x - half, top: point.y - half, right: point.x + half, bottom: point.y + half };
  }, { source, target, fraction });
}

async function edgeLabelLassoBox(page: Page, source: string, target: string): Promise<LassoBox> {
  await expect(page.locator('text.link-label')).not.toHaveCount(0);
  await expect(page.locator('text.link-label').first()).toBeVisible();
  return page.evaluate(({ source, target }) => {
    const id = (value: any) => typeof value === 'object' ? value.id : value;
    const labels = [...document.querySelectorAll('text.link-label')] as SVGTextElement[];
    const label = labels.find(el => id((el as any).__data__?.source) === source && id((el as any).__data__?.target) === target);
    const paths = [...document.querySelectorAll('path.link')] as SVGPathElement[];
    const path = paths.find(el => id((el as any).__data__?.source) === source && id((el as any).__data__?.target) === target);
    if (!label || !path) throw new Error(`labeled edge ${source}->${target} not rendered`);
    const r = label.getBoundingClientRect();
    const localMid = path.getPointAtLength(path.getTotalLength() / 2);
    const midpoint = new DOMPoint(localMid.x, localMid.y).matrixTransform(path.getScreenCTM()!);
    const candidates: LassoBox[] = [
      { left: r.left - 2, top: r.top - 2, right: Math.min(r.right, r.left + 12), bottom: Math.min(r.bottom, r.top + 10) },
      { left: Math.max(r.left, r.right - 12), top: r.top - 2, right: r.right + 2, bottom: Math.min(r.bottom, r.top + 10) },
      { left: r.left - 2, top: Math.max(r.top, r.bottom - 10), right: Math.min(r.right, r.left + 12), bottom: r.bottom + 2 },
      { left: Math.max(r.left, r.right - 12), top: Math.max(r.top, r.bottom - 10), right: r.right + 2, bottom: r.bottom + 2 },
    ];
    const pointIn = (p: DOMPoint, box: LassoBox) =>
      p.x >= box.left && p.x <= box.right && p.y >= box.top && p.y <= box.bottom;
    const circleIntersects = (circle: Element, box: LassoBox) => {
      const c = circle.getBoundingClientRect();
      const cx = (c.left + c.right) / 2, cy = (c.top + c.bottom) / 2;
      const radius = Math.min(c.width, c.height) / 2;
      const x = Math.max(box.left, Math.min(cx, box.right));
      const y = Math.max(box.top, Math.min(cy, box.bottom));
      return (cx - x) ** 2 + (cy - y) ** 2 <= radius ** 2;
    };
    const chosen = candidates.find(box => !pointIn(midpoint, box) &&
      ![...document.querySelectorAll('g.node circle')].some(circle => circleIntersects(circle, box)));
    if (!chosen) throw new Error(`edge label has no isolated lasso corner: ${JSON.stringify({ r, midpoint })}`);
    return chosen;
  }, { source, target });
}

async function crossingOnlyLassoBox(page: Page, source: string, target: string): Promise<LassoBox> {
  return page.evaluate(({ source, target }) => {
    const id = (value: any) => typeof value === 'object' ? value.id : value;
    const paths = [...document.querySelectorAll('path.link')] as SVGPathElement[];
    const path = paths.find(el => id((el as any).__data__?.source) === source && id((el as any).__data__?.target) === target);
    if (!path) throw new Error(`edge ${source}->${target} not rendered`);
    const screenPoint = (fraction: number) => {
      const p = path.getPointAtLength(path.getTotalLength() * fraction);
      return new DOMPoint(p.x, p.y).matrixTransform(path.getScreenCTM()!);
    };
    const midpoint = screenPoint(0.5);
    const rectIntersects = (a: DOMRect, box: LassoBox) =>
      !(a.right < box.left || a.left > box.right || a.bottom < box.top || a.top > box.bottom);
    const circleIntersects = (circle: Element, box: LassoBox) => {
      const c = circle.getBoundingClientRect();
      const cx = (c.left + c.right) / 2, cy = (c.top + c.bottom) / 2;
      const radius = Math.min(c.width, c.height) / 2;
      const x = Math.max(box.left, Math.min(cx, box.right));
      const y = Math.max(box.top, Math.min(cy, box.bottom));
      return (cx - x) ** 2 + (cy - y) ** 2 <= radius ** 2;
    };
    for (const fraction of [0.2, 0.25, 0.3, 0.7, 0.75, 0.8]) {
      const p = screenPoint(fraction);
      const box = { left: p.x - 14, top: p.y - 14, right: p.x + 14, bottom: p.y + 14 };
      const containsMidpoint = midpoint.x >= box.left && midpoint.x <= box.right && midpoint.y >= box.top && midpoint.y <= box.bottom;
      const catchesLabel = [...document.querySelectorAll('text.link-label')]
        .some(label => getComputedStyle(label).display !== 'none' && rectIntersects(label.getBoundingClientRect(), box));
      const catchesNode = [...document.querySelectorAll('g.node circle')].some(circle => circleIntersects(circle, box));
      if (!containsMidpoint && !catchesLabel && !catchesNode) return box;
    }
    throw new Error('fixture has no isolated edge-segment crossing probe');
  }, { source, target });
}

async function readSubmittedGraphSelection(s: { dir: string }, page: Page) {
  await page.locator('#fbpanel-send').click();
  await expect.poll(() => fs.existsSync(path.join(s.dir, 'result.json')), { timeout: 5000 }).toBe(true);
  const result = JSON.parse(fs.readFileSync(path.join(s.dir, 'result.json'), 'utf8'));
  const artifact = result.artifacts.find((a: any) => a.format === 'graph-selection');
  expect(artifact).toBeTruthy();
  return JSON.parse(fs.readFileSync(path.join(s.dir, artifact.path), 'utf8'));
}

test('Inspect, Select, and Lasso are explicit modes; plain clicks follow the mode and modifiers remain selection shortcuts', async ({ page }) => {
  await page.goto(surfaceURL);
  const inspect = page.getByRole('button', { name: 'Inspect', exact: true });
  const select = page.getByRole('button', { name: 'Select', exact: true });
  const lasso = page.getByRole('button', { name: 'Lasso', exact: true });

  await expect(inspect).toBeVisible();
  await expect(select).toBeVisible();
  await expect(lasso).toBeVisible();
  await expect(inspect).toHaveAttribute('aria-pressed', 'true');
  await expect(select).toHaveAttribute('aria-pressed', 'false');
  await expect(lasso).toHaveAttribute('aria-pressed', 'false');

  // Inspect mode: a plain click opens the card and does not alter membership.
  await clickNode(page, EGO);
  await expect(page.locator('#panel-comment')).toBeVisible();
  await expect(page.locator('g.node.selection-member')).toHaveCount(0);
  await page.locator('#panel-close').click();

  // Select mode: plain node and edge clicks toggle active membership.
  await select.click();
  await expect(select).toHaveAttribute('aria-pressed', 'true');
  await expect(inspect).toHaveAttribute('aria-pressed', 'false');
  await clickNode(page, EGO);
  await clickFirstEdge(page);
  await expect(page.locator('g.node.selection-member')).toHaveCount(1);
  await expect(page.locator('path.link.selection-member')).toHaveCount(1);
  await expect(page.locator('#right-col')).not.toHaveClass(/open/);

  // The modifier shortcut still toggles selection even after returning to Inspect.
  await inspect.click();
  await modifierClickNode(page, NBR_IN);
  await expect(page.locator('g.node.selection-member')).toHaveCount(2);
});

test('background drag pans normally while persistent Lasso mode previews and selects every intersected node circle', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForTimeout(1000); // zoned layout + zoomFit must settle

  const pan = await page.evaluate(() => {
    const svg = document.querySelector('svg')!;
    const r = svg.getBoundingClientRect();
    for (let y = r.top + 20; y < r.bottom - 20; y += 20) {
      for (let x = r.left + 20; x < r.right - 360; x += 20) {
        if (document.elementFromPoint(x, y) === svg && document.elementFromPoint(x + 40, y + 20) === svg)
          return { x, y };
      }
    }
    throw new Error('no clear background pan gesture found');
  });
  const beforePan = await page.evaluate(() => {
    const t = (window as any).d3.zoomTransform(document.querySelector('svg'));
    return { x: t.x, y: t.y, k: t.k };
  });
  await page.mouse.move(pan.x, pan.y);
  await page.mouse.down();
  await page.mouse.move(pan.x + 40, pan.y + 20, { steps: 4 });
  await page.mouse.up();
  const afterPan = await page.evaluate(() => {
    const t = (window as any).d3.zoomTransform(document.querySelector('svg'));
    return { x: t.x, y: t.y, k: t.k };
  });
  expect(afterPan.x - beforePan.x).toBeCloseTo(40, 0);
  expect(afterPan.y - beforePan.y).toBeCloseTo(20, 0);
  expect(afterPan.k).toBeCloseTo(beforePan.k, 5);
  await expect(page.locator('g.node.selection-member')).toHaveCount(0);

  const lasso = await page.evaluate(() => {
    const circles = [...document.querySelectorAll('g.node circle')].map(circle => {
      const r = circle.getBoundingClientRect();
      return {
        id: (circle.parentElement as any).__data__.id as string,
        cx: (r.left + r.right) / 2,
        cy: (r.top + r.bottom) / 2,
        radius: Math.min(r.width, r.height) / 2,
        top: r.top,
        right: r.right,
        bottom: r.bottom,
      };
    });
    const anchor = circles.reduce((largest, c) => c.radius > largest.radius ? c : largest);
    // Put the left boundary through the largest circle with its center outside
    // the rectangle. This guards true circle/rectangle intersection, not merely
    // selecting nodes whose centers are enclosed.
    const left = anchor.cx + anchor.radius / 2;
    const top = Math.min(...circles.map(c => c.top)) - 12;
    const right = Math.max(...circles.map(c => c.right)) + 12;
    const bottom = Math.max(...circles.map(c => c.bottom)) + 12;
    const intersects = (c: typeof circles[number]) => {
      const closestX = Math.max(left, Math.min(c.cx, right));
      const closestY = Math.max(top, Math.min(c.cy, bottom));
      return (c.cx - closestX) ** 2 + (c.cy - closestY) ** 2 <= c.radius ** 2;
    };
    const expected = circles.filter(intersects).map(c => c.id).sort();
    if (!expected.includes(anchor.id) || anchor.cx >= left)
      throw new Error('lasso fixture does not exercise boundary intersection');
    if (document.elementFromPoint(left, top) !== document.querySelector('svg'))
      throw new Error('lasso must begin on clear SVG background');
    return { left, top, right, bottom, expected };
  });

  const lassoButton = page.getByRole('button', { name: 'Lasso' });
  await expect(lassoButton).toBeVisible();
  await lassoButton.click();
  await expect(lassoButton).toHaveAttribute('aria-pressed', 'true');
  await page.mouse.move(lasso.left, lasso.top);
  await page.mouse.down();
  await page.mouse.move(lasso.right, lasso.bottom, { steps: 5 });
  const preview = page.locator('#selection-lasso');
  await expect(preview).toBeVisible();
  const previewSize = await preview.evaluate(rect => ({
    width: Number(rect.getAttribute('width')),
    height: Number(rect.getAttribute('height')),
  }));
  expect(previewSize.width).toBeCloseTo(lasso.right - lasso.left, 0);
  expect(previewSize.height).toBeCloseTo(lasso.bottom - lasso.top, 0);
  await page.mouse.up();

  await expect(preview).toBeHidden();
  // Lasso is a real interaction mode, not a one-shot armed button.
  await expect(lassoButton).toHaveAttribute('aria-pressed', 'true');
  const selected = await page.locator('g.node.selection-member').evaluateAll(nodes =>
    nodes.map((node: any) => node.__data__.id).sort());
  expect(selected).toEqual(lasso.expected);
  await expect(page.locator('#active-selection-count')).toContainText(`${lasso.expected.length} notes`);
});

test('edge-aware Lasso directly catches an edge midpoint with a real pointer gesture', async ({ page }) => {
  await settleInLassoMode(page, surfaceURL);
  await pointerLasso(page, await edgePointLassoBox(page, EGO, NBR_OUT, 0.5));

  await expect(page.locator('path.link.selection-member')).toHaveCount(1);
  await expect(page.locator('g.node.selection-member')).toHaveCount(0);
});

test('edge-aware Lasso directly catches a visible edge label even when its midpoint is outside', async ({ page }) => {
  await settleInLassoMode(page, surfaceURL);
  await pointerLasso(page, await edgeLabelLassoBox(page, EGO, NBR_OUT));

  await expect(page.locator('path.link.selection-member')).toHaveCount(1);
  await expect(page.locator('g.node.selection-member')).toHaveCount(0);
});

test('edge-aware Lasso ignores a mere edge-segment crossing without its midpoint or visible label', async ({ page }) => {
  await settleInLassoMode(page, surfaceURL);
  await pointerLasso(page, await crossingOnlyLassoBox(page, EGO, NBR_OUT));

  await expect(page.locator('path.link.selection-member')).toHaveCount(0);
  await expect(page.locator('g.node.selection-member')).toHaveCount(0);
  await expect(page.locator('#active-selection-count')).toContainText('0 notes · 0 relationships');
});

test('edge-aware Lasso artifacts preserve direct versus derived node and edge membership', async ({ page }) => {
  const s = await launchSession();
  await settleInLassoMode(page, s.url);

  // Two separate endpoint gestures avoid the relationship midpoint and label:
  // both notes are direct lasso hits, while their connecting edge is derived.
  await pointerLasso(page, await nodeLassoBox(page, EGO));
  await pointerLasso(page, await nodeLassoBox(page, NBR_OUT));
  await expect(page.locator('g.node.selection-member')).toHaveCount(2);
  await expect(page.locator('path.link.selection-member')).toHaveCount(0);
  await page.locator('#group-name').fill('Lassoed endpoints');
  await page.locator('#save-group').click();

  // A midpoint-only gesture directly catches the relationship. Its endpoint
  // notes remain outside the rectangle and are pulled into the group implicitly.
  await pointerLasso(page, await edgePointLassoBox(page, NBR_IN, EGO, 0.5));
  await expect(page.locator('path.link.selection-member')).toHaveCount(1);
  await expect(page.locator('g.node.selection-member')).toHaveCount(0);
  await page.locator('#group-name').fill('Lassoed midpoint');
  await page.locator('#save-group').click();

  // A label-only gesture is also a direct edge catch, despite excluding the
  // path midpoint and both endpoint circles.
  await pointerLasso(page, await edgeLabelLassoBox(page, EGO, NBR_OUT));
  await expect(page.locator('path.link.selection-member')).toHaveCount(1);
  await expect(page.locator('g.node.selection-member')).toHaveCount(0);
  await page.locator('#group-name').fill('Lassoed label');
  await page.locator('#save-group').click();

  // Reselect restores only direct working-state membership. The midpoint group
  // keeps its endpoint notes implicit instead of silently promoting them.
  await page.locator('.saved-group').nth(1).getByRole('button', { name: 'Reselect' }).click();
  await expect(page.locator('path.link.selection-member')).toHaveCount(1);
  await expect(page.locator('g.node.selection-member')).toHaveCount(0);

  const selection = await readSubmittedGraphSelection(s, page);
  expectCanonicalGraphSelectionShape(selection);
  expect(selection.groups).toHaveLength(3);

  expect(selection.groups[0]).toMatchObject({
    name: 'Lassoed endpoints',
    nodes: expect.arrayContaining([
      { id: EGO, selection: 'explicit' },
      { id: NBR_OUT, selection: 'explicit' },
    ]),
    edges: expect.arrayContaining([
      expect.objectContaining({ source: EGO, target: NBR_OUT, type: 'refines', selection: 'implicit' }),
    ]),
  });
  expect(selection.groups[1]).toMatchObject({
    name: 'Lassoed midpoint',
    nodes: expect.arrayContaining([
      { id: NBR_IN, selection: 'implicit' },
      { id: EGO, selection: 'implicit' },
    ]),
    edges: expect.arrayContaining([
      expect.objectContaining({ source: NBR_IN, target: EGO, type: 'supports', selection: 'explicit' }),
    ]),
  });
  expect(selection.groups[2]).toMatchObject({
    name: 'Lassoed label',
    nodes: expect.arrayContaining([
      { id: EGO, selection: 'implicit' },
      { id: NBR_OUT, selection: 'implicit' },
    ]),
    edges: expect.arrayContaining([
      expect.objectContaining({ source: EGO, target: NBR_OUT, type: 'refines', selection: 'explicit' }),
    ]),
  });
});

test('saved groups can be edited in place, reselected with explicit membership, and deleted', async ({ page }) => {
  await page.goto(surfaceURL);

  await modifierClickNode(page, EGO);
  await modifierClickNode(page, NBR_OUT);
  await modifierClickFirstEdge(page);
  await page.locator('#group-name').fill('Original group');
  await page.locator('#group-classification').selectOption('relevant');
  await page.locator('#group-comment').fill('Original comment.');
  await page.locator('#save-group').click();

  const group = page.locator('.saved-group');
  await expect(group).toHaveCount(1);
  await expect(group).toHaveAttribute('data-group-id', 'group-1');
  await expect(group.getByRole('button', { name: 'Edit' })).toBeVisible();
  await expect(group.getByRole('button', { name: 'Delete' })).toBeVisible();
  await expect(group.getByRole('button', { name: 'Reselect' })).toBeVisible();

  await group.getByRole('button', { name: 'Edit' }).click();
  await expect(page.locator('#group-name')).toHaveValue('Original group');
  await expect(page.locator('#group-classification')).toHaveValue('relevant');
  await expect(page.locator('#group-comment')).toHaveValue('Original comment.');
  await expect(page.locator('#save-group')).toHaveText('Update group');
  await page.locator('#group-name').fill('Edited group');
  await page.locator('#group-comment').fill('Edited without changing identity.');
  await page.locator('#save-group').click();

  await expect(page.locator('.saved-group')).toHaveCount(1);
  await expect(page.locator('.saved-group')).toHaveAttribute('data-group-id', 'group-1');
  await expect(page.locator('.saved-group')).toContainText('Edited group');
  await expect(page.locator('.saved-group')).toContainText('Edited without changing identity.');

  await page.locator('.saved-group').getByRole('button', { name: 'Reselect' }).click();
  await expect(page.locator('g.node.selection-member')).toHaveCount(2);
  await expect(page.locator('path.link.selection-member')).toHaveCount(1);
  await expect(page.locator('#active-selection-count')).toContainText('2 notes · 1 relationships');

  await page.locator('.saved-group').getByRole('button', { name: 'Delete' }).click();
  await expect(page.locator('.saved-group')).toHaveCount(0);
});

test('modifier multi-select saves multiple named annotated groups with explicit and implicit membership', async ({ page }) => {
  const s = await launchSession();
  await page.goto(s.url);

  await modifierClickNode(page, EGO);
  await modifierClickNode(page, NBR_OUT);
  await expect(page.locator('g.node.selection-member')).toHaveCount(2);
  await expect(page.locator('#active-selection-count')).toContainText('2 notes');
  await page.locator('#group-name').fill('Core argument');
  await page.locator('#group-classification').selectOption('belong-together');
  await page.locator('#group-comment').fill('Review these as one argument.');
  await page.locator('#save-group').click();

  await expect(page.locator('#saved-groups')).toContainText('Core argument');
  await expect(page.locator('#saved-groups')).toContainText('Review these as one argument.');
  await expect(page.locator('g.node.selection-member')).toHaveCount(0);

  await modifierClickFirstEdge(page);
  await expect(page.locator('path.link.selection-member')).toHaveCount(1);
  await page.locator('#group-name').fill('Relationship tension');
  await page.locator('#group-comment').fill('Inspect this relationship separately.');
  await page.locator('#save-group').click();
  await expect(page.locator('.saved-group')).toHaveCount(2);

  await page.locator('#fbpanel-send').click();
  await expect.poll(() => fs.existsSync(path.join(s.dir, 'result.json')), { timeout: 5000 }).toBe(true);
  const result = JSON.parse(fs.readFileSync(path.join(s.dir, 'result.json'), 'utf8'));
  const art = result.artifacts.find((a: any) => a.format === 'graph-selection');
  const selection = JSON.parse(fs.readFileSync(path.join(s.dir, art.path), 'utf8'));

  expectCanonicalGraphSelectionShape(selection);
  expect(selection.groups).toHaveLength(2);
  expect(selection.groups[0]).toMatchObject({
    id: 'group-1', name: 'Core argument', classification: 'belong-together',
    comment: 'Review these as one argument.',
  });
  expect(selection.groups[0].nodes).toEqual(expect.arrayContaining([
    { id: EGO, selection: 'explicit' },
    { id: NBR_OUT, selection: 'explicit' },
  ]));
  expect(selection.groups[0].edges).toEqual(expect.arrayContaining([
    expect.objectContaining({ source: EGO, target: NBR_OUT, type: 'refines', selection: 'implicit' }),
  ]));
  expect(selection.groups[1].edges[0].selection).toBe('explicit');
});

// retained properties [17]-[18], [21]: the three actions are mutually
// exclusive terminal submissions. Drive their actual pointer targets and prove
// one click writes one draft, posts one submit, closes the session, and emits
// only the artifact appropriate to its singular handoff.
for (const action of [
  { label: 'Send', id: '#fbpanel-send', handoff: null, formats: ['graph-selection'] },
  { label: 'Send to Canvas', id: '#send-to-canvas', handoff: 'canvas', formats: ['graph-selection', 'canvas-seed'] },
  { label: 'Send to Document', id: '#send-to-document', handoff: 'document', formats: ['graph-selection'] },
]) {
  test(`${action.label} real-pointer click submits exactly once and closes`, async ({ page }) => {
    const s = await launchSession();
    await page.goto(s.url);

    if (action.handoff === 'canvas') {
      await modifierClickNode(page, EGO);
      await modifierClickNode(page, NBR_OUT);
      await page.locator('#group-name').fill('Canvas handoff');
      await page.locator('#group-comment').fill('Explain the derived structure.');
      await page.locator('#save-group').click();
    }

    let draftRequests = 0;
    let submitRequests = 0;
    page.on('request', request => {
      const pathname = new URL(request.url()).pathname;
      if (request.method() === 'PUT' && pathname.endsWith('/draft')) draftRequests++;
      if (request.method() === 'POST' && pathname.endsWith('/submit')) submitRequests++;
    });

    const button = page.locator(action.id);
    await expect(button).toHaveText(action.label);
    await expect(button).not.toHaveAttribute('aria-pressed');
    await realPointerClick(page, button);

    await expect.poll(() => fs.existsSync(path.join(s.dir, 'result.json')), { timeout: 5000 }).toBe(true);
    await expect.poll(() => ({ draftRequests, submitRequests })).toEqual({ draftRequests: 1, submitRequests: 1 });
    await expect.poll(() => s.proc.exitCode).not.toBeNull();
    await expect(page.getByText('Feedback sent. You can close this tab.')).toBeVisible();
    await expect(page.locator('#fbpanel-actions')).toHaveCount(0);

    const result = JSON.parse(fs.readFileSync(path.join(s.dir, 'result.json'), 'utf8'));
    expect(result.artifacts.map((a: any) => a.format)).toEqual(action.formats);
    const graphArtifact = result.artifacts.find((a: any) => a.format === 'graph-selection');
    const selection = JSON.parse(fs.readFileSync(path.join(s.dir, graphArtifact.path), 'utf8'));
    expectCanonicalGraphSelectionShape(selection);
    expect(selection.handoff).toBe(action.handoff);

    const canvasSeed = result.artifacts.find((a: any) => a.format === 'canvas-seed');
    if (action.handoff === 'canvas') {
      const seed = JSON.parse(fs.readFileSync(path.join(s.dir, canvasSeed.path), 'utf8'));
      expect(seed.storage).toBe('NON_STORED');
      expect(seed.groups[0]).toMatchObject({ name: 'Canvas handoff', comment: 'Explain the derived structure.' });
    } else {
      expect(canvasSeed).toBeUndefined();
    }
  });
}

// retained property [1]: Graph Ask emits only the canonical schema and never
// mutates the notebook while promoting human feedback.
test('Send writes canonical grouped comments + overall_comment without notebook mutation', async ({ page }) => {
  const s = await launchSession();
  const headBefore = execSync('git rev-parse HEAD', { cwd: notebookDir, encoding: 'utf8' });
  const statusBefore = execSync('git status --short', { cwd: notebookDir, encoding: 'utf8' });
  await page.goto(s.url);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  await page.locator('#panel-comment').fill('this note is load-bearing');
  await page.locator('#fbpanel-answer').fill('the outbound neighbor is the strongest support');
  await page.locator('#fbpanel-send').click();

  await expect.poll(() => fs.existsSync(path.join(s.dir, 'result.json')), { timeout: 5000 }).toBe(true);
  const result = JSON.parse(fs.readFileSync(path.join(s.dir, 'result.json'), 'utf8'));
  const art = result.artifacts.find((a: any) => a.format === 'graph-selection');
  expect(art).toBeTruthy();
  const selection = JSON.parse(fs.readFileSync(path.join(s.dir, art.path), 'utf8'));
  expectCanonicalGraphSelectionShape(selection);
  expect(selection.overall_comment).toContain('strongest support');
  expect(selection.handoff).toBeNull();
  expect(selection.groups.flatMap((group: any) => group.nodes).map((node: any) => node.comment))
    .toContain('this note is load-bearing');
  expect(execSync('git rev-parse HEAD', { cwd: notebookDir, encoding: 'utf8' })).toBe(headBefore);
  expect(execSync('git status --short', { cwd: notebookDir, encoding: 'utf8' })).toBe(statusBefore);
});

// Clicking a thin/curved SVG stroke by coordinate is flaky, so drive the edge's
// own click handler directly — the same event the user's pointer produces.
test('E1+E3: clicking an edge opens a relationship comment box', async ({ page }) => {
  await page.goto(surfaceURL);
  await clickFirstEdge(page);
  await expect(page.locator('#edge-comment')).toBeVisible();
  // The affordance names the relationship (a link type is shown).
  await expect(page.locator('#edge-rel')).toBeVisible();
});

test('E4: after viewing an edge, clicking a node restores the node comment field', async ({ page }) => {
  await page.goto(surfaceURL);
  await clickFirstEdge(page);
  await expect(page.locator('#edge-comment')).toBeVisible();
  // Switching to a node must clear edge-mode so the node comment box shows.
  await page.locator('.node').first().click();
  await expect(page.locator('#panel-comment')).toBeVisible();
  await expect(page.locator('#edge-card')).toBeHidden();
});

test('E1+E2: commenting on an edge collects it and Send carries it in a canonical group', async ({ page }) => {
  const s = await launchSession();
  await page.goto(s.url);
  await clickFirstEdge(page);
  await page.locator('#edge-comment').fill('this relationship is actually a contradiction');
  // The edge appears in the persistent feedback panel.
  await expect(page.locator('#fbpanel-list')).toContainText('this relationship is actually a contradiction');

  await page.locator('#fbpanel-send').click();
  await expect.poll(() => fs.existsSync(path.join(s.dir, 'result.json')), { timeout: 5000 }).toBe(true);
  const result = JSON.parse(fs.readFileSync(path.join(s.dir, 'result.json'), 'utf8'));
  const art = result.artifacts.find((a: any) => a.format === 'graph-selection');
  const selection = JSON.parse(fs.readFileSync(path.join(s.dir, art.path), 'utf8'));
  expectCanonicalGraphSelectionShape(selection);
  const edges = selection.groups.flatMap((group: any) => group.edges);
  expect(edges.length).toBeGreaterThan(0);
  const e = edges[0];
  expect(e).toHaveProperty('source');
  expect(e).toHaveProperty('target');
  expect(e).toHaveProperty('type');
  expect(e.comment).toContain('actually a contradiction');
});

test('P: @-mention ArrowDown advances the active item by exactly one (no skips)', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  // Open several node cards first: attachMention runs on each openPanel, and if
  // it stacks keydown listeners on the shared textarea, ArrowDown fires N times
  // and skips entries. Reopen a few times to expose that.
  const nodes = page.locator('.node');
  const count = await nodes.count();
  for (let i = 0; i < Math.min(count, 3); i++) await nodes.nth(i).click();
  await nodes.first().click();
  await page.locator('#panel-comment').fill('@');
  await page.waitForSelector('#mention-menu .mention-item');

  const activeIdx = () =>
    page.$$eval('#mention-menu .mention-item', els =>
      els.findIndex(el => el.classList.contains('active')));

  const seq: number[] = [await activeIdx()];
  for (let k = 0; k < 3; k++) {
    await page.locator('#panel-comment').press('ArrowDown');
    seq.push(await activeIdx());
  }
  // Correct sequence increments by exactly 1 each press: [0,1,2,3].
  expect(seq).toEqual([0, 1, 2, 3]);
});

// P1+P2: popovers stay within the viewport (regression guards for the earlier
// round where tooltips and the @-menu rendered off-screen for bottom elements).
test('P1: the hover tooltip for the lowest node stays within the viewport', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  const inViewport = await page.evaluate(() => {
    const nodes = [...document.querySelectorAll('g.node')] as SVGGElement[];
    const lowest = nodes.sort((a, b) => b.getBoundingClientRect().top - a.getBoundingClientRect().top)[0];
    const r = lowest.getBoundingClientRect();
    lowest.dispatchEvent(new MouseEvent('mousemove', { bubbles: true, clientX: r.left, clientY: r.top }));
    const tip = document.getElementById('tooltip')!;
    if (tip.style.display !== 'block') return 'no-tip';
    const t = tip.getBoundingClientRect();
    return t.right <= window.innerWidth + 1 && t.bottom <= window.innerHeight + 1 && t.left >= -1 && t.top >= -1;
  });
  expect(inViewport).toBe(true);
});

test('P2: the @-mention menu stays within the viewport from a low comment box', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  await page.locator('#panel-comment').fill('@');
  await page.waitForSelector('#mention-menu .mention-item');
  const inViewport = await page.evaluate(() => {
    const m = document.getElementById('mention-menu')!.getBoundingClientRect();
    return m.right <= window.innerWidth + 1 && m.bottom <= window.innerHeight + 1 && m.left >= -1 && m.top >= -1;
  });
  expect(inViewport).toBe(true);
});

test('P4: @-mention candidates include edges (relationships)', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  await page.locator('#panel-comment').fill('@');
  await page.waitForSelector('#mention-menu .mention-item');
  const hasEdge = await page.$$eval('#mention-menu .mention-item', els =>
    els.some(el => (el.textContent || '').trim().startsWith('↔')));
  expect(hasEdge).toBe(true);
});

test('P5: the node whose comment box is open is highlighted in the graph', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  await expect(page.locator('g.node.comment-active')).toHaveCount(1);
});

test('H: hovering a node gives it a prominent highlight ring class', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  // Dispatch a hover on a node and assert it gains the .hovered class (the
  // prominent ring), matching the @-candidate/comment-active affordance.
  const hovered = await page.evaluate(() => {
    const n = document.querySelector('g.node') as SVGGElement;
    n.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
    n.dispatchEvent(new MouseEvent('mousemove', { bubbles: true }));
    return n.classList.contains('hovered');
  });
  expect(hovered).toBe(true);
});

test('H2: real pointer hover visibly identifies and highlights an edge', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForTimeout(1000); // initial force layout + zoomFit must settle
  const hit = page.locator('svg path.link-hit').first();
  await expect(hit).toBeVisible();
  expect(await hit.evaluate(el => parseFloat(getComputedStyle(el).strokeWidth))).toBeGreaterThanOrEqual(16);
  await expect(hit).toHaveCSS('cursor', 'pointer');
  const probe = await page.locator('svg path.link-hit').evaluateAll((paths: SVGPathElement[]) => {
    const samples: Array<{ x: number; y: number; top: string }> = [];
    for (const path of paths) {
      for (const fraction of [0.25, 0.5, 0.75]) {
        const p = path.getPointAtLength(path.getTotalLength() * fraction);
        const screen = new DOMPoint(p.x, p.y).matrixTransform(path.getScreenCTM()!);
        const top = document.elementFromPoint(screen.x, screen.y);
        if (top?.classList.contains('link-hit')) return { point: { x: screen.x, y: screen.y }, samples };
        samples.push({ x: screen.x, y: screen.y, top: top?.getAttribute('class') || top?.tagName || 'none' });
      }
    }
    return { point: null, samples };
  });
  expect(probe.point, `at least one edge segment must be a real pointer target; samples=${JSON.stringify(probe.samples)}`).not.toBeNull();
  await page.mouse.move(probe.point!.x, probe.point!.y);
  await expect(page.locator('#tooltip')).toBeVisible();
  await expect(page.locator('#tooltip .tip-title')).not.toBeEmpty();
  await expect(page.locator('svg path.link.edge-hover')).toHaveCount(1);
});

test('B: badges count nodes that recentering can actually add within ask scope', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  // FAR contributes to NBR_OUT's full-notebook degree but is outside AllowedNodes,
  // so recentering cannot reveal it and must not advertise +1.
  await expect(page.locator('text.hidden-badge')).toHaveCount(0);
});

test('C: a node with a comment gets a persistent has-comment glow in the graph', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  await page.locator('#panel-comment').fill('this note matters');
  // The commented node keeps a persistent glow (distinct from the transient
  // hover / open-card highlight) so you can see what you've commented on.
  await expect(page.locator('g.node.has-comment')).toHaveCount(1);
  // Clearing the comment removes the glow.
  await page.locator('#panel-comment').fill('');
  await expect(page.locator('g.node.has-comment')).toHaveCount(0);
});

test('D: the focus breadcrumb is a non-dismissable label on the ask surface', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  // Dx: the ✕ dismiss glyph is not visible on the crumb in feedback mode.
  await expect(page.locator('#focus-crumb-x')).toBeHidden();
  // Dc: clicking the crumb must NOT drop out of the zoned scoped view.
  await page.locator('#focus-crumb').click();
  await expect(page.locator('#btn-zoned')).toHaveClass(/active/);
});

test('F: selecting a node focuses its comment box so you can type immediately', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  await expect(page.locator('#panel-comment')).toBeFocused();
});

test('K: a keyboard key cycles selection through neighbors and focuses the comment', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  const firstTitle = await page.locator('#panel-title').textContent();
  // Cycle to the next neighbor via the keyboard.
  await page.keyboard.press('Tab');
  const secondTitle = await page.locator('#panel-title').textContent();
  expect(secondTitle).not.toEqual(firstTitle); // K1: selection advanced
  await expect(page.locator('#panel-comment')).toBeFocused(); // K2: comment focused
  // K3: the keyboard-selected node is visibly marked on the graph (exactly one).
  await expect(page.locator('g.node.node-selected')).toHaveCount(1);
});
