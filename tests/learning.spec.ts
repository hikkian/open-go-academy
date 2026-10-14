import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
const lessonTitle='Что такое компьютер, программа и Go';
const valid='package main\n\nimport "fmt"\n\nfunc main() {\n fmt.Println("Go готов!")\n}\n';
async function enter(page:any){await page.goto('/');if(await page.getByLabel('Твоё имя').isVisible()){await page.getByLabel('Твоё имя').fill('Ада');await page.getByRole('button',{name:'Создать профиль'}).click()};await expect(page.getByRole('heading',{name:'Твоя практика начинается здесь.'})).toBeVisible();}
async function open(page:any){await page.getByRole('button',{name:new RegExp(lessonTitle)}).first().click();await expect(page.getByLabel('Go-код',{exact:true})).toBeVisible();}
async function code(page:any,source:string){const editor=page.getByLabel('Go-код',{exact:true});await editor.click();await editor.press('ControlOrMeta+a');await editor.fill(source);await expect(page.getByText('Сохранено',{exact:true})).toBeVisible();}
test('profile, roadmap, safe content, themes, drafts and responsive layout',async({page},info)=>{
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));await enter(page);
 await page.getByRole('button',{name:'Путь обучения',exact:true}).click();for(const rank of ['Zero','Junior','Junior+','Middle','Middle+','Senior'])await expect(page.getByRole('heading',{name:rank,exact:true})).toBeVisible();await page.getByRole('button',{name:new RegExp(lessonTitle)}).click();await expect(page.getByLabel('Go-код',{exact:true})).toBeVisible();
 await code(page,valid);await page.reload();await open(page);await expect(page.getByLabel('Go-код',{exact:true})).toContainText('Go готов!');
 await page.getByRole('button',{name:'Показать решение'}).click();await expect(page.locator('.solution-source')).toContainText('fmt.Println');await page.getByRole('button',{name:'Скрыть решение'}).click();await expect(page.locator('.solution-source')).toHaveCount(0);
 const light=await new AxeBuilder({page}).analyze();expect(light.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>({target:n.target,message:n.failureSummary}))}))).toEqual([]);
 await page.screenshot({path:`artifacts/${info.project.name}-light.png`,fullPage:true});
 await page.getByRole('button',{name:'Включить тёмную тему'}).click();await expect(page.locator('html')).toHaveAttribute('data-theme','dark');await page.reload();await open(page);await expect(page.locator('html')).toHaveAttribute('data-theme','dark');
 const dark=await new AxeBuilder({page}).analyze();expect(dark.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>({target:n.target,message:n.failureSummary}))}))).toEqual([]);await page.screenshot({path:`artifacts/${info.project.name}-dark.png`,fullPage:true});expect(errors).toEqual([]);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
 await page.getByRole('button',{name:'Включить светлую тему'}).click();
});
test('real Go errors, output, server grading, no AI and idempotent XP',async({page},info)=>{
 test.skip(process.env.ACADEMY_E2E_RUNNER!=='1','requires independently running confined local runner');test.skip(info.project.name!=='desktop','resource-intensive golden path only desktop; mobile reader/editor covered');
 await enter(page);await open(page);
 await code(page,'package main\nfunc main() { missing() }');await page.getByRole('button',{name:'Проверить',exact:true}).click();await expect(page.locator('.console-output')).toContainText('undefined: missing',{timeout:65000});
 await page.getByRole('button',{name:'Обзор',exact:true}).click();await expect(page.getByText('0 XP',{exact:true})).toBeVisible();await open(page);
 await code(page,valid);await page.getByRole('button',{name:'Запустить',exact:true}).click();await expect(page.locator('.console-output')).toContainText('Go готов!',{timeout:65000});
 for(let i=0;i<2;i++){await page.getByRole('button',{name:'Проверить',exact:true}).click();await expect(page.getByText('Задание принято. Прогресс обновлён.')).toBeVisible({timeout:65000})}
 await page.getByRole('button',{name:'Форматировать',exact:true}).click();await expect(page.getByText('Форматирование завершено.')).toBeVisible({timeout:65000});
 await page.getByRole('button',{name:'Обзор',exact:true}).click();await expect(page.getByText('10 XP',{exact:true})).toBeVisible();await page.reload();await expect(page.getByText('10 XP',{exact:true})).toBeVisible();await open(page);await expect(page.getByLabel('Go-код',{exact:true})).toContainText('Go готов!');await page.screenshot({path:'artifacts/golden-path.png',fullPage:true});
});
test('failed draft write has explicit recovery and preserves code',async({page},info)=>{
 await enter(page);await open(page);await expect(page.getByText('Сохранено',{exact:true})).toBeVisible();
 let failOnce=true;await page.route('**/api/drafts/hello',async route=>{if(route.request().method()==='PUT'&&failOnce){failOnce=false;await route.abort('failed')}else await route.continue()});
 const editor=page.getByLabel('Go-код',{exact:true});await editor.fill(valid+'// recovered draft '+info.project.name);await expect(page.getByText(/Не сохранено:/)).toBeVisible();await page.getByRole('button',{name:'Повторить сохранение'}).click();await expect(page.getByText('Сохранено',{exact:true})).toBeVisible();await page.unroute('**/api/drafts/hello');await page.reload();await open(page);await expect(editor).toContainText('// recovered draft '+info.project.name);
});
test('course search and server criteria/skills reflect only verified exercises',async({page},info)=>{
 await enter(page);await page.getByRole('button',{name:'Курс',exact:true}).click();await page.getByLabel('Найти уроки').fill('несуществующая тема');await expect(page.getByText('По этому запросу уроков не найдено.',{exact:true})).toBeVisible();await page.getByLabel('Найти уроки').fill('компьютер');await page.getByRole('button',{name:new RegExp(lessonTitle)}).click();await expect(page.getByLabel('Go-код',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'Путь обучения',exact:true}).click();await expect(page.getByRole('heading',{name:'Senior',exact:true})).toBeVisible();await page.getByText('Критерии уровня',{exact:true}).first().click();await expect(page.getByText('Пройти итоговую проверку Zero (ещё не реализована).',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'Навыки',exact:true}).click();await expect(page.getByRole('heading',{name:'Карта навыков'})).toBeVisible();const program=page.getByRole('row').filter({hasText:'Исходный код и запуск программы'});await expect(program).toBeVisible();
 if(process.env.ACADEMY_E2E_RUNNER==='1'){await expect(program).toContainText('2 / 4');await program.getByRole('button',{name:'Поздоровайся с компьютером'}).click();await expect(page.getByLabel('Go-код',{exact:true})).toContainText('Go готов!');await page.getByRole('button',{name:'Навыки',exact:true}).click()}
 await expect(page.getByRole('row').filter({hasText:'Архитектура и компромиссы'})).toContainText('0 / 4');const scan=await new AxeBuilder({page}).analyze();expect(scan.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>n.target)}))).toEqual([]);await page.screenshot({path:`artifacts/${info.project.name}-skills.png`,fullPage:true});
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
});
