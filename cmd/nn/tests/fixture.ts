import { execSync, spawn, ChildProcess } from 'child_process';
import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import * as net from 'net';

export const PORT = 19876;
export const BASE_URL = `http://localhost:${PORT}`;

let serverProc: ChildProcess | null = null;
export let notebookDir: string = '';

function freePort(): Promise<number> {
  return new Promise((resolve) => {
    const srv = net.createServer();
    srv.listen(0, () => {
      const addr = srv.address() as net.AddressInfo;
      srv.close(() => resolve(addr.port));
    });
  });
}

function writeNote(dir: string, id: string, title: string, body: string) {
  const content = `---
id: ${id}
title: ${title}
type: concept
status: draft
tags: []
created: 2026-01-01T00:00:00Z
modified: 2026-01-01T00:00:00Z
---

${body}
`;
  fs.writeFileSync(path.join(dir, `${id}.md`), content);
}

export async function startServer(): Promise<void> {
  notebookDir = fs.mkdtempSync(path.join(os.tmpdir(), 'nn-pw-'));

  // Init git repo (nn requires it)
  execSync('git init && git config user.email "test@test.com" && git config user.name "Test"', {
    cwd: notebookDir,
    stdio: 'ignore',
  });

  // Write two test notes
  writeNote(notebookDir, 'alpha-0001', 'Alpha Note', 'Body of alpha note.');
  writeNote(notebookDir, 'beta-0002', 'Beta Note', 'Body of beta note.');
  execSync('git add . && git commit -m "init"', { cwd: notebookDir, stdio: 'ignore' });

  // Build the nn binary
  const repoRoot = path.resolve(__dirname, '../../..');
  execSync('go build -o /tmp/nn-test ./cmd/nn', { cwd: repoRoot, stdio: 'ignore' });

  // Write config.toml pointing at the notebook dir
  const cfgDir = fs.mkdtempSync(path.join(os.tmpdir(), 'nn-pw-cfg-'));
  const cfgPath = path.join(cfgDir, 'config.toml');
  fs.writeFileSync(cfgPath, `[notebooks]\ndefault = "test"\n\n[notebooks.test]\npath = ${JSON.stringify(notebookDir)}\nbackend = "gitlocal"\n`);

  serverProc = spawn('/tmp/nn-test', [
    'graph', 'export', '--format', 'html', '--serve',
    '--port', String(PORT),
  ], { stdio: 'ignore', env: { ...process.env, NN_CONFIG_DIR: cfgDir } });

  // Wait for server to be ready
  const deadline = Date.now() + 5000;
  while (Date.now() < deadline) {
    try {
      const resp = await fetch(`${BASE_URL}/`);
      if (resp.ok) return;
    } catch {}
    await new Promise(r => setTimeout(r, 50));
  }
  throw new Error(`Server did not start on port ${PORT}`);
}

export function stopServer(): void {
  serverProc?.kill();
  serverProc = null;
  if (notebookDir) fs.rmSync(notebookDir, { recursive: true, force: true });
}
