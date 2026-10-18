import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import {readFileSync,readdirSync} from 'node:fs';
import {join,relative} from 'node:path';

async function enter(page:any){
 await page.goto('/');
 if(await page.getByLabel('Твоё имя').isVisible()){
  await page.getByLabel('Твоё имя').fill('P3 браузерная проверка');
  await page.getByRole('button',{name:'Создать профиль'}).click();
 }
 await page.getByRole('button',{name:'Проекты',exact:true}).click();
 await page.getByRole('button',{name:/TaskFlow backend/}).click();
 await expect(page.getByRole('heading',{name:'TaskFlow backend',exact:true})).toBeVisible();
}

function referenceFiles(){
 const root='content/projects/p3/reference';
 const files:Record<string,string>={};
 const collect=(dir:string)=>{for(const item of readdirSync(dir,{withFileTypes:true})){
  const path=join(dir,item.name);
  if(item.isDirectory())collect(path);
  else if(/\.(go|sql)$/.test(item.name)||['go.mod','go.sum'].includes(item.name)) files[relative(root,path)]=readFileSync(path,'utf8');
 }};
 collect(root);
 files['README.md']=readFileSync('content/projects/p3/README.md','utf8');
 // These export-only examples belong to the complete learner artifact but
 // must be stripped before the isolated runner receives executable sources.
 files['Dockerfile']=readFileSync('content/projects/p3/reference/Dockerfile','utf8');
 files['compose.yaml']=readFileSync('content/projects/p3/compose.yaml','utf8');
 files['.env.example']=readFileSync('content/projects/p3/.env.example','utf8');
 return files;
}

test('P3 catalog and workspace are responsive, accessible, and capability-gated',async({page},info)=>{
 await page.goto('/');
 if(await page.getByLabel('Твоё имя').isVisible()){
  await page.getByLabel('Твоё имя').fill('P3 capability check');
  await page.getByRole('button',{name:'Создать профиль'}).click();
 }
 const catalog=await page.request.get('/api/projects');
 expect(catalog.ok(),`GET /api/projects returned ${catalog.status()}: ${await catalog.text()}`).toBe(true);
 const projects=await catalog.json();
 const p3Cards=projects.filter((project:any)=>project.id==='p3');
 expect(p3Cards,process.env.ACADEMY_E2E_P3_RUNNER==='1'
  ? 'P3 TaskFlow checker is enabled in the isolated combined runner; the API must publish one P3 card.'
  : 'P3 must remain hidden when no TaskFlow-capable checker is available.').toHaveLength(process.env.ACADEMY_E2E_P3_RUNNER==='1'?1:0);
 if(process.env.ACADEMY_E2E_P3_RUNNER!=='1') return;
 if(p3Cards.length!==1) return; // The assertion above reports the capability-propagation failure without a long locator timeout.
 await enter(page);
 const details=await page.request.get('/api/projects/p3');
 expect((await details.json()).execution_available).toBe(true);
 await page.getByRole('button',{name:'go.mod',exact:true}).click();
 await expect(page.getByLabel('Содержимое файла go.mod')).toHaveAttribute('readonly','');
 await expect(page.getByRole('button',{name:'Удалить выбранный файл',exact:true})).toBeDisabled();
 await expect(page.getByLabel('Аргументы запуска (JSON-массив строк)')).toHaveCount(0);
 for(const label of ['Запустить','Запустить тесты','Проверить проект']) await expect(page.getByRole('button',{name:label,exact:true})).toBeEnabled();
 const scan=await new AxeBuilder({page}).analyze();
 expect(scan.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>n.target)}))).toEqual([]);
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true);
 await page.screenshot({path:`artifacts/${info.project.name}-p3-workspace.png`,fullPage:true});
 if(info.project.name==='mobile'){
  const profile=await(await page.request.get('/api/profile')).json();
  const save=await page.request.put('/api/projects/p3/files',{headers:{'X-CSRF-Token':profile.csrf_token},data:{files:referenceFiles()}});
  expect(save.ok()).toBe(true);
  await page.reload();
  await enter(page);
  let intercepted=0;
  const labels=['Архитектура и обоснование компромиссов','Качество кода и педагогическая ясность','Эксплуатационный разбор failure и security limits'];
  const fakeEvidence={submission_id:'mobile-submission-123',evaluation_id:'mobile-evaluation-456',project_version:'p3-taskflow-v1',criteria_hash:`sha256:${'1'.repeat(64)}`,artifact_hash:`sha256:${'2'.repeat(64)}`,executed_source_hash:`sha256:${'3'.repeat(64)}`,checker_version:'taskflow-v1',checker_digest:`sha256:${'4'.repeat(64)}`,runtime_policy_hash:`sha256:${'5'.repeat(64)}`,rubric_version:'taskflow-automatic-v1',rubric_hash:`sha256:${'6'.repeat(64)}`,evaluation_status:'automatic_pass',exit_code:0,checks_hash:`sha256:${'7'.repeat(64)}`};
  await page.route('**/api/projects/p3/submit',async route=>{
   if(route.request().method()!=='POST'){await route.continue();return}
   intercepted++;
   await route.fulfill({status:200,contentType:'application/json',body:JSON.stringify({status:'automatic_pass',exit_code:0,passed:true,checks:Array.from({length:45},(_,index)=>({name:`mobile.acceptance.${index+1}`,passed:true})),manual_review_required:true,manual_review_labels:labels,project_evidence:fakeEvidence,rank_awarded:false})});
  });
  await page.getByRole('button',{name:'Проверить проект',exact:true}).click();
  await expect(page.locator('.project-output')).toContainText('Автоматические проверки пройдены.');
  expect(intercepted).toBe(1);
  await expect(page.locator('.project-check')).toHaveCount(45);
  const savedEvidence=page.locator('details.project-evidence-details');
  await expect(savedEvidence.locator('summary')).toHaveText('Сохранённое доказательство этой версии проекта');
  await expect(savedEvidence).not.toHaveAttribute('open','');
  await savedEvidence.locator('summary').click();
  await expect(savedEvidence).toContainText(fakeEvidence.submission_id);
  await expect(savedEvidence).toContainText(fakeEvidence.evaluation_id);
  await expect(savedEvidence).toContainText('автоматическое evidence не подтверждает ручное ревью и не присваивает ранг');
  await expect(page.locator('.project-output')).toContainText('не присваивает учебный ранг');
  for(const label of labels) await expect(page.locator('.project-output')).toContainText(label);
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=document.documentElement.clientWidth)).toBe(true);
  const resultScan=await new AxeBuilder({page}).analyze();
  expect(resultScan.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>n.target)}))).toEqual([]);
  await page.screenshot({path:'artifacts/mobile-p3-submit-evidence.png',fullPage:true});
 }
});

