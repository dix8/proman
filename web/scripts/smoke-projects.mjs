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
page.setDefaultTimeout(12000);
const results = [];
const createdProjectIds = [];

async function record(label, fn) {
  try {
    const value = await fn();
    results.push({ label, ok: true, value });
  } catch (error) {
    results.push({ label, ok: false, error: String(error) });
  }
}

function isProjectsListRequest(request) {
  const url = new URL(request.url());
  return (
    request.method() === "GET" &&
    url.origin === new URL(apiBaseUrl).origin &&
    url.pathname === "/api/projects"
  );
}

async function captureProjectsListRequests(action) {
  const urls = [];
  const onRequestFinished = (request) => {
    if (isProjectsListRequest(request)) {
      urls.push(request.url());
    }
  };

  page.on("requestfinished", onRequestFinished);

  try {
    await action();
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(300);
    return urls;
  } finally {
    page.off("requestfinished", onRequestFinished);
  }
}

async function waitForProjectRowGone(name) {
  await page.waitForFunction((projectName) => {
    const rows = Array.from(document.querySelectorAll(".ant-table-tbody tr"));
    return rows.every((row) => !row.textContent?.includes(projectName));
  }, name);
}

async function loginViaUi() {
  await page.goto(`${baseUrl}/login`);
  await page.locator('input[placeholder="请输入用户名"]').fill("admin");
  await page.locator('input[type="password"]').fill("admin123456");
  await page.locator('button[type="submit"]').click();
  await page.waitForURL("**/projects");
  await page.locator("text=项目列表").first().waitFor();
}

async function getToken() {
  return page.evaluate(() => localStorage.getItem("proman_admin_token") || "");
}

async function createProjectByApi(name, description = "") {
  const token = await getToken();
  const response = await page.request.post(`${apiBaseUrl}/api/projects`, {
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    data: {
      name,
      description,
    },
  });
  const body = await response.json();
  createdProjectIds.push(body.data.project.id);
  return body.data.project.id;
}

async function deleteAllCreatedProjects() {
  const token = await getToken();
  while (createdProjectIds.length > 0) {
    const projectId = createdProjectIds.pop();
    await page.request.delete(`${apiBaseUrl}/api/projects/${projectId}`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
  }
}

await record("login_and_enter_projects", async () => {
  await loginViaUi();
  return {
    url: page.url(),
    titleVisible: await page.locator("text=项目列表").first().isVisible(),
  };
});

const stamp = new Date()
  .toISOString()
  .replace(/[-:TZ.]/g, "")
  .slice(0, 14);
const uiProjectName = `UI-Project-${stamp}`;
const seededSearchTarget = `Seed-Project-${stamp}-01`;

await record("seed_projects_for_pagination", async () => {
  for (let i = 1; i <= 6; i += 1) {
    await createProjectByApi(
      `Seed-Project-${stamp}-${String(i).padStart(2, "0")}`,
      "seed project",
    );
  }
  await page.reload();
  await page.locator("text=项目列表").first().waitFor();
  return {
    paginationVisible: await page.locator(".ant-pagination").isVisible(),
    pageTwoVisible: await page.locator(".ant-pagination-item-2").isVisible(),
  };
});

await record("pagination_switch_to_page_two", async () => {
  await page.locator(".ant-pagination-item-2").click();
  await page.waitForLoadState("networkidle");
  return {
    pageTwoActive: await page
      .locator(".ant-pagination-item-2.ant-pagination-item-active")
      .isVisible(),
  };
});

await record("create_project_via_ui", async () => {
  await page.getByRole("button", { name: "新建项目" }).click();
  await page
    .locator('.ant-modal input[placeholder*="OpenAPI"]')
    .fill(uiProjectName);
  await page.locator(".ant-modal textarea").fill("created from smoke test");
  await page.getByRole("button", { name: "创建项目" }).click();
  const tokenModal = page
    .locator(".ant-modal")
    .filter({ hasText: "项目 Token（仅显示一次）" });
  await tokenModal.waitFor();
  const tokenText =
    (await tokenModal.locator(".token-text").textContent())?.trim() || "";
  await tokenModal.getByRole("button", { name: "我已知晓" }).click();
  await tokenModal.waitFor({ state: "hidden" });
  await page.locator(`text=${uiProjectName}`).first().waitFor();
  return {
    tokenShownOnce: tokenText.length > 10,
    tokenStillVisibleAfterClose: await page.locator(".token-text").count(),
  };
});

await record("search_project", async () => {
  const searchInput = page.locator('input[placeholder="按项目名称搜索"]');
  await searchInput.fill(seededSearchTarget);
  const searchRequests = await captureProjectsListRequests(async () => {
    await page.getByTestId("projects-search-button").click();
  });
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: seededSearchTarget })
    .first()
    .waitFor();
  const rows = await page.locator(".ant-table-tbody tr").count();
  const resetRequests = await captureProjectsListRequests(async () => {
    await page.getByTestId("projects-reset-button").click();
  });
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: uiProjectName })
    .first()
    .waitFor();

  if (searchRequests.length !== 1) {
    throw new Error(`search triggered ${searchRequests.length} list requests`);
  }
  if (resetRequests.length !== 1) {
    throw new Error(`reset triggered ${resetRequests.length} list requests`);
  }

  return {
    searchValue: seededSearchTarget,
    rowCount: rows,
    searchRequestCount: searchRequests.length,
    resetRequestCount: resetRequests.length,
  };
});

