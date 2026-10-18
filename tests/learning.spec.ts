import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
const lessonTitle='Что такое компьютер, программа и Go';
const valid='package main\n\nimport "fmt"\n\nfunc main() {\n fmt.Println("Go готов!")\n}\n';
async function enter(page:any){await page.goto('/');await expect.poll(async()=>await page.getByLabel('Твоё имя').isVisible()||await page.getByRole('heading',{name:'Твоя практика начинается здесь.'}).isVisible()).toBe(true);if(await page.getByLabel('Твоё имя').isVisible()){await page.getByLabel('Твоё имя').fill('Ада');await page.getByRole('button',{name:'Создать профиль'}).click()};await expect(page.getByRole('heading',{name:'Твоя практика начинается здесь.'})).toBeVisible();}
async function open(page:any){await page.getByRole('button',{name:new RegExp(lessonTitle)}).first().click();await expect(page.getByLabel('Go-код',{exact:true})).toBeVisible();}
async function code(page:any,source:string){const editor=page.getByLabel('Go-код',{exact:true});await editor.click();await editor.press('ControlOrMeta+a');await editor.fill(source);await expect(page.getByText('Сохранено',{exact:true})).toBeVisible();}
test('expanded local course publishes and renders Middle through Senior lessons',async({page},info)=>{
 await enter(page);
 const response=await page.request.get('/api/lessons');expect(response.ok()).toBeTruthy();
 const lessons=await response.json();expect(lessons).toHaveLength(128);
 const expected=[['Middle',40],['Middle+',24],['Senior',24]];
 await page.getByRole('button',{name:'Путь обучения',exact:true}).click();
 for(const [rank,count] of expected){
  const level=page.locator('.path-level').filter({has:page.getByRole('heading',{name:rank,exact:true})});
  await expect(level).toContainText(`${count} уроков доступно`);
 }
 const middle=page.locator('.path-level').filter({has:page.getByRole('heading',{name:'Middle',exact:true})});
 await middle.getByRole('button',{name:/Параллельная работа и конкурентность/}).click();
 await expect(page.getByLabel('Go-код',{exact:true})).toBeVisible();
 await expect(page.getByText(/Практический мост: от модели к реальному параллельному worker-у/)).toBeVisible();
 await page.screenshot({path:`artifacts/${info.project.name}-middle-course.png`,fullPage:true});
 await page.getByRole('button',{name:'Путь обучения',exact:true}).click();
 const senior=page.locator('.path-level').filter({has:page.getByRole('heading',{name:'Senior',exact:true})});
 const seniorLesson=lessons.find((lesson:any)=>lesson.id==='senior-23');expect(seniorLesson).toBeTruthy();
 await senior.getByRole('button',{name:new RegExp(seniorLesson.title.replace(/[.*+?^${}()|[\]\\]/g,'\\$&'))}).click();
 await expect(page.getByLabel('Go-код',{exact:true})).toBeVisible();
 await expect(page.getByText(/P5 defense|независимую проверку/)).toBeVisible();
 const axe=await new AxeBuilder({page}).analyze();expect(axe.violations.map(v=>v.id)).toEqual([]);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
 await page.screenshot({path:`artifacts/${info.project.name}-senior-course.png`,fullPage:true});
});
test('profile, roadmap, safe content, themes, drafts and responsive layout',async({page},info)=>{
 const errors:string[]=[];page.on('pageerror',e=>errors.push(e.message));await enter(page);if(await page.locator('html').getAttribute('data-theme')==='dark')await page.getByRole('button',{name:'Включить светлую тему'}).click();
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
 await enter(page);const initial=await(await page.request.get('/api/progress')).json();const expectedXP=initial.xp+(initial.completed.includes('hello')?0:10);await open(page);
 await code(page,'package main\nfunc main() { missing() }');await page.getByRole('button',{name:'Проверить',exact:true}).click();await expect(page.locator('.console-output')).toContainText('undefined: missing',{timeout:65000});
 await page.getByRole('button',{name:'Обзор',exact:true}).click();await expect(page.getByText(`${initial.xp} XP`,{exact:true})).toBeVisible();await open(page);
 await code(page,valid);await page.getByRole('button',{name:'Запустить',exact:true}).click();await expect(page.locator('.console-output')).toContainText('Go готов!',{timeout:65000});
 for(let i=0;i<2;i++){await page.getByRole('button',{name:'Проверить',exact:true}).click();await expect(page.getByText('Задание принято. Прогресс обновлён.')).toBeVisible({timeout:65000})}
 await page.getByRole('button',{name:'Форматировать',exact:true}).click();await expect(page.getByText('Форматирование завершено.')).toBeVisible({timeout:65000});
 await page.getByRole('button',{name:'Обзор',exact:true}).click();await expect(page.getByText(`${expectedXP} XP`,{exact:true})).toBeVisible();await page.reload();await expect(page.getByText(`${expectedXP} XP`,{exact:true})).toBeVisible();await open(page);await expect(page.getByLabel('Go-код',{exact:true})).toContainText('Go готов!');await page.screenshot({path:'artifacts/golden-path.png',fullPage:true});
});
test('failed draft write has explicit recovery and preserves code',async({page},info)=>{
 await enter(page);await open(page);await expect(page.getByText('Сохранено',{exact:true})).toBeVisible();
 let failOnce=true;await page.route('**/api/drafts/hello',async route=>{if(route.request().method()==='PUT'&&failOnce){failOnce=false;await route.abort('failed')}else await route.continue()});
 const editor=page.getByLabel('Go-код',{exact:true});await editor.fill(valid+'// recovered draft '+info.project.name);await expect(page.getByText(/Не сохранено:/)).toBeVisible();await page.getByRole('button',{name:'Повторить сохранение'}).click();await expect(page.getByText('Сохранено',{exact:true})).toBeVisible();await page.unroute('**/api/drafts/hello');await page.reload();await open(page);await expect(editor).toContainText('// recovered draft '+info.project.name);
});
test('course search and server criteria/skills reflect only verified exercises',async({page},info)=>{
 await enter(page);const learning=await(await page.request.get('/api/learning')).json();const progress=await(await page.request.get('/api/progress')).json();await page.getByRole('button',{name:'Курс',exact:true}).click();await page.getByLabel('Найти уроки').fill('несуществующая тема');await expect(page.getByText('По этому запросу уроков не найдено.',{exact:true})).toBeVisible();await page.getByLabel('Найти уроки').fill('компьютер');await page.getByRole('button',{name:new RegExp(lessonTitle)}).click();await expect(page.getByLabel('Go-код',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'Путь обучения',exact:true}).click();await expect(page.getByRole('heading',{name:'Senior',exact:true})).toBeVisible();await page.getByText('Критерии уровня',{exact:true}).first().click();await expect(page.getByText('Пройти экзамен Zero: все вопросы и задания по версии критериев.',{exact:true})).toBeVisible();const rankTrust=page.locator('.rank-trust-note');await expect(rankTrust).toContainText(learning.current_rank);await expect(rankTrust).toContainText(learning.verified?'подтверждён по критериям курса.':'пока не подтверждён как достигнутый ранг.');await expect(rankTrust).toContainText('сами по себе не подтверждают ранг');
 await page.getByRole('button',{name:'Навыки',exact:true}).click();await expect(page.getByRole('heading',{name:'Карта навыков'})).toBeVisible();const skillTrust=page.locator('.rank-trust-note');await expect(skillTrust).toContainText(learning.current_rank);await expect(skillTrust).toContainText(learning.verified?'подтверждён по критериям курса.':'не подтверждён как достигнутый ранг.');await expect(skillTrust).toContainText('XP не заменяет проверку ранга.');
 const lessons=await(await page.request.get('/api/lessons')).json();for(const skill of learning.skills){const row=page.getByRole('row').filter({hasText:skill.title});await expect(row).toBeVisible();await expect(row).toContainText(`${skill.level} / 4`);const evidenceCell=row.getByRole('cell').nth(2);await expect(evidenceCell.locator('.evidence-list li')).toHaveCount(skill.evidence.length);for(const id of skill.evidence){const exercise=lessons.flatMap((lesson:any)=>lesson.exercises||[]).find((candidate:any)=>candidate.id===id);const label=exercise?.title||(skill.id==='communication'&&id.startsWith('sha256:')?'P5: итоговое ручное ревью Senior':id);await expect(evidenceCell).toContainText(label)}}
 const program=page.getByRole('row').filter({hasText:'Исходный код и запуск программы'});await expect(program).toBeVisible();if(process.env.ACADEMY_E2E_RUNNER==='1'){const expected=learning.skills.find((skill:any)=>skill.id==='program');await expect(program).toContainText(`${expected.level} / 4`);await program.getByRole('button',{name:'Поздоровайся с компьютером'}).click();await expect(page.getByLabel('Go-код',{exact:true})).toContainText('Go готов!');await page.getByRole('button',{name:'Навыки',exact:true}).click()}
 const after=await(await page.request.get('/api/learning')).json();expect(after.current_rank).toBe(learning.current_rank);expect(after.verified).toBe(learning.verified);expect((await(await page.request.get('/api/progress')).json()).xp).toBe(progress.xp);const scan=await new AxeBuilder({page}).analyze();expect(scan.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>n.target)}))).toEqual([]);await page.screenshot({path:`artifacts/${info.project.name}-skills.png`,fullPage:true});
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
});

