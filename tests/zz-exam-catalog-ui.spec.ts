import {test,expect} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import {execFileSync,spawn} from 'node:child_process';
import {mkdirSync,mkdtempSync,rmSync} from 'node:fs';
import {resolve,join} from 'node:path';
import {createServer} from 'node:net';

// This test owns a separate disposable API/DB. Seeded UI prerequisites never
// alter the main desktop/mobile suite or any real learner's installation.
test('new exam selection, versioned history and cancellation use the selected catalog entry',async({page},info)=>{
 mkdirSync('.data/e2e-exam-ui',{recursive:true});
 const folder=mkdtempSync(resolve('.data/e2e-exam-ui/run-'));
 const reservation=createServer();await new Promise<void>(done=>reservation.listen(0,'127.0.0.1',done));
 const address=reservation.address();if(!address||typeof address==='string')throw Error('test port reservation failed');const port=address.port;await new Promise<void>(done=>reservation.close(()=>done()));
 execFileSync('sh',['scripts/go.sh','build','-o','.tools/academy-ui-test-api','./cmd/api'],{timeout:30000});
 const database=join(folder,'academy.db'),base=`http://127.0.0.1:${port}`;
 const apiProcess=spawn(resolve('.tools/academy-ui-test-api'),{env:{...process.env,ACADEMY_ADDR:`127.0.0.1:${port}`,ACADEMY_DB:database,ACADEMY_WEB:resolve('web/dist')},stdio:'ignore'});
 try{
  await expect.poll(async()=>{try{return(await page.request.get(base+'/api/health',{timeout:1000})).ok()}catch{return false}}).toBe(true);
  await page.goto(base);await page.getByLabel('Твоё имя').fill('Каталог экзаменов');await page.getByRole('button',{name:'Создать профиль'}).click();await expect(page.getByRole('heading',{name:'Твоя практика начинается здесь.'})).toBeVisible();
  execFileSync('sh',['scripts/go.sh','run','scripts/seed-exam-ui.go'],{env:{...process.env,ACADEMY_E2E_SEED:'1',ACADEMY_E2E_DB:database},timeout:30000});
  await page.getByRole('button',{name:'Экзамены',exact:true}).click();
  const catalog=await(await page.request.get(base+'/api/exams')).json();expect(catalog).toHaveLength(3);
  for(const id of ['junior-core','juniorplus-api']){
   const exam=catalog.find((item:any)=>item.id===id);expect(exam.eligible).toBe(true);
   const card=page.locator('.exam-catalog-item').filter({has:page.getByRole('heading',{name:exam.title,exact:true})});
   await card.getByRole('button',{name:'Начать экзамен',exact:true}).click();await expect(page.getByRole('heading',{name:exam.title,exact:true})).toBeVisible();await expect(page.locator('.exam-question')).toHaveCount(20);await expect(page.locator('.exam-task')).toHaveCount(3);
   await page.locator('.exam-question input').first().check();await expect(page.getByText('Черновик сохранён',{exact:true})).toBeVisible();
   const scan=await new AxeBuilder({page}).analyze();expect(scan.violations.map(v=>({id:v.id,targets:v.nodes.map(n=>n.target)}))).toEqual([]);expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
   await page.screenshot({path:`artifacts/${info.project.name}-${id}-exam.png`,fullPage:true});
   page.once('dialog',dialog=>dialog.accept());await page.getByRole('button',{name:'Отменить попытку',exact:true}).click();await expect(page.getByRole('heading',{name:'Экзамены',exact:true})).toBeVisible();
   const history=await(await page.request.get(base+'/api/exams/attempts')).json();expect(history.some((item:any)=>item.exam_id===id&&item.status==='cancelled')).toBe(true);
  }
 }finally{
  apiProcess.kill('SIGTERM');await new Promise<void>(done=>{if(apiProcess.exitCode!==null){done();return}const timer=setTimeout(()=>apiProcess.kill('SIGKILL'),5000);apiProcess.once('exit',()=>{clearTimeout(timer);done()})});rmSync(folder,{recursive:true,force:true});
 }
});
