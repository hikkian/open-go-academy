import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test('advanced exam catalog explains artifact-bound human review and reviewer-key limits',async({page},info)=>{
  await page.route('**/api/exams',route=>route.fulfill({json:[{
    id:'senior-exam',version:'1',title:'Экзамен Senior',rank:'Senior',estimated_minutes:90,
    description:'Проверка навыков старшего уровня.',eligible:false,missing_prerequisites:['rank_evidence'],criteria:['Проверка проекта'],
    rank_evidence:{exam_id:'senior-exam',exam_version:'1',rank:'Senior',policy_version:1,eligible:false,requirements:[
      {requirement_id:'P5',kind:'project',status:'verified'},
      {requirement_id:'review:P5',kind:'manual_review',status:'missing',reason:'manual_review_quorum_missing'},
    ]},
  }]}));
  await page.goto('/');
  await expect.poll(async()=>await page.getByLabel('Твоё имя').isVisible()||await page.getByRole('heading',{name:'Твоя практика начинается здесь.'}).isVisible()).toBe(true);
  if(await page.getByLabel('Твоё имя').isVisible()){
    await page.getByLabel('Твоё имя').fill('Проверка ручного ревью');
    await page.getByRole('button',{name:'Создать профиль'}).click();
  }
  await page.getByRole('button',{name:'Экзамены',exact:true}).click();
  const card=page.locator('.exam-catalog-item');
  await expect(card.getByRole('heading',{name:'Экзамен Senior'})).toBeVisible();
  await card.getByText('Проектные и курсовые доказательства').click();
  await expect(card.getByText('ручная проверка проекта P5')).toBeVisible();
  await expect(card.getByText('не хватает решений двух разных разрешённых ключей проверяющих для точной версии проекта')).toBeVisible();
  await expect(card.getByText(/конкретной отправленной версии проекта и её прошедшей автоматической оценке/)).toBeVisible();
  await expect(card.getByText(/не подтверждает личность проверяющего или независимость проверок/)).toBeVisible();
  const scan=await new AxeBuilder({page}).analyze();
  expect(scan.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>n.target)}))).toEqual([]);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true);
  await page.screenshot({path:`artifacts/${info.project.name}-manual-review-prerequisite.png`,fullPage:true});
});