test('verified Senior review evidence has a clear label without exposing its token',async({page},info)=>{
 await enter(page);
 const snapshot=await(await page.request.get('/api/learning')).json();
 const communication=snapshot.skills.find((skill:any)=>skill.id==='communication');
 expect(communication).toBeTruthy();
 communication.level=4;
 communication.evidence=['sha256:'+'a'.repeat(64)];
 await page.route('**/api/learning',route=>route.fulfill({json:snapshot}));
 await page.reload();
 await page.getByRole('button',{name:'Навыки',exact:true}).click();
 const row=page.getByRole('row').filter({hasText:'Ревью и инженерное объяснение'});
 await expect(row).toContainText('4 / 4');
 await expect(row).toContainText('P5: итоговое ручное ревью Senior');
 await expect(row).not.toContainText('sha256:');
 const table=page.locator('.skill-table-wrap');
 await page.screenshot({path:`artifacts/${info.project.name}-senior-communication-evidence-left.png`,fullPage:true});
 if(info.project.name==='mobile'){
  await expect(table.locator('.skill-table-hint')).toBeVisible();
  const dimensions=await table.evaluate((element:any)=>({client:element.clientWidth,scroll:element.scrollWidth}));
  expect(dimensions.scroll).toBeGreaterThan(dimensions.client);
  await table.evaluate((element:any)=>{element.scrollLeft=element.scrollWidth});
  expect(await table.evaluate((element:any)=>element.scrollLeft)).toBeGreaterThan(0);
 }
 const axe=await new AxeBuilder({page}).analyze();
 expect(axe.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>n.target)}))).toEqual([]);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
 await page.screenshot({path:`artifacts/${info.project.name}-senior-communication-evidence-right.png`,fullPage:true});
});
