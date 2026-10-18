import {test, expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import {readFileSync, readdirSync} from 'node:fs';
import {join, relative} from 'node:path';

function collectGoFiles(root: string, dir = root): Record<string, string> {
  const files: Record<string, string> = {};
  for (const entry of readdirSync(dir, {withFileTypes: true})) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) Object.assign(files, collectGoFiles(root, path));
    else if (entry.name.endsWith('.go')) files[relative(root, path)] = readFileSync(path, 'utf8');
  }
  return files;
}

test('P4/P5 isolated submissions persist machine evidence but never award rank', async ({page}, info) => {
  test.skip(process.env.ACADEMY_E2E_PROJECT_RUNNER !== '1', 'requires the real rootless OCI project runner');
  test.skip(info.project.name !== 'desktop', 'one real runner golden path covers both profiles; responsive UI is covered separately');
  test.setTimeout(300_000);
  await page.goto('/');
  if (await page.getByLabel('Твоё имя').isVisible()) {
    await page.getByLabel('Твоё имя').fill('P4 P5 acceptance');
    await page.getByRole('button', {name: 'Создать профиль'}).click();
  }
  const before = await (await page.request.get('/api/learning')).json();
  const csrf = (await (await page.request.get('/api/profile')).json()).csrf_token;

  for (const id of ['p4', 'p5'] as const) {
    const response = await page.request.get(`/api/projects/${id}/files`);
    expect(response.ok(), `${id} starter is available`).toBe(true);
    const files = (await response.json()).files as Record<string, string>;
    Object.assign(files, collectGoFiles(`content/projects/${id}/reference`));
    files['README.md'] = readFileSync(`content/projects/${id}/README.md`, 'utf8');
    if (id === 'p5') {
      const reference = 'content/projects/p5/reference';
      for (const [from, to] of [
        ['system-design.md', 'design/system-design.md'],
        ['operations.md', 'design/operations.md'],
        ['incident-postmortem.md', 'design/incident-postmortem.md'],
      ]) files[to] = readFileSync(join(reference, from), 'utf8');
      for (const entry of readdirSync(`${reference}/adr`, {withFileTypes: true})) {
        if (entry.isFile() && entry.name.endsWith('.md')) {
          files[`design/adr/${entry.name}`] = readFileSync(join(reference, 'adr', entry.name), 'utf8');
        }
      }
    }
    const saved = await page.request.put(`/api/projects/${id}/files`, {headers: {'X-CSRF-Token': csrf}, data: {files}});
    expect(saved.ok(), `${id} reference artifact saves`).toBe(true);

    await page.reload();
    await page.getByRole('button', {name: 'Проекты', exact: true}).click();
    await page.getByRole('button', {name: id === 'p4' ? /ScaleLab/ : /Architecture.*Capstone|Capstone/}).click();
    await expect(page.getByRole('heading', {name: id === 'p4' ? /ScaleLab/ : /Capstone/})).toBeVisible();
    await page.getByRole('button', {name: 'Проверить проект', exact: true}).click();
    const output = page.locator('.project-output');
    await expect(output).toContainText('Автоматические проверки пройдены.', {timeout: 220_000});
    await expect(output).toContainText('automatic_pass');
    await expect(output).toContainText('не присваивает ранг');
    await expect(output).toContainText('требуется отдельное ревью');
    if (id === 'p4') {
      const malicious = {...files, 'main.go': 'package main\nimport "os"\nfunc init(){ os.Exit(0) }\n', 'exit_test.go': 'package main\nimport ("os"; "testing")\nfunc TestExit(t *testing.T){ os.Exit(0) }\n'};
      const blocked = await page.request.post('/api/projects/p4/submit', {headers: {'X-CSRF-Token': csrf}, data: {files: malicious}});
      expect(blocked.status()).toBe(422);
      expect(await blocked.text()).toContain('не может');
    }
    const after = await (await page.request.get('/api/learning')).json();
    expect(after.current_rank).toBe(before.current_rank);
    expect(after.verified).toBe(before.verified);
    const axe = await new AxeBuilder({page}).analyze();
    expect(axe.violations.map(v => ({id: v.id, targets: v.nodes.map(n => n.target)}))).toEqual([]);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await page.screenshot({path: `artifacts/desktop-${id}-evidence.png`, fullPage: true});
  }
});

test('P4/P5 workspaces remain accessible and fit a mobile viewport', async ({page}, info) => {
  test.skip(process.env.ACADEMY_E2E_PROJECT_RUNNER !== '1', 'requires verified P4/P5 catalog profiles');
  test.skip(info.project.name !== 'mobile', 'mobile layout is checked once per viewport');
  await page.goto('/');
  if (await page.getByLabel('Твоё имя').isVisible()) {
    await page.getByLabel('Твоё имя').fill('P4 P5 mobile acceptance');
    await page.getByRole('button', {name: 'Создать профиль'}).click();
  }
  for (const id of ['p4', 'p5'] as const) {
    await page.getByRole('button', {name: 'Проекты', exact: true}).click();
    await page.getByRole('button', {name: id === 'p4' ? /ScaleLab/ : /Architecture.*Capstone|Capstone/}).click();
    await expect(page.getByRole('heading', {name: id === 'p4' ? /ScaleLab/ : /Capstone/})).toBeVisible();
    const axe = await new AxeBuilder({page}).analyze();
    expect(axe.violations.map(v => ({id: v.id, targets: v.nodes.map(n => n.target)}))).toEqual([]);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    const downloadPromise = page.waitForEvent('download');
    await page.getByRole('button', {name: 'Скачать ZIP', exact: true}).click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toBe(`${id}.zip`);
    const archivePath = await download.path();
    expect(archivePath).toBeTruthy();
    const archive = readFileSync(archivePath!);
    expect(archive.subarray(0, 2).toString()).toBe('PK');
    expect(archive.toString()).toContain(id === 'p4' ? 'aggregate.go' : 'queue.go');
    await page.screenshot({path: `artifacts/mobile-${id}-workspace.png`, fullPage: true});
    await page.getByRole('button', {name: '‹ К проектам'}).click();
  }
});
