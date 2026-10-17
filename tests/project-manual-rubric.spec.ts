import {test, expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const requirements = [
  '[M27-access-boundaries] Проверь матрицу actor/tenant/role и отрицательные IDOR-сценарии с реальными HTTP/SQL результатами.',
  '[M28-dependency-graph] Покажи типизированный граф handler → service → repository, composition root, владение ресурсами и cleanup.',
  '[M30-32-operations] Приложи фактически полученные logs/metrics/traces с источниками и ограниченной cardinality.',
  '[M33-slo-error-budget] Опиши SLI, SLO и error budget и приложи фиксированный workload с наблюдёнными ошибками и latency, а также параметры машины.',
  '[M34-35-fault-resilience] Покажи fault injection, восстановление, идемпотентность и освобождение ресурсов.',
  '[M36-40-release-evidence] Приложи release evidence, rollback-план, совместимость данных и оставшиеся риски.',
  '<img src=x onerror=alert(1)>',
];

async function openProject(page: import('@playwright/test').Page) {
  let writeRequests = 0;
  await page.route('**/api/**', async route => {
    const request = route.request();
    const {pathname} = new URL(request.url());
    if (request.method() !== 'GET') writeRequests += 1;
    if (pathname === '/api/profile') return route.fulfill({json: {profile: {id: 'local', name: 'Rubric learner', theme: 'light', rank: 'Zero', xp: 0}, csrf_token: 'csrf'}});
    if (pathname === '/api/lessons') return route.fulfill({json: []});
    if (pathname === '/api/progress') return route.fulfill({json: {completed: [], attempts: [], xp: 0}});
    if (pathname === '/api/learning') return route.fulfill({json: {policy_version: 'v1', current_rank: 'Zero', verified: false, ranks: [], skills: [], review_exercises: []}});
    if (pathname === '/api/projects' && request.method() === 'GET') return route.fulfill({json: [{id: 'p3', title: 'TaskFlow backend', rank: 'Middle', description: 'Практический проект'}]});
    if (pathname === '/api/projects/p3' && request.method() === 'GET') return route.fulfill({json: {
      id: 'p3', title: 'TaskFlow backend', rank: 'Middle', description: 'Практический проект', execution_available: true,
      requirements: ['Реализуй сервис'], milestones: [], starter_files: {'main.go': 'package main\n', 'README.md': '# TaskFlow\n', 'go.mod': 'module taskflow\n\ngo 1.27.2\n'}, public_examples: [],
      manual_review_rubric: {version: 'p3-taskflow-v1-manual-review-v2', hash: 'a'.repeat(64), project_id: 'p3', project: 'TaskFlow backend', requirements, milestones: []},
    }});
    if (pathname === '/api/projects/p3/files') return route.fulfill({json: {files: {'main.go': 'package main\n', 'README.md': '# TaskFlow\n', 'go.mod': 'module taskflow\n\ngo 1.27.2\n'}}});
    if (pathname === '/api/projects/p3/snapshots') return route.fulfill({json: []});
    return route.fulfill({status: 404, json: {error: `Unmocked ${request.method()} ${pathname}`}});
  });
  await page.goto('/');
  await page.getByRole('button', {name: 'Проекты', exact: true}).click();
  await page.getByRole('button', {name: /TaskFlow backend/}).click();
  await expect(page.getByRole('heading', {name: 'TaskFlow backend'})).toBeVisible();
  return () => writeRequests;
}

test('P3 ручные критерии доступны с клавиатуры до отправки и не исполняют содержимое API', async ({page}, info) => {
  const writeCount = await openProject(page);
  const criteria = page.getByText('Критерии ручной приёмки', {exact: true});
  await expect(criteria).toBeVisible();
  const details = page.locator('details.manual-rubric-details');
  await expect(details).not.toHaveAttribute('open', '');
  expect(writeCount()).toBe(0);

  await criteria.focus();
  await page.keyboard.press('Enter');
  await expect(details).toHaveAttribute('open', '');
  for (const criterion of requirements.slice(0, -1)) await expect(details).toContainText(criterion);
  await expect(details.locator('img')).toHaveCount(0);
  await expect(details).toContainText('Автоматическая проверка подтверждает только выполненные программные проверки.');
  await expect(details).toContainText('Версия критериев: p3-taskflow-v1-manual-review-v2');
  expect(writeCount()).toBe(0);

  const axe = await new AxeBuilder({page}).analyze();
  expect(axe.violations.map(v => ({id: v.id, targets: v.nodes.map(n => n.target)}))).toEqual([]);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({path: `artifacts/${info.project.name}-project-manual-rubric.png`, fullPage: true});
});
