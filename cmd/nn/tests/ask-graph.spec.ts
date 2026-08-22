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

test.beforeAll(async () => {
  notebookDir = fs.mkdtempSync(path.join(os.tmpdir(), 'nn-askg-'));
  execSync('git init && git config user.email "t@t.com" && git config user.name "T"', { cwd: notebookDir, stdio: 'ignore' });
  writeNote(notebookDir, EGO, 'Ego Note', [{ to: NBR_OUT, type: 'refines' }]);
  writeNote(notebookDir, NBR_OUT, 'Outbound Neighbor');
  writeNote(notebookDir, NBR_IN, 'Inbound Neighbor', [{ to: EGO, type: 'supports' }]);
  writeNote(notebookDir, FAR, 'Far Node', [{ to: NBR_OUT, type: 'supports' }]); // depth-2 from ego
  execSync('git add . && git commit -m init', { cwd: notebookDir, stdio: 'ignore' });

  const repoRoot = path.resolve(__dirname, '../../..');
  execSync('go build -o /tmp/nn-askg ./cmd/nn', { cwd: repoRoot, stdio: 'ignore' });

  const cfgDir = fs.mkdtempSync(path.join(os.tmpdir(), 'nn-askg-cfg-'));
  fs.writeFileSync(path.join(cfgDir, 'config.toml'),
    `[notebooks]\ndefault = "test"\n\n[notebooks.test]\npath = ${JSON.stringify(notebookDir)}\nbackend = "gitlocal"\n`);

  proc = spawn('/tmp/nn-askg', ['ask', '--surface', 'graph', '--focus', EGO, '--instructions', 'React to this neighborhood.'],
    { env: { ...process.env, NN_CONFIG_DIR: cfgDir, NN_ASK_PRINT_URL_ONLY: '1' } });

  // Capture the surface URL the process prints instead of opening a browser.
  surfaceURL = await new Promise<string>((resolve, reject) => {
    let buf = '';
    const to = setTimeout(() => reject(new Error('no ASK_SURFACE_URL within 8s: ' + buf)), 8000);
    proc!.stdout!.on('data', d => {
      buf += d.toString();
      const m = buf.match(/ASK_SURFACE_URL (\S+)/);
      if (m) { clearTimeout(to); resolve(m[1]); }
    });
  });
  const sid = new URL(surfaceURL).searchParams.get('session')!;
  sessionDir = path.join(cfgDir, 'feedback', sid);
});

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

test('the surface shows a discoverable commentary banner and a Done button', async ({ page }) => {
  await page.goto(surfaceURL);
  await expect(page.locator('#feedback-banner')).toBeVisible();
  await expect(page.locator('#feedback-banner')).toContainText('comment');
  await expect(page.locator('#btn-done')).toBeVisible();
});

test('the surface opens in Zoned spatial layout focused on the ego', async ({ page }) => {
  await page.goto(surfaceURL);
  // Zoned reuses the directional spatial organization; the surface should adopt
  // it automatically rather than dropping the human into the flat force layout.
  await expect(page.locator('#btn-zoned')).toHaveClass(/active/);
  await expect(page.locator('#focus-crumb')).toBeVisible();
});

test('clicking a node reveals an inline comment box (commentary is discoverable on click)', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');
  await page.locator('.node').first().click();
  await expect(page.locator('#panel-comment')).toBeVisible();
  await expect(page.locator('#panel-comment-label')).toContainText('Comment');
});

test('Done → Send writes a graph-selection result with the inline comment and answer', async ({ page }) => {
  await page.goto(surfaceURL);
  await page.waitForSelector('.node');

  // Comment on a node inline, which also marks it selected.
  await page.locator('.node').first().click();
  await page.locator('#panel-comment').fill('this note is load-bearing');

  // Finish: open the brief, add a free-text answer, send.
  await page.locator('#btn-done').click();
  await expect(page.locator('#brief-modal')).toBeVisible();
  await page.locator('#brief-answer').fill('the outbound neighbor is the strongest support');
  await page.locator('#btn-submit').click();

  // The result envelope + graph-selection artifact land on disk.
  await expect.poll(() => fs.existsSync(path.join(sessionDir, 'result.json')), { timeout: 5000 }).toBe(true);
  const result = JSON.parse(fs.readFileSync(path.join(sessionDir, 'result.json'), 'utf8'));
  const art = result.artifacts.find((a: any) => a.format === 'graph-selection');
  expect(art).toBeTruthy();
  const sel = JSON.parse(fs.readFileSync(path.join(sessionDir, art.path), 'utf8'));
  expect(sel.answer).toContain('strongest support');
  expect(Object.values(sel.annotations)).toContain('this note is load-bearing');
  expect(sel.selected.length).toBeGreaterThan(0);
});
