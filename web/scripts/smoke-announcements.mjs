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
let targetProjectId = "";
let createdAnnouncementId = "";

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
      description: "used by T605 smoke test",
    },
  });
  const body = await response.json();
  createdProjectIds.push(body.data.project.id);
  return body.data.project.id;
}

async function createAnnouncementByApi(
  projectId,
  { title, content, isPinned = false },
) {
  const token = await getToken();
  const response = await page.request.post(
    `${apiBaseUrl}/api/projects/${projectId}/announcements`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      data: {
        title,
        content,
        is_pinned: isPinned,
      },
    },
  );
  return response.json();
}

async function cleanupProjects() {
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

async function selectProject(projectName) {
  await page.locator('[data-testid="announcement-project-select"]').click();
  await page
    .locator(".ant-select-dropdown:visible")
    .getByText(projectName, { exact: true })
    .click();
}

async function selectStatus(statusLabel) {
  await page.locator('[data-testid="announcement-status-select"]').click();
  await page
    .locator(".ant-select-dropdown:visible")
    .getByText(statusLabel, { exact: true })
    .click();
}

await record("login_success", async () => {
  await loginViaUi();
  return { url: page.url() };
});

const stamp = new Date()
  .toISOString()
  .replace(/[-:TZ.]/g, "")
  .slice(0, 14);
const projectName = `Announcement-Smoke-${stamp}`;
const searchTarget = `Seed Announcement ${stamp} 01`;
const draftTitle = `Draft Notice ${stamp}`;
const publishedTitle = `Published Notice ${stamp}`;

await record("enter_announcements_list_after_login", async () => {
  targetProjectId = String(await createProjectByApi(projectName));
  for (let index = 1; index <= 6; index += 1) {
    await createAnnouncementByApi(targetProjectId, {
      title: `Seed Announcement ${stamp} ${String(index).padStart(2, "0")}`,
      content: `seed content ${index}`,
      isPinned: index === 1,
    });
  }

  await page.getByText("公告管理", { exact: true }).click();
  await page.waitForURL("**/announcements");
  await selectProject(projectName);
  await page.waitForURL(`**/announcements?projectId=${targetProjectId}`);
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "Seed Announcement" })
    .first()
    .waitFor();
  return {
    url: page.url(),
    titleVisible: await page.locator("text=公告管理").first().isVisible(),
  };
});

await record("pagination_keyword_and_status_filters", async () => {
  await page.reload();
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "Seed Announcement" })
    .first()
    .waitFor();
  const pageTwoVisible = await page
    .locator(".ant-pagination-item-2")
    .isVisible();
  await page.locator(".ant-pagination-item-2").click();
  await page
    .locator(".ant-pagination-item-2.ant-pagination-item-active")
    .waitFor();
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "Seed Announcement" })
    .first()
    .waitFor();
  const pageTwoActive = await page
    .locator(".ant-pagination-item-2.ant-pagination-item-active")
    .isVisible();

  await page.locator('input[placeholder="按公告标题搜索"]').fill(searchTarget);
  await page.getByTestId("announcements-search-button").click();
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: searchTarget })
    .first()
    .waitFor();
  const searchRows = await page.locator(".ant-table-tbody tr").count();

  await page.getByTestId("announcements-reset-button").click();
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: `Seed Announcement ${stamp} 06` })
    .first()
    .waitFor();

  await selectStatus("草稿");
  await page.locator(".ant-table-tbody tr").first().waitFor();
  const draftTagVisible = await page
    .locator(".ant-table-tbody .ant-tag")
    .filter({ hasText: "草稿" })
    .first()
    .isVisible();

  await selectStatus("全部状态");

  return {
    pageTwoVisible,
    pageTwoActive,
    searchRows,
    draftTagVisible,
  };
});

await record("create_draft_announcement", async () => {
  await page.getByTestId("announcement-create-button").click();
  await page.waitForURL("**/announcements/new**");
  await page
    .locator('input[placeholder="例如：服务升级通知"]')
    .fill(draftTitle);
  await page.locator("textarea").fill("draft content");
  await page.getByTestId("announcement-submit-button").click();
  await page.waitForURL("**/announcements/*/edit**");
  createdAnnouncementId =
    page.url().match(/\/announcements\/(\d+)\/edit/)?.[1] || "";
  return { url: page.url(), announcementId: createdAnnouncementId };
});

await record("publish_and_edit_published_announcement", async () => {
  await page.goto(`${baseUrl}/announcements?projectId=${targetProjectId}`);
  const row = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: draftTitle })
    .first();
  await row.locator('[data-testid^="announcement-publish-"]').click();
  await page.getByRole("button", { name: "确认发布" }).click();
  await page.locator(`text=公告「${draftTitle}」已发布`).waitFor();

  await selectStatus("已发布");
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: draftTitle })
    .first()
    .waitFor();
  const publishedRow = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: draftTitle })
    .first();
  await publishedRow.locator('[data-testid^="announcement-edit-"]').click();
  await page.waitForURL(`**/announcements/${createdAnnouncementId}/edit**`);
  await page.locator("text=当前公告已发布").first().waitFor();
  await page
    .locator('input[placeholder="例如：服务升级通知"]')
    .fill(publishedTitle);
  await page.locator("textarea").fill("published content updated");
  await page.getByTestId("announcement-submit-button").click();
  await page.locator(`text=公告「${publishedTitle}」已保存`).waitFor();
  await page.getByRole("button", { name: "返回列表" }).click();
  await page.waitForURL(`**/announcements?projectId=${targetProjectId}`);
  await selectStatus("已发布");
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: publishedTitle })
    .first()
    .waitFor();
  const publishedTagVisible = await page
    .locator(".ant-table-tbody .ant-tag")
    .filter({ hasText: "已发布" })
    .first()
    .isVisible();
  return { publishedTagVisible };
});

await record("revoke_and_delete_announcement", async () => {
  const row = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: publishedTitle })
    .first();
  await row.locator('[data-testid^="announcement-revoke-"]').click();
  await page.getByRole("button", { name: "确认撤回" }).click();
  await page.locator(`text=公告「${publishedTitle}」已撤回为草稿`).waitFor();

  await selectStatus("草稿");
  const draftRow = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: publishedTitle })
    .first();
  await draftRow.waitFor();
  await draftRow.locator('[data-testid^="announcement-delete-"]').click();
  await page.getByRole("button", { name: "确认删除" }).click();
  await page.locator("text=公告已删除").waitFor();
  await page.reload();
  await page.locator(".ant-table").waitFor();
  await selectStatus("草稿");

  return {
    deletedVisible: await page
      .locator(".ant-table-tbody tr")
      .filter({ hasText: publishedTitle })
      .count(),
  };
});

await cleanupProjects();
await browser.close();
console.log(JSON.stringify(results, null, 2));
