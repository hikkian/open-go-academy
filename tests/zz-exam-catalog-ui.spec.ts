import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import {execFileSync} from 'node:child_process';
// This test runs last because it seeds explicit, disposable UI prerequisites.
// Native grading and rank/credential integrity have a separate real-runner test.
test('new exam selection, versioned history and cancellation use the selected catalog entry',async({page},info)=>{
 await page.goto('/');if(await page.getByLabel('Твоё имя').isVisible()){await page.getByLabel('Твоё имя').fill('Каталог экзаменов');await page.getByRole('button',{name:'Создать профиль'}).click()}
 execFileSync('sh',['scripts/go.sh','run','scripts/seed-exam-ui.go'],{env:{...process.env,ACADEMY_E2E_SEED:'1'},timeout:30000});
 await page.getByRole('button',{name:'Экзамены',exact:true}).click();
 const catalog=await(await page.request.get('/api/exams')).json();expect(catalog).toHaveLength(3);
 for(const id of ['junior-core','juniorplus-api']){
  const exam=catalog.find((item:any)=>item.id===id);expect(exam.eligible).toBe(true);
  const card=page.locator('.exam-catalog-item').filter({has:page.getByRole('heading',{name:exam.title,exact:true})});
  await card.getByRole('button',{name:'Начать экзамен',exact:true}).click();await expect(page.getByRole('heading',{name:exam.title,exact:true})).toBeVisible();await expect(page.locator('.exam-question')).toHaveCount(20);await expect(page.locator('.exam-task')).toHaveCount(3);
  await page.locator('.exam-question input').first().check();await expect(page.getByText('Черновик сохранён',{exact:true})).toBeVisible();
  const scan=await new AxeBuilder({page}).analyze();expect(scan.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>n.target)}))).toEqual([]);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
  await page.screenshot({path:`artifacts/${info.project.name}-${id}-exam.png`,fullPage:true});
  await page.getByRole('button',{name:'Отменить попытку',exact:true}).click();await expect(page.getByRole('heading',{name:'Экзамены',exact:true})).toBeVisible();
  const history=await(await page.request.get('/api/exams/attempts')).json();expect(history.some((item:any)=>item.exam_id===id&&item.status==='cancelled')).toBe(true);
 }
});
