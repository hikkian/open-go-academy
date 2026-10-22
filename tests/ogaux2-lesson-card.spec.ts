import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// OGAUX2 (docs/LESSON_UI_SPEC.md §4): the task card, the expected output, the criteria, the «Ждали / Получили» block and the 422 answer.
const lessonTitle = 'Что такое компьютер, программа и Go';
const wrongOutput = 'Привет, Go!\n';
const submitVerdict = {
  stdout: wrongOutput, stderr: '', exit_code: 0, status: 'ok', passed: false,
  checks: [
    { name: 'Программа успешно завершилась', passed: true, criterion: 0 },
    { name: 'Публичный пример вывода', passed: false, expected: 'Go готов!\n', actual: wrongOutput, criterion: 1,
      hint: 'вывод должен совпасть буква в букву, включая знаки препинания и регистр' },
    { name: 'Точная проверка строк и отсутствия лишнего вывода', passed: false, expected: 'Go готов!\n', actual: wrongOutput, criterion: 2,
      hint: 'есть лишние строки или пробелы: см. блоки «ждали» и «получили»' },
    { name: 'Обязательная структура задания', passed: true, criterion: 3 },
  ],
};

async function enter(page: any) {
  await page.goto('/');
  await expect.poll(async () => await page.getByLabel('Твоё имя').isVisible() || await page.getByRole('heading', { name: 'Твоя практика начинается здесь.' }).isVisible()).toBe(true);
  if (await page.getByLabel('Твоё имя').isVisible()) {
    await page.getByLabel('Твоё имя').fill('Ада');
    await page.getByRole('button', { name: 'Создать профиль' }).click();
  }
  await expect(page.getByRole('heading', { name: 'Твоя практика начинается здесь.' })).toBeVisible();
}

async function openZero01(page: any) {
  await page.getByRole('button', { name: new RegExp(lessonTitle) }).first().click();
  await expect(page.locator('.task-card')).toBeVisible();
}

test('the task card shows the goal, the expected output, the criteria and a collapsed hint', async ({ page }, info) => {
  await enter(page);
  await openZero01(page);
  const card = page.locator('.task-card');
  await expect(card.locator('.task-card-title')).toHaveText('Поздоровайся с компьютером');
  const expand = card.getByRole('button', { name: 'Развернуть' });
  if (await expand.isVisible()) await expand.click();
  await expect(card.locator('.task-expected-output')).toContainText('Go готов!');
  await expect(card.locator('.task-criteria li')).toHaveCount(4);
  const hint = card.locator('details').filter({ hasText: 'Подсказка' });
  await expect(hint).toHaveCount(1);
  await expect(hint).not.toHaveAttribute('open', '');
  await hint.locator('summary').click();
  await expect(hint).toHaveAttribute('open', '');
  await expect(page.locator('.task-chips')).toContainText('01');
  await expect(page.locator('.task-chips button')).toHaveCount(3);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  const axe = await new AxeBuilder({ page }).include('.task-card').analyze();
  expect(axe.violations.map(v => v.id)).toEqual([]);
  await page.screenshot({ path: `artifacts/${info.project.name}-task-card.png`, fullPage: true });
});

test('a rejected answer shows what was expected and what came out', async ({ page }, info) => {
  await enter(page);
  await page.route('**/api/exercises/hello/submit', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(submitVerdict) }));
  await openZero01(page);
  await page.getByRole('button', { name: 'Проверить', exact: true }).click();
  await expect(page.getByText('Не принято — вот что разошлось.')).toBeVisible();
  const compare = page.locator('.check-compare').first();
  await expect(compare.locator('h4')).toHaveText(['Ждали', 'Получили']);
  await expect(compare.locator('.compare-lines li.mismatch').first()).toBeVisible();
  await expect(compare.locator('.compare-lines').nth(0)).toContainText('Go·готов!⏎');
  await expect(compare.locator('.compare-lines').nth(1)).toContainText('Привет,·Go!⏎');
  const card = page.locator('.task-card');
  const expand = card.getByRole('button', { name: 'Развернуть' });
  if (await expand.isVisible()) await expand.click();
  await expect(card.locator('.criterion-fail')).toHaveCount(2);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  await page.screenshot({ path: `artifacts/${info.project.name}-compare.png`, fullPage: true });
});

test('a refused code (422) is an answer in the result, not an error banner', async ({ page }) => {
  await enter(page);
  await page.route('**/api/exercises/hello/run', route => route.fulfill({ status: 422, contentType: 'application/json', body: JSON.stringify({ error: 'импорт "os" не разрешён в этом задании' }) }));
  await openZero01(page);
  await page.getByRole('button', { name: 'Запустить', exact: true }).click();
  await expect(page.locator('.result-status')).toHaveText('Не принято — код не допущен к проверке');
  await expect(page.getByText('Не удалось выполнить действие')).toHaveCount(0);
});
