import { chromium } from 'playwright-core';

const BASE = 'http://localhost:5173';
const routes = [
  '/sales', '/sales/history', '/sales/held', '/sales/returns',
  '/shifts', '/branches', '/mpesa', '/customers', '/employees',
  '/catalog', '/inventory', '/pricing', '/receipts/designer',
  '/terminals', '/audit', '/roles', '/modules', '/settings', '/account',
];

const login = await fetch(`${BASE}/api/v1/t/auth/login`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ tenant: 'demo', identity: 'owner@demo.local', password: 'ChangeMe123!' }),
}).then((r) => r.json());

const browser = await chromium.launch({ channel: 'chrome', headless: true });
const page = await browser.newPage();
let failures = [];
const collect = () => {
  page.on('pageerror', (e) => failures.push(`[${page.url()}] PAGEERROR: ${e.message}`));
  page.on('response', (r) => {
    if (r.status() >= 500) failures.push(`[${page.url()}] HTTP ${r.status()} ${r.url()}`);
  });
};
collect();

await page.goto(`${BASE}/login`);
await page.evaluate(([t]) => {
  localStorage.setItem('mauzo_web_token', t);
  localStorage.setItem('mauzo_tenant_slug', 'demo');
}, [login.data.token]);

for (const route of routes) {
  failures = [];
  await page.goto(`${BASE}${route}`, { waitUntil: 'networkidle', timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(1000);
  const body = await page.evaluate(() => document.body.innerText);
  const bad = /500 Internal Error|Cannot read|undefined is not/i.test(body);
  console.log(`${route}: ${bad ? 'ERROR-PAGE' : 'ok'}`);
  for (const f of failures) console.log(`   ${f}`);
}

// Open a sale row — the detail drawer was the returns:null crash.
failures = [];
await page.goto(`${BASE}/sales`, { waitUntil: 'networkidle' }).catch(() => {});
await page.waitForTimeout(1200);
const row = page.locator('tbody tr').first();
if (await row.count()) {
  await row.click();
  await page.waitForTimeout(1500);
  const body = await page.evaluate(() => document.body.innerText);
  console.log(`/sales detail drawer: ${/500 Internal Error|Cannot read/i.test(body) ? 'ERROR-PAGE' : 'ok'}`);
  for (const f of failures) console.log(`   ${f}`);
} else {
  console.log('/sales: no rows to click');
}
await browser.close();