test('P3 real Run, Test, Submit, reload persistence and snapshot restore',async({page},info)=>{
 test.skip(process.env.ACADEMY_E2E_P3_RUNNER!=='1','requires the isolated combined P2/P3 runner profile');
 test.skip(info.project.name!=='desktop','runtime acceptance runs once on desktop; responsive workspace covered above');
 test.setTimeout(420000);
 await page.goto('/');
 if(await page.getByLabel('Твоё имя').isVisible()){
  await page.getByLabel('Твоё имя').fill('P3 acceptance learner');
  await page.getByRole('button',{name:'Создать профиль'}).click();
 }
 const catalog=await(await page.request.get('/api/projects')).json();
 if(!catalog.some((project:any)=>project.id==='p3')) test.skip(true,'P3 capability is not surfaced by the API; the catalog acceptance test records this failure.');
 const before=await(await page.request.get('/api/learning')).json();
 const profile=await(await page.request.get('/api/profile')).json();
 const acceptedFiles=referenceFiles();
 const save=await page.request.put('/api/projects/p3/files',{headers:{'X-CSRF-Token':profile.csrf_token},data:{files:acceptedFiles}});
 expect(save.ok()).toBe(true);
 await enter(page);
 await expect(page.getByRole('button',{name:'go.mod',exact:true})).toBeVisible();
 await page.getByRole('button',{name:'go.mod',exact:true}).click();
 await expect(page.getByLabel('Содержимое файла go.mod')).toHaveAttribute('readonly','');
 await expect(page.getByRole('button',{name:'Удалить выбранный файл',exact:true})).toBeDisabled();
 await expect(page.getByLabel('Аргументы запуска (JSON-массив строк)')).toHaveCount(0);

 // TaskFlow has a fixed liveness contract and intentionally exposes no arbitrary run arguments.
 await page.getByRole('button',{name:'Запустить',exact:true}).click();
 await expect(page.locator('.project-output')).toContainText('Состояние: ok; код завершения: 0',{timeout:120000});
 await expect(page.locator('.project-output')).toContainText('Сервер запущен в изолированной среде.');

 await page.getByRole('button',{name:'Запустить тесты',exact:true}).click();
 await expect(page.locator('.project-output')).toContainText('Состояние: ok; код завершения: 0',{timeout:120000});
 await expect(page.locator('.project-output')).toContainText('Пользовательские Go-тесты выполнены');

 const submitResponsePromise=page.waitForResponse(response=>response.url().endsWith('/api/projects/p3/submit')&&response.request().method()==='POST');
 await page.getByRole('button',{name:'Проверить проект',exact:true}).click();
 await expect(page.locator('.project-output')).toContainText('Автоматические проверки пройдены.',{timeout:240000});
 const submitted=await(await submitResponsePromise).json();
 expect(submitted.status).toBe('automatic_pass');
 expect(submitted.rank_awarded).toBe(false);
 expect(submitted.checks).toHaveLength(45);
 expect(submitted.checks.every((check:any)=>check.passed)).toBe(true);
 const evidence=submitted.project_evidence;
 expect(evidence).toBeTruthy();
 for(const field of ['submission_id','evaluation_id','project_version','criteria_hash','artifact_hash','executed_source_hash','checker_version','checker_digest','runtime_policy_hash','rubric_version','rubric_hash','evaluation_status','checks_hash']) {
  expect(typeof evidence[field],`project_evidence.${field} is required`).toBe('string');
  expect(evidence[field].length,`project_evidence.${field} must not be empty`).toBeGreaterThan(0);
 }
 expect(evidence.evaluation_status).toBe('automatic_pass');
 expect(evidence.exit_code).toBe(0);
 for(const field of ['criteria_hash','artifact_hash','executed_source_hash','checker_digest','runtime_policy_hash','rubric_hash']) {
  expect(evidence[field],`project_evidence.${field} must be a SHA-256 digest`).toMatch(/^sha256:[0-9a-f]{64}$/);
 }
 expect(evidence.checks_hash).toMatch(/^sha256:[0-9a-f]{64}$/);
 expect(evidence.artifact_hash).not.toBe(evidence.executed_source_hash);
 await expect(page.locator('.project-check')).toHaveCount(45);
 await expect(page.locator('.project-output')).toContainText('не присваивает учебный ранг');
 for(const label of ['Архитектура и обоснование компромиссов','Качество кода и педагогическая ясность','Эксплуатационный разбор failure и security limits']) {
  await expect(page.locator('.project-output')).toContainText(label);
 }
 const savedEvidence=page.locator('details.project-evidence-details');
 await expect(savedEvidence).toHaveCount(1);
 await expect(savedEvidence.locator('summary')).toHaveText('Сохранённое доказательство этой версии проекта');
 await expect(savedEvidence).not.toHaveAttribute('open','');
 await savedEvidence.locator('summary').click();
 await expect(savedEvidence).toContainText(evidence.submission_id);
 await expect(savedEvidence).toContainText(evidence.evaluation_id);
 await expect(savedEvidence).toContainText('автоматическое evidence не подтверждает ручное ревью и не присваивает ранг');
 await page.screenshot({path:'artifacts/p3-automatic-passed.png',fullPage:true});
 const retryResponse=await page.request.post('/api/projects/p3/submit',{headers:{'X-CSRF-Token':profile.csrf_token},data:{files:acceptedFiles},timeout:240000});
 expect(retryResponse.ok(),`Identical P3 retry returned ${retryResponse.status()}: ${await retryResponse.text()}`).toBe(true);
 const retried=await retryResponse.json();
 expect(retried.rank_awarded).toBe(false);
 expect(retried.project_evidence).toEqual(evidence);
 expect(retried.project_evidence.submission_id).toBe(evidence.submission_id);
 expect(retried.project_evidence.evaluation_id).toBe(evidence.evaluation_id);
 const after=await(await page.request.get('/api/learning')).json();
 expect(after.current_rank).toBe(before.current_rank);
 expect(after.verified).toBe(before.verified);

 // Exercise product autosave, durable reload and UI snapshot restore.
 await page.getByRole('button',{name:'README.md',exact:true}).click();
 const readme=page.getByLabel('Содержимое файла README.md');
 await readme.fill('# TaskFlow acceptance\n\nVersion A persisted by Playwright.');
 await expect(page.getByText('Файлы сохранены.',{exact:true})).toBeVisible({timeout:15000});
 const snapshots=await(await page.request.get('/api/projects/p3/snapshots')).json();
 expect(snapshots.length).toBeGreaterThanOrEqual(1);
 await readme.fill('# TaskFlow acceptance\n\nVersion B should be rolled back.');
 await expect(page.getByText('Файлы сохранены.',{exact:true})).toBeVisible({timeout:15000});
 await page.reload();
 await enter(page);
 await page.getByRole('button',{name:'README.md',exact:true}).click();
 await expect(page.getByLabel('Содержимое файла README.md')).toContainText('Version B should be rolled back.');
 const history=page.locator('.snapshot-row');
 await expect(history).toHaveCount(3);
 await history.nth(1).getByRole('button',{name:'Восстановить'}).click();
 await expect(page.getByText(/Версия восстановлена/)).toBeVisible();
 await page.getByRole('button',{name:'README.md',exact:true}).click();
 await expect(page.getByLabel('Содержимое файла README.md')).toContainText('Version A persisted by Playwright.');
 const finalFiles=await(await page.request.get('/api/projects/p3/files')).json();
 expect(finalFiles.files['README.md']).toContain('Version A persisted by Playwright.');
 await page.screenshot({path:'artifacts/p3-snapshot-restored.png',fullPage:true});
});
