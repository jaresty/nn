import { test, expect, Page } from '@playwright/test';
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

test('Send writes a graph-selection result from the collected comments + answer', async ({ page }) => {
  const s = await launchSession();
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
  const sel = JSON.parse(fs.readFileSync(path.join(s.dir, art.path), 'utf8'));
  expect(sel.answer).toContain('strongest support');
  expect(Object.values(sel.annotations)).toContain('this note is load-bearing');
  expect(sel.selected.length).toBeGreaterThan(0);
});

// Clicking a thin/curved SVG stroke by coordinate is flaky, so drive the edge's
// own click handler directly — the same event the user's pointer produces.
async function clickFirstEdge(page: Page) {
  // link-hit paths are transparent (not "visible"), so wait for attachment.
  await page.waitForSelector('path.link-hit', { state: 'attached' });
  await page.evaluate(() => {
    const hit = document.querySelector('path.link-hit') as SVGElement;
    hit.dispatchEvent(new MouseEvent('click', { bubbles: true }));
  });
}

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

test('E1+E2: commenting on an edge collects it and Send carries it in edges[]', async ({ page }) => {
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
  const sel = JSON.parse(fs.readFileSync(path.join(s.dir, art.path), 'utf8'));
  expect(Array.isArray(sel.edges)).toBe(true);
  expect(sel.edges.length).toBeGreaterThan(0);
  const e = sel.edges[0];
  expect(e).toHaveProperty('source');
  expect(e).toHaveProperty('target');
  expect(e).toHaveProperty('type');
  expect(e.comment).toContain('actually a contradiction');
});
