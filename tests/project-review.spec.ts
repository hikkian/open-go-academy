import {test, expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const evidence = {
  submission_id: 'submission-review-1234567890', evaluation_id: 'evaluation-review-1234567890',
  project_version: 'p3-taskflow-v1', criteria_hash: `sha256:${'1'.repeat(64)}`,
  artifact_hash: `sha256:${'2'.repeat(64)}`, executed_source_hash: `sha256:${'3'.repeat(64)}`,
  checker_version: 'taskflow-v1', checker_digest: `sha256:${'4'.repeat(64)}`,
  runtime_policy_hash: `sha256:${'5'.repeat(64)}`, rubric_version: 'taskflow-auto-v1',
  rubric_hash: `sha256:${'6'.repeat(64)}`, checks_hash: `sha256:${'7'.repeat(64)}`,
  evaluation_status: 'automatic_pass', exit_code: 0,
};

const reviewPacket = {
  project_id: 'p3', submission_id: evidence.submission_id, evaluation_id: evidence.evaluation_id,
  project_version: evidence.project_version, criteria_hash: evidence.criteria_hash,
  artifact_schema_version: 'project-artifact-v1', artifact_hash: '2'.repeat(64),
  executed_source_hash: evidence.executed_source_hash, evaluation_status: 'automatic_pass',
  checks_hash: evidence.checks_hash, checker_version: evidence.checker_version, checker_digest: evidence.checker_digest,
  rubric: {version: 'p3-v1-manual-review-v1', hash: '8'.repeat(64), project_id: 'p3', project: 'TaskFlow backend', requirements: ['HTTP'], milestones: [], rubric_files: {'docs/rubric.md': '# Review rubric'}},
  attestation_template: {
    version: 1, reviewer_id: 'replace-with-allowlisted-reviewer-id', submission_id: evidence.submission_id,
    evaluation_id: evidence.evaluation_id, project_id: 'p3', project_version: evidence.project_version,
    artifact_hash: '2'.repeat(64), rubric_version: 'p3-v1-manual-review-v1', rubric_hash: '8'.repeat(64),
    decision: '', comments: '', issued_at: '2026-10-16T12:00:00Z', signature: '',
  },
  files: {'main.go': 'package main\n', 'README.md': '# TaskFlow'},
};

const signedApproval = {
  version: 1, reviewer_id: 'operator-reviewer', submission_id: evidence.submission_id,
  evaluation_id: evidence.evaluation_id, project_id: 'p3', project_version: evidence.project_version,
  artifact_hash: '2'.repeat(64), rubric_version: reviewPacket.rubric.version, rubric_hash: reviewPacket.rubric.hash,
  decision: 'approve', comments: 'Reviewed.', issued_at: '2026-10-16T12:00:00Z', signature: 'c2lnbmF0dXJl',
};
const reviewerKeyFingerprint = `sha256:${'a'.repeat(64)}`;

async function setupReviewPage(page: import('@playwright/test').Page) {
  const persistedReviews: Record<string, unknown>[] = [];
  const projectFiles = {
    'main.go': 'package main\nfunc main() {}\n', 'README.md': '# TaskFlow\n',
    'go.mod': 'module taskflow\n\ngo 1.27.2\n', 'go.sum': 'locked sums\n',
  };
  await page.route('**/api/**', async route => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    if (path === '/api/profile') return route.fulfill({json: {profile: {id: 'local', name: 'Review learner', theme: 'light', rank: 'Zero', xp: 0}, csrf_token: 'csrf'}});
    if (path === '/api/lessons') return route.fulfill({json: []});
    if (path === '/api/progress') return route.fulfill({json: {completed: [], attempts: [], xp: 0}});
    if (path === '/api/learning') return route.fulfill({json: {policy_version: 'v1', current_rank: 'Zero', verified: false, ranks: [], skills: [], review_exercises: []}});
    if (path === '/api/projects' && request.method() === 'GET') return route.fulfill({json: [{id: 'p3', title: 'TaskFlow backend', rank: 'Middle', description: 'Практический проект'}]});
    if (path === '/api/projects/p3' && request.method() === 'GET') return route.fulfill({json: {id: 'p3', title: 'TaskFlow backend', rank: 'Middle', description: 'Практический проект', execution_available: true, requirements: ['Реализуй сервис'], milestones: [], starter_files: projectFiles, public_examples: []}});
    if (path === '/api/projects/p3/files' && request.method() === 'GET') return route.fulfill({json: {files: projectFiles}});
    if (path === '/api/projects/p3/snapshots') return route.fulfill({json: []});
    if (path === '/api/projects/p3/submit' && request.method() === 'POST') return route.fulfill({json: {status: 'automatic_pass', exit_code: 0, passed: true, checks: [{name: 'acceptance', passed: true}], manual_review_required: true, manual_review_labels: ['Архитектура'], project_evidence: evidence, rank_awarded: false}});
    if (path.endsWith('/review-packet') && request.method() === 'GET') return route.fulfill({json: reviewPacket});
    if (path.endsWith('/reviews') && request.method() === 'GET') return route.fulfill({json: {reviews: persistedReviews}});
    if (path.endsWith('/reviews') && request.method() === 'POST') {
      const body = request.postDataJSON();
      if (body.decision === 'reject') return route.fulfill({status: 422, json: {error: 'подпись reviewer не прошла проверку доверенного allowlist'}});
      const saved = {id: 'review-saved-1234567890', reviewer_id: 'operator-reviewer', reviewer_key_fingerprint: reviewerKeyFingerprint, decision: 'approve', comments: 'Reviewed.', artifact_hash: '2'.repeat(64), rubric_version: reviewPacket.rubric.version, rubric_hash: reviewPacket.rubric.hash, attestation_digest: `sha256:${'9'.repeat(64)}`, created_at: '2026-10-16T12:01:00Z'};
      if (!persistedReviews.some(item => item.id === saved.id)) persistedReviews.push(saved);
      return route.fulfill({status: 201, json: {review_id: saved.id, reviewer_id: saved.reviewer_id, reviewer_key_fingerprint: saved.reviewer_key_fingerprint, decision: saved.decision, artifact_hash: saved.artifact_hash, rubric_version: saved.rubric_version, rubric_hash: saved.rubric_hash, attestation_digest: saved.attestation_digest, created_at: saved.created_at, rank_awarded: false}});
    }
    return route.fulfill({status: 404, json: {error: `Unmocked ${request.method()} ${path}`}});
  });
  await page.goto('/');
  await page.getByRole('button', {name: 'Проекты', exact: true}).click();
  await page.getByRole('button', {name: /TaskFlow backend/}).click();
  await expect(page.getByRole('heading', {name: 'TaskFlow backend'})).toBeVisible();
  await page.getByRole('button', {name: 'Проверить проект', exact: true}).click();
  await expect(page.getByRole('heading', {name: 'Подписанное ручное ревью'})).toBeVisible();
  const pointer = await page.evaluate(() => JSON.parse(localStorage.getItem('open-go-academy:p3:last-review-evidence') || 'null'));
  expect(pointer).toEqual({submission_id: evidence.submission_id, evaluation_id: evidence.evaluation_id});
}

