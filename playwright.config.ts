import { defineConfig } from '@playwright/test';
export default defineConfig({
 testDir:'./tests',timeout:240000,expect:{timeout:10000},workers:1,fullyParallel:false,
 outputDir:'artifacts/playwright',reporter:[['list'],['html',{outputFolder:'artifacts/playwright-report',open:'never'}]],
 use:{baseURL:'http://127.0.0.1:8081',trace:'retain-on-failure',screenshot:'only-on-failure'},
 projects:[{name:'desktop',use:{viewport:{width:1440,height:1000}}},{name:'mobile',use:{viewport:{width:390,height:844},isMobile:true}}],
 webServer:{command:'sh scripts/e2e-server.sh',url:'http://127.0.0.1:8081/api/health',reuseExistingServer:false,timeout:30000},
});
