import { chromium } from "playwright-core";

const baseUrl = process.env.FRONTEND_BASE_URL || "http://127.0.0.1:5173";
const browserExecutablePath =
  process.env.BROWSER_EXECUTABLE_PATH ||
  "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe";

const browser = await chromium.launch({
  executablePath: browserExecutablePath,
  headless: true,
});

const page = await browser.newPage();
page.setDefaultTimeout(10000);
const results = [];

async function record(label, fn) {
  try {
    const value = await fn();
    results.push({ label, ok: true, value });
  } catch (error) {
    results.push({ label, ok: false, error: String(error) });
  }
}

await record("login_page_accessible", async () => {
  await page.goto(`${baseUrl}/login`);
  await page.locator("text=Proman 后台登录").waitFor();
  return {
    url: page.url(),
    titleVisible: await page.locator("text=Proman 后台登录").isVisible(),
  };
});

await record("guard_redirects_unauthenticated_projects", async () => {
  await page.goto(`${baseUrl}/projects`);
  await page.waitForURL("**/login");
  return { url: page.url() };
});

await record("login_success_enters_admin", async () => {
  await page.goto(`${baseUrl}/login`);
  await page.locator('input[placeholder="请输入用户名"]').fill("admin");
  await page.locator('input[type="password"]').fill("admin123456");
  await page.locator('button[type="submit"]').click();
  await page.waitForURL("**/projects");
  await page.locator("text=项目列表").first().waitFor();
  return {
    url: page.url(),
    projectsVisible: await page.locator("text=项目列表").first().isVisible(),
  };
});

await record("menu_switch_announcements", async () => {
  await page.getByText("公告管理", { exact: true }).click();
  await page.waitForURL("**/announcements");
  return {
    url: page.url(),
    titleVisible: await page.locator("text=公告管理").first().isVisible(),
  };
});

await record("menu_switch_version_compare", async () => {
  await page.getByText("版本对比", { exact: true }).click();
  await page.waitForURL("**/versions/compare");
  return {
    url: page.url(),
    titleVisible: await page.locator("text=版本对比").first().isVisible(),
  };
});

await record("manual_clear_token_redirects_to_login", async () => {
  await page.evaluate(() => localStorage.removeItem("proman_admin_token"));
  await page.goto(`${baseUrl}/projects`);
  await page.waitForURL("**/login");
  return { url: page.url() };
});

await record("invalid_token_request_failure_redirects_to_login", async () => {
  await page.evaluate(() =>
    localStorage.setItem("proman_admin_token", "invalid-token"),
  );
  await page.goto(`${baseUrl}/projects`);
  await page.waitForURL("**/login");
  return { url: page.url() };
});

await browser.close();
console.log(JSON.stringify(results, null, 2));
