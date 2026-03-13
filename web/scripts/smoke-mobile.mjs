import { chromium } from "playwright-core";

const baseUrl = process.env.FRONTEND_BASE_URL || "http://127.0.0.1:5173";
const apiBaseUrl = process.env.API_BASE_URL || "http://localhost:8080";
const browserExecutablePath =
  process.env.BROWSER_EXECUTABLE_PATH ||
  "C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe";

const browser = await chromium.launch({
  executablePath: browserExecutablePath,
  headless: true,
});

const page = await browser.newPage({
  viewport: { width: 390, height: 844 },
  isMobile: true,
});
page.setDefaultTimeout(15000);

const results = [];
const createdProjectIds = [];
let projectId = "";
let versionId = "";

async function record(label, fn) {
  try {
    const value = await fn();
    results.push({ label, ok: true, value });
  } catch (error) {
    results.push({ label, ok: false, error: String(error) });
  }
}

async function loginViaUi() {
  await page.goto(`${baseUrl}/login`);
  await page.locator('input[placeholder="请输入用户名"]').fill("admin");
  await page.locator('input[type="password"]').fill("admin123456");
  await page.locator('button[type="submit"]').click();
  await page.waitForURL("**/projects");
}

async function getToken() {
  return page.evaluate(() => localStorage.getItem("proman_admin_token") || "");
}

async function createProjectByApi(name) {
  const token = await getToken();
  const response = await page.request.post(`${apiBaseUrl}/api/projects`, {
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    data: {
      name,
      description: "used by mobile smoke",
    },
  });
  const body = await response.json();
  createdProjectIds.push(body.data.project.id);
  return body.data.project.id;
}

async function createVersionByApi(targetProjectId, major, minor, patch) {
  const token = await getToken();
  const response = await page.request.post(
    `${apiBaseUrl}/api/projects/${targetProjectId}/versions`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      data: { major, minor, patch },
    },
  );
  const body = await response.json();
  return body.data.id;
}

async function cleanupProjects() {
  const token = await getToken();
  while (createdProjectIds.length > 0) {
    const targetProjectId = createdProjectIds.pop();
    await page.request.delete(`${apiBaseUrl}/api/projects/${targetProjectId}`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  }
}

async function measureOverflow() {
  return page.evaluate(() => ({
    innerWidth: window.innerWidth,
    scrollWidth: document.documentElement.scrollWidth,
  }));
}

await record("mobile_login_and_nav_drawer", async () => {
  await loginViaUi();
  await page.locator(".mobile-nav-trigger").waitFor();
  await page.locator(".mobile-nav-trigger").click();
  await page.locator(".mobile-nav-drawer").waitFor();
  return {
    url: page.url(),
    drawerVisible: await page.locator(".mobile-nav-drawer").isVisible(),
  };
});

const stamp = new Date()
  .toISOString()
  .replace(/[-:TZ.]/g, "")
  .slice(0, 14);

await record("seed_mobile_entities", async () => {
  projectId = String(await createProjectByApi(`Mobile-Smoke-${stamp}`));
  versionId = String(await createVersionByApi(projectId, 1, 0, 0));
  return { projectId, versionId };
});

await record("projects_page_mobile_usable", async () => {
  await page.goto(`${baseUrl}/projects`);
  await page.locator('button:has-text("新建项目")').waitFor();
  await page.getByRole("button", { name: "新建项目" }).click();
  const modal = page.locator(".ant-modal").last();
  await modal.waitFor();
  const overflow = await measureOverflow();
  await page.keyboard.press("Escape");
  return {
    modalVisible: true,
    noMajorOverflow: overflow.scrollWidth <= overflow.innerWidth + 16,
  };
});

await record("project_detail_version_list_and_changelog_mobile", async () => {
  await page.goto(`${baseUrl}/projects/${projectId}`);
  await page.getByRole("button", { name: "查看版本" }).click();
  await page.waitForURL(`**/projects/${projectId}/versions`);
  await page.getByRole("button", { name: "新建版本" }).waitFor();

  await page.goto(
    `${baseUrl}/projects/${projectId}/versions/${versionId}/changelogs`,
  );
  await page.getByRole("button", { name: "新增日志" }).click();
  const modal = page.locator(".ant-modal").last();
  await modal.waitFor();
  const previewTabVisible = await modal
    .getByRole("tab", { name: "预览" })
    .isVisible();
  const overflow = await measureOverflow();
  await page.keyboard.press("Escape");

  return {
    previewTabVisible,
    noMajorOverflow: overflow.scrollWidth <= overflow.innerWidth + 16,
  };
});

await record("announcements_and_compare_mobile", async () => {
  await page.goto(`${baseUrl}/announcements?projectId=${projectId}`);
  await page.getByRole("button", { name: "新建公告" }).click();
  await page.waitForURL(`**/announcements/new?projectId=${projectId}`);
  await page.locator('input[placeholder="例如：服务升级通知"]').waitFor();
  const announcementOverflow = await measureOverflow();

  await page.goto(`${baseUrl}/versions/compare`);
  await page.getByTestId("compare-project-select").waitFor();
  const compareOverflow = await measureOverflow();

  return {
    announcementNoMajorOverflow:
      announcementOverflow.scrollWidth <= announcementOverflow.innerWidth + 16,
    compareNoMajorOverflow:
      compareOverflow.scrollWidth <= compareOverflow.innerWidth + 16,
  };
});

await cleanupProjects();
await browser.close();
console.log(JSON.stringify(results, null, 2));
