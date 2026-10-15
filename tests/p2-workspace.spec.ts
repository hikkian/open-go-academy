import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import {readFileSync} from 'node:fs';
test('P2 nested workspace, locked policy, snapshots and export',async({page},info)=>{
 await page.goto('/');
 if(await page.getByLabel('Твоё имя').isVisible()){await page.getByLabel('Твоё имя').fill('Студент LinkHub');await page.getByRole('button',{name:'Создать профиль'}).click()}
 const open=async()=>{await page.getByRole('button',{name:'Проекты',exact:true}).click();await page.getByRole('button',{name:/LinkHub API/}).click();await expect(page.getByRole('heading',{name:'LinkHub API',exact:true})).toBeVisible()};
 const filename=`internal/browser_${info.project.name}.go`;
 await open();await expect(page.getByText(/До 48 файлов/)).toBeVisible();
 await page.getByRole('button',{name:'go.mod',exact:true}).click();await expect(page.getByLabel('Содержимое файла go.mod')).toHaveAttribute('readonly','');
 await expect(page.getByRole('button',{name:'Удалить выбранный файл',exact:true})).toBeDisabled();
 await page.getByLabel('Добавить файл').fill(filename);await page.getByRole('button',{name:'Добавить',exact:true}).click();
 const source='package internal\n\n// Saved by automatic browser validation.\n';await page.getByLabel('Go-код',{exact:true}).fill(source);await expect(page.getByText('Файлы сохранены.',{exact:true})).toBeVisible();
 await page.reload();await open();await page.getByRole('button',{name:filename,exact:true}).click();await expect(page.getByLabel('Go-код',{exact:true})).toContainText('Saved by automatic');
 const pending=page.waitForEvent('download');await page.getByRole('button',{name:'Скачать ZIP',exact:true}).click();const file=await pending;expect(file.suggestedFilename()).toBe('p2.zip');const path=await file.path();const bytes=readFileSync(path!);expect(bytes.toString()).toContain(filename);expect(bytes.toString()).toContain('compose.yaml');
 const scan=await new AxeBuilder({page}).analyze();expect(scan.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>n.target)}))).toEqual([]);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);await page.screenshot({path:`artifacts/${info.project.name}-p2-workspace.png`,fullPage:true});
});

test('P2 real isolated API check persists evidence without assigning rank',async({page},info)=>{
 test.skip(process.env.ACADEMY_E2E_PROJECT_RUNNER!=='1','separate confined PostgreSQL project worker required');test.skip(info.project.name!=='desktop','mobile workspace separately covered');test.setTimeout(240000);
 await page.goto('/');if(await page.getByLabel('Твоё имя').isVisible()){await page.getByLabel('Твоё имя').fill('P2 native');await page.getByRole('button',{name:'Создать профиль'}).click()}
 const {readdirSync}=await import('node:fs');const {join,relative}=await import('node:path');
 const root='content/projects/p2/reference', files:Record<string,string>={};
 const collect=(dir:string)=>{for(const item of readdirSync(dir,{withFileTypes:true})){const path=join(dir,item.name);if(item.isDirectory())collect(path);else if(/\.(go|sql)$/.test(item.name)||['go.mod','go.sum'].includes(item.name))files[relative(root,path)]=readFileSync(path,'utf8')}};collect(root);files['README.md']='# Trusted reference\n\nAutomated isolated P2 acceptance test.\n';
 const before=await(await page.request.get('/api/learning')).json();const csrf=(await(await page.request.get('/api/profile')).json()).csrf_token;
 const save=await page.request.put('/api/projects/p2/files',{headers:{'X-CSRF-Token':csrf},data:{files}});expect(save.ok()).toBe(true);
 await page.getByRole('button',{name:'Проекты',exact:true}).click();await page.getByRole('button',{name:/LinkHub API/}).click();
 await page.getByRole('button',{name:'Проверить проект',exact:true}).click();await expect(page.locator('.project-output')).toContainText('Автоматические проверки пройдены.',{timeout:220000});
 await expect(page.locator('.project-output')).toContainText('persistence.link');await expect(page.locator('.project-output')).toContainText('security.wrong_csrf');
 const progress=await(await page.request.get('/api/progress')).json();expect(progress.completed).toContain('p2:automatic');const after=await(await page.request.get('/api/learning')).json();expect(after.current_rank).toBe(before.current_rank);expect(after.verified).toBe(before.verified);
 await page.screenshot({path:'artifacts/p2-native-passed.png',fullPage:true});
});