test('passing evidence can export packet and import signed review with explicit trust limits', async ({page}, info) => {
  await setupReviewPage(page);
  await expect(page.getByText(/если ключ reviewer не настроен, запрос будет отклонён/)).toBeVisible();
  await expect(page.getByText(/Личность и независимость reviewer зависят от внешней настройки/)).toBeVisible();
  await expect(page.getByText('rank', {exact: true})).toHaveCount(0);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);

  const packetDownload = page.waitForEvent('download');
  await page.getByRole('button', {name: 'Скачать JSON review packet'}).click();
  const download = await packetDownload;
  expect(download.suggestedFilename()).toContain('submission-review-1234567890-evaluation-review-1234567890-review-packet.json');
  const packetPath = await download.path();
  expect(packetPath).toBeTruthy();
  const {readFileSync} = await import('node:fs');
  const exported = JSON.parse(readFileSync(packetPath!, 'utf8'));
  expect(exported).toEqual(reviewPacket);
  expect(exported.attestation_template.artifact_hash).toBe(exported.artifact_hash);
  expect(exported.attestation_template.rubric_hash).toBe(exported.rubric.hash);
  expect(exported.attestation_template.signature).toBe('');
  await expect(page.getByText(/Пакет для ручного ревью скачан/)).toBeVisible();

  const upload = page.getByLabel('Загрузить подписанное решение reviewer JSON');
  await upload.setInputFiles({name: 'signed-review.json', mimeType: 'application/json', buffer: Buffer.from(JSON.stringify(signedApproval))});
  const reviewRegion = page.getByRole('region', {name: 'Подписанное ручное ревью'});
  const history = page.locator('details').filter({hasText: 'Сохранённые решения reviewer'});
  await expect(reviewRegion.getByRole('status')).toContainText('Подписанное решение сохранено');
  await expect(reviewRegion.getByRole('status')).toContainText('review-saved-1234567890');
  await expect(reviewRegion.getByRole('status')).toContainText(reviewerKeyFingerprint);
  await expect(reviewRegion.getByRole('status')).toContainText('Fingerprint идентифицирует ключ, не человека; ранг автоматически не начислен.');
  await expect(page.getByText(/Сохранённые решения reviewer \(1\)/)).toBeVisible();
  await expect(history).toContainText('review-saved-1234567890');
  await expect(history).toContainText(reviewerKeyFingerprint);
  await history.locator('summary').click();
  await expect(history).toContainText(reviewerKeyFingerprint);

  await upload.setInputFiles({name: 'invalid-review.json', mimeType: 'application/json', buffer: Buffer.from(JSON.stringify({...signedApproval, decision: 'reject'}))});
  await expect(page.getByRole('alert')).toContainText('allowlist');
  await expect(reviewRegion.getByRole('status')).toHaveCount(0);
  const axe = await new AxeBuilder({page}).analyze();
  expect(axe.violations.map(v => ({id: v.id, targets: v.nodes.map(n => n.target)}))).toEqual([]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await page.screenshot({path: `artifacts/${info.project.name}-project-review-flow.png`, fullPage: true});

  await page.reload();
  await page.getByRole('button', {name: 'Проекты', exact: true}).click();
  await page.getByRole('button', {name: /TaskFlow backend/}).click();
  await expect(page.locator('.cm-editor')).toBeVisible();
  await expect(page.getByRole('heading', {name: 'Подписанное ручное ревью'})).toBeVisible();
  await expect(page.getByRole('button', {name: 'Скачать JSON review packet'})).toBeEnabled();
  await expect(page.getByLabel('Загрузить подписанное решение reviewer JSON')).toBeEnabled();
  const reloadedPacketDownload = page.waitForEvent('download');
  await page.getByRole('button', {name: 'Скачать JSON review packet'}).click();
  expect((await reloadedPacketDownload).suggestedFilename()).toContain('review-packet.json');
  await expect(page.getByText(/Сохранённые решения reviewer \(1\)/)).toBeVisible();
  const reloadedHistory = page.locator('details').filter({hasText: 'Сохранённые решения reviewer'});
  await expect(reloadedHistory).toContainText('review-saved-1234567890');
  await expect(reloadedHistory).toContainText('не начисляет ранг');
  const historyAxe = await new AxeBuilder({page}).analyze();
  expect(historyAxe.violations.map(v => ({id: v.id, impact: v.impact, nodes: v.nodes.map(n => ({target: n.target, html: n.html, failureSummary: n.failureSummary, any: n.any.map(check => ({message: check.message, data: check.data}))}))})), 'Full page axe audit after workspace reload').toEqual([]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  await upload.setInputFiles({name: 'signed-review-again.json', mimeType: 'application/json', buffer: Buffer.from(JSON.stringify(signedApproval))});
  await expect(page.getByRole('region', {name: 'Подписанное ручное ревью'}).getByRole('status')).toContainText('review-saved-1234567890');
  await expect(reloadedHistory).toContainText('review-saved-1234567890');
  await page.screenshot({path: `artifacts/${info.project.name}-project-review-reload.png`, fullPage: true});

  await page.evaluate(() => localStorage.setItem('open-go-academy:p3:last-review-evidence', JSON.stringify({submission_id: 'untrusted-submission', evaluation_id: 'untrusted-evaluation', decision: 'approve', rank: 'Senior'})));
  await page.reload();
  await page.getByRole('button', {name: 'Проекты', exact: true}).click();
  await page.getByRole('button', {name: /TaskFlow backend/}).click();
  await expect(page.locator('.cm-editor')).toBeVisible();
  await expect.poll(() => page.evaluate(() => localStorage.getItem('open-go-academy:p3:last-review-evidence'))).toBeNull();
  await expect(page.getByRole('heading', {name: 'Подписанное ручное ревью'})).toHaveCount(0);
});