await record("edit_project", async () => {
  const row = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: uiProjectName })
    .first();
  const editButton = row.locator('[data-testid^=\"project-edit-\"]');
  await editButton.click();
  const input = page.locator(".ant-modal input").first();
  await input.waitFor();
  await input.fill(`${uiProjectName}-Edited`);
  await page.getByRole("button", { name: "保存修改" }).click();
  await page.locator(`text=${uiProjectName}-Edited`).first().waitFor();
  return {
    updatedVisible: await page
      .locator(`text=${uiProjectName}-Edited`)
      .first()
      .isVisible(),
  };
});

await record("refresh_token_one_time_prompt", async () => {
  const row = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: `${uiProjectName}-Edited` })
    .first();
  const refreshButton = row.locator(
    '[data-testid^=\"project-refresh-token-\"]',
  );
  await refreshButton.click();
  const tokenModal = page
    .locator(".ant-modal")
    .filter({ hasText: "新项目 Token（仅显示一次）" });
  await tokenModal.waitFor();
  const warningVisible = await tokenModal
    .locator("text=旧 Token 已立即失效")
    .isVisible();
  const tokenText =
    (await tokenModal.locator(".token-text").textContent())?.trim() || "";
  await tokenModal.getByRole("button", { name: "我已知晓" }).click();
  await tokenModal.waitFor({ state: "hidden" });
  return {
    warningVisible,
    tokenShownOnce: tokenText.length > 10,
    tokenRetainedInPage: await page.locator(".token-text").count(),
  };
});

await record("delete_with_confirmation", async () => {
  const row = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: `${uiProjectName}-Edited` })
    .first();
  const deleteButton = row.locator('[data-testid^=\"project-delete-\"]');
  await deleteButton.click();
  const confirmButton = page.getByRole("button", { name: "确认删除" });
  await confirmButton.waitFor();
  const warningVisible = await page.locator("text=确认删除项目").isVisible();
  await confirmButton.click();
  await page.locator("text=项目已删除").waitFor();
  await waitForProjectRowGone(`${uiProjectName}-Edited`);
  const rowAfterDelete = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: `${uiProjectName}-Edited` })
    .first();
  return {
    warningVisible,
    deletedStillVisible: await rowAfterDelete.isVisible().catch(() => false),
  };
});

await record("invalid_token_redirects_to_login", async () => {
  await page.evaluate(() =>
    localStorage.setItem("proman_admin_token", "invalid-token"),
  );
  await page.goto(`${baseUrl}/projects`);
  await page.waitForURL("**/login");
  return { url: page.url() };
});

await deleteAllCreatedProjects();
await browser.close();
console.log(JSON.stringify(results, null, 2));
