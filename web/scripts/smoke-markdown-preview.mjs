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

const page = await browser.newPage();
page.setDefaultTimeout(15000);
const results = [];
const createdProjectIds = [];
let projectId = "";
let versionId = "";

function isPreviewRequest(request) {
  const url = new URL(request.url());
  return (
    request.method() === "POST" &&
    url.origin === new URL(apiBaseUrl).origin &&
    url.pathname === "/api/markdown/preview"
  );
}

async function record(label, fn) {
  try {
    const value = await fn();
    results.push({ label, ok: true, value });
  } catch (error) {
    results.push({ label, ok: false, error: String(error) });
  }
}

async function capturePreviewRequests(action) {
  const requests = [];
  const handler = (request) => {
    if (isPreviewRequest(request)) {
      requests.push(request.url());
    }
  };

  page.on("requestfinished", handler);

  try {
    await action();
    await page.waitForTimeout(400);
    return requests;
  } finally {
    page.off("requestfinished", handler);
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
      description: "used by T606 smoke test",
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
      data: {
        major,
        minor,
        patch,
      },
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

await record("login_success", async () => {
  await loginViaUi();
  return { url: page.url() };
});

const stamp = new Date()
  .toISOString()
  .replace(/[-:TZ.]/g, "")
  .slice(0, 14);

await record("prepare_project_and_version", async () => {
  projectId = String(await createProjectByApi(`Preview-Smoke-${stamp}`));
  versionId = String(await createVersionByApi(projectId, 1, 0, 0));
  return { projectId, versionId };
});

await record("changelog_preview_empty_and_normal_markdown", async () => {
  await page.goto(
    `${baseUrl}/projects/${projectId}/versions/${versionId}/changelogs`,
  );
  await page.getByTestId("changelog-create-button").click();
  const modal = page.locator(".ant-modal").last();
  await modal.waitFor();

  const emptyPreviewRequests = await capturePreviewRequests(async () => {
    await modal.getByRole("tab", { name: "预览" }).click();
  });
  await modal.getByText("输入日志内容后即可预览").waitFor();

  await modal.getByRole("tab", { name: "编辑" }).click();
  const textarea = modal.locator("textarea");
  await textarea.fill("## 日志标题\n\n- 第一项");

  const normalPreviewRequests = await capturePreviewRequests(async () => {
    await modal.getByRole("tab", { name: "预览" }).click();
  });

  await modal
    .locator(".markdown-preview-pane h2")
    .filter({ hasText: "日志标题" })
    .waitFor();
  await modal
    .locator(".markdown-preview-pane li")
    .filter({ hasText: "第一项" })
    .waitFor();
  await page.keyboard.press("Escape");

  return {
    emptyPreviewRequestCount: emptyPreviewRequests.length,
    normalPreviewRequestCount: normalPreviewRequests.length,
    headingVisible: true,
  };
});

await record("changelog_preview_sanitizes_and_handles_failure", async () => {
  await page.goto(
    `${baseUrl}/projects/${projectId}/versions/${versionId}/changelogs`,
  );
  await page.getByTestId("changelog-create-button").click();
  const modal = page.locator(".ant-modal").last();
  await modal.waitFor();

  await modal
    .locator("textarea")
    .fill(
      "safe text\n\n<script>alert(1)</script>\n\n[bad](javascript:alert(1))",
    );
  await modal.getByRole("tab", { name: "预览" }).click();
  await modal.locator(".markdown-preview-pane").waitFor();
  const sanitizedHtml = await modal
    .locator(".markdown-preview-pane")
    .evaluate((element) => element.innerHTML);

  const failureHandler = async (route) => {
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: JSON.stringify({ code: 50001, message: "预览服务异常" }),
    });
    await page.unroute("**/api/markdown/preview", failureHandler);
  };

  await modal.getByRole("tab", { name: "编辑" }).click();
  await modal.locator("textarea").fill("## 这次预览会失败");
  await page.route("**/api/markdown/preview", failureHandler);
  await modal.getByRole("tab", { name: "预览" }).click();
  await modal.getByText("预览服务异常").waitFor();
  await page.keyboard.press("Escape");

  return {
    containsScriptTag: sanitizedHtml.includes("<script"),
    containsJavascriptScheme: sanitizedHtml.includes("javascript:"),
    failureMessageVisible: true,
  };
});

await record(
  "announcement_preview_uses_server_html_and_sanitizes",
  async () => {
    await page.goto(`${baseUrl}/announcements/new?projectId=${projectId}`);
    await page
      .locator('input[placeholder="例如：服务升级通知"]')
      .fill(`Preview Announcement ${stamp}`);
    const editor = page.locator("textarea");
    await editor.fill("## 公告标题\n\n[正常链接](https://example.com)");

    const announcementPreviewRequests = await capturePreviewRequests(
      async () => {
        await page.getByRole("tab", { name: "预览" }).click();
      },
    );
    await page
      .locator(".markdown-preview-pane h2")
      .filter({ hasText: "公告标题" })
      .waitFor();

    await page.getByRole("tab", { name: "编辑" }).click();
    await editor.fill(
      "notice\n\n<script>alert(2)</script>\n\n[bad](javascript:alert(2))",
    );
    await page.getByRole("tab", { name: "预览" }).click();
    await page.locator(".markdown-preview-pane").waitFor();
    const sanitizedHtml = await page
      .locator(".markdown-preview-pane")
      .evaluate((element) => element.innerHTML);

    return {
      previewRequestCount: announcementPreviewRequests.length,
      containsScriptTag: sanitizedHtml.includes("<script"),
      containsJavascriptScheme: sanitizedHtml.includes("javascript:"),
      headingVisible: true,
    };
  },
);

await cleanupProjects();
await browser.close();
console.log(JSON.stringify(results, null, 2));
