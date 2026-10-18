import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

async function enter(page: import('@playwright/test').Page) {
  await page.goto('/');
  await expect.poll(async () =>
    await page.getByLabel('Твоё имя').isVisible() ||
    await page.getByRole('heading', { name: 'Твоя практика начинается здесь.' }).isVisible(),
  ).toBe(true);
  if (await page.getByLabel('Твоё имя').isVisible()) {
    await page.getByLabel('Твоё имя').fill('M12 keyboard audit');
    await page.getByRole('button', { name: 'Создать профиль' }).click();
  }
  await expect(page.getByRole('heading', { name: 'Твоя практика начинается здесь.' })).toBeVisible();
}

test('keyboard focus, reduced motion and responsive accessibility at tablet and wide viewports', async ({ page }, info) => {
  await enter(page);

  const navigationNames = ['Обзор', 'Курс', 'Путь обучения', 'Экзамены', 'Проекты', 'ИИ-наставник'];
  for (const name of navigationNames) {
    const control = page.getByRole('button', { name, exact: true });
    await expect(control).toBeVisible();
    await expect(control).toBeEnabled();
    await expect(control).toHaveAccessibleName(name);
  }
  const themeToggle = page.getByRole('button', { name: /Включить (тёмную|светлую) тему/ });
  await expect(themeToggle).toBeVisible();
  await expect(themeToggle).toBeEnabled();
  await expect(themeToggle).toHaveAccessibleName(/Включить (тёмную|светлую) тему/);

  // Reach the primary navigation using Tab only, then confirm the focused control
  // is exposed as keyboard-visible focus rather than merely being programmatically focused.
  await page.locator('body').click({ position: { x: 1, y: 1 } });
  let reachedCourse = false;
  for (let i = 0; i < 24; i++) {
    await page.keyboard.press('Tab');
    if (await page.getByRole('button', { name: 'Курс', exact: true }).evaluate(el => el === document.activeElement)) {
      reachedCourse = true;
      break;
    }
  }
  expect(reachedCourse, 'Tab navigation should reach the Course control').toBe(true);
  expect(await page.getByRole('button', { name: 'Курс', exact: true }).evaluate(el => el.matches(':focus-visible'))).toBe(true);

  await page.emulateMedia({ reducedMotion: 'reduce' });
  expect(await page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true);
  expect(await page.getByRole('button', { name: 'Курс', exact: true }).evaluate(el => getComputedStyle(el).transitionDuration)).toMatch(/^(0s|0ms)(,\s*(0s|0ms))*$/);

  for (const viewport of [{ width: 768, height: 1024, label: 'tablet-768' }, { width: 1920, height: 1080, label: 'wide-1920' }]) {
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    for (const name of navigationNames) {
      const bounds = await page.getByRole('button', { name, exact: true }).evaluate(el => {
        const rect = el.getBoundingClientRect();
        return { left: rect.left, right: rect.right, viewport: window.innerWidth };
      });
      expect(bounds.left >= 0 && bounds.right <= bounds.viewport,
        `${viewport.label}: navigation item "${name}" must not be clipped (${JSON.stringify(bounds)})`).toBe(true);
    }
    const overflow = await page.evaluate(() => ({width: document.documentElement.scrollWidth, viewport: window.innerWidth, elements: [...document.querySelectorAll('body *')].map(el => ({tag: el.tagName, className: typeof el.className === 'string' ? el.className : '', text: (el.textContent || '').trim().slice(0, 70), right: Math.round(el.getBoundingClientRect().right)})).filter(el => el.right > window.innerWidth + 1).slice(0, 12)}));
    expect.soft(overflow.width <= overflow.viewport, `${viewport.label} horizontal overflow: ${JSON.stringify(overflow)}`).toBe(true);
    const scan = await new AxeBuilder({ page }).analyze();
    expect.soft(scan.violations.map(({ id, nodes }) => ({ id, targets: nodes.map(node => node.target) })), `${viewport.label} axe violations`).toEqual([]);
    await page.screenshot({ path: `artifacts/${info.project.name}-m12-${viewport.label}-reduced-motion.png`, fullPage: true });
  }
});
