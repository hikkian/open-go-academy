import assert from 'node:assert/strict';
import { mkdir, readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';
import AxeBuilder from '@axe-core/playwright';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const data = JSON.parse(await readFile(path.join(root, 'docs/project-progress.json'), 'utf8'));
const target = path.join(root, 'docs/PROJECT_PROGRESS.html');
const screenshots = path.join(root, 'artifacts');
await mkdir(screenshots, { recursive: true });

const browser = await chromium.launch({ headless: true });
try {
  for (const [name, viewport, colorScheme] of [
    ['desktop', { width: 1440, height: 1100 }, 'light'],
    ['mobile', { width: 390, height: 844 }, 'dark'],
  ]) {
    const context = await browser.newContext({ viewport, colorScheme });
    try {
      const page = await context.newPage();
      await page.goto(new URL(`file://${target}`).href);
      await page.locator('h1').waitFor();
      const tracks = page.locator('article.track');
      if (data.m11_acceptance?.status === 'running') {
        assert.match(await page.locator('.updated').innerText(), /выполняется/,
          `${name}: active fingerprint-bound gate must be visible`);
      }
      assert.equal(await page.getByRole('heading', { name: 'M11 — курс Middle → Senior', exact: true }).count(), 1,
        `${name}: advanced course workstream must be visible`);
      const m11Track = tracks.filter({ has: page.getByRole('heading', { name: 'M11 — курс Middle → Senior', exact: true }) });
      assert.match(await m11Track.innerText(), /88 из 88/, `${name}: all advanced chapter bundles must be counted`);
      if (data.m11_acceptance?.status !== 'passed') {
        assert.match(await m11Track.innerText(), /gate/i, `${name}: incomplete acceptance must remain explicit`);
      }
      assert.equal(await tracks.count(), data.workstreams.length, `${name}: workstream count`);
      assert.equal(await page.locator('.milestone').count(), data.milestones.length, `${name}: milestone count`);
      for (let i = 0; i < data.workstreams.length; i++) {
        const item = data.workstreams[i];
        const progress = tracks.nth(i).locator('progress');
        assert.equal(Number(await progress.getAttribute('value')), item.value, `${name}: ${item.title} value`);
        assert.equal(Number(await progress.getAttribute('max')), item.max, `${name}: ${item.title} maximum`);
      }
      const width = await page.evaluate(() => ({
        viewport: document.documentElement.clientWidth,
        document: document.documentElement.scrollWidth,
      }));
      assert.ok(width.document <= width.viewport, `${name}: horizontal overflow ${JSON.stringify(width)}`);
      const accessibility = await new AxeBuilder({ page }).analyze();
      assert.deepEqual(accessibility.violations.map((violation) => violation.id), [], `${name}: axe violations`);
      await page.screenshot({ path: path.join(screenshots, `project-progress-${name}-latest.png`), fullPage: true });
      console.log(`PASS ${name} (${colorScheme}): ${data.workstreams.length} workstreams, no overflow, axe clean`);
    } finally {
      await context.close();
    }
  }
} finally {
  await browser.close();
}
