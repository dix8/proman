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
let draftVersionId = "";
let publishCandidateVersionId = "";

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
      description: "used by T604 smoke test",
    },
  });
  const body = await response.json();
  createdProjectIds.push(body.data.project.id);
  return body.data.project.id;
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

async function fillVersionForm(major, minor, patch) {
  const inputs = page.locator('input[role="spinbutton"]');
  const values = [major, minor, patch];

  for (let index = 0; index < values.length; index += 1) {
    const input = inputs.nth(index);
    await input.click();
    await input.press("Control+A");
    await input.fill(String(values[index]));
    await input.press("Tab");
  }
}

async function createChangelogViaUi({ typeLabel = "新增", content }) {
  await page.getByTestId("changelog-create-button").click();
  const modal = page.locator(".ant-modal").last();
  await modal.waitFor();

  if (typeLabel !== "新增") {
    await modal.locator(".ant-select").click();
    await page
      .locator(".ant-select-dropdown:visible")
      .getByText(typeLabel, { exact: true })
      .click();
  }

  await modal.locator("textarea").fill(content);
  await modal.getByRole("button", { name: "创建日志" }).click();
  await modal.waitFor({ state: "hidden" });
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: content })
    .first()
    .waitFor();
}

await record("login_success", async () => {
  await loginViaUi();
  return { url: page.url() };
});

const stamp = new Date()
  .toISOString()
  .replace(/[-:TZ.]/g, "")
  .slice(0, 14);
const projectName = `Version-Smoke-${stamp}`;
let projectId = 0;

await record("enter_version_list_after_login", async () => {
  projectId = await createProjectByApi(projectName);
  await page.goto(`${baseUrl}/projects`);
  const projectEntry = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: projectName })
    .first();
  await projectEntry.waitFor();
  await projectEntry
    .getByRole("button", { name: projectName, exact: true })
    .click();
  await page.waitForURL(`**/projects/${projectId}`);
  await page.locator("text=返回项目列表").first().waitFor();
  await page.getByRole("button", { name: "查看版本" }).click();
  await page.waitForURL(`**/projects/${projectId}/versions`);
  await page.locator("text=新建版本").waitFor();
  return {
    url: page.url(),
    detailVisible: await page
      .locator(`text=${projectName}`)
      .first()
      .isVisible(),
    createVersionVisible: await page.locator("text=新建版本").isVisible(),
  };
});

await record("create_draft_version", async () => {
  await page.getByTestId("version-create-button").click();
  await page.waitForURL(`**/projects/${projectId}/versions/new`);
  await fillVersionForm(1, 0, 0);
  await page.getByTestId("version-submit-button").click();
  await page.waitForURL(`**/projects/${projectId}/versions/*/edit`);
  draftVersionId = page.url().match(/\/versions\/(\d+)\/edit/)?.[1] || "";
  return { url: page.url(), versionId: draftVersionId };
});

await record("edit_draft_version", async () => {
  await fillVersionForm(1, 0, 1);
  await page.getByTestId("version-submit-button").click();
  await page.locator("text=已更新").first().waitFor();
  await page.goto(`${baseUrl}/projects/${projectId}/versions`);
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "1.0.1" })
    .first()
    .waitFor();
  return {
    updatedVisible: await page
      .locator(".ant-table-tbody tr")
      .filter({ hasText: "1.0.1" })
      .first()
      .isVisible(),
  };
});

await record("delete_draft_version", async () => {
  await page.goto(`${baseUrl}/projects/${projectId}/versions`);
  const row = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "1.0.1" })
    .first();
  await row.locator('[data-testid^="version-delete-"]').click();
  await page.getByRole("button", { name: "确认删除" }).click();
  await page.locator("text=草稿版本已删除").waitFor();
  await page.reload();
  await page.locator(".ant-table-tbody").waitFor();
  return {
    deletedVisible: await page
      .locator(".ant-table-tbody tr")
      .filter({ hasText: "1.0.1" })
      .count(),
  };
});

await record("create_version_for_publish_and_logs", async () => {
  await page.goto(`${baseUrl}/projects/${projectId}/versions`);
  await page.getByTestId("version-create-button").click();
  await page.waitForURL(`**/projects/${projectId}/versions/new`);
  await fillVersionForm(2, 0, 0);
  await page.getByTestId("version-submit-button").click();
  await page.waitForURL(`**/projects/${projectId}/versions/*/edit`);
  publishCandidateVersionId =
    page.url().match(/\/versions\/(\d+)\/edit/)?.[1] || "";
  await page.getByTestId("version-open-changelogs-button").click();
  await page.waitForURL(`**/projects/${projectId}/versions/*/changelogs`);
  return { url: page.url(), versionId: publishCandidateVersionId };
});

await record("create_edit_delete_and_reorder_changelogs", async () => {
  await page.goto(
    `${baseUrl}/projects/${projectId}/versions/${publishCandidateVersionId}/changelogs`,
  );
  await createChangelogViaUi({
    typeLabel: "新增",
    content: "support feature A",
  });
  await createChangelogViaUi({
    typeLabel: "修复",
    content: "fix bug B",
  });

  const editRow = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "fix bug B" })
    .first();
  await editRow.locator('[data-testid^="changelog-edit-"]').click();
  const modal = page.locator(".ant-modal").last();
  await modal.waitFor();
  await modal.locator("textarea").fill("fix bug B updated");
  await modal.getByRole("button", { name: "保存修改" }).click();
  await modal.waitFor({ state: "hidden" });
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "fix bug B updated" })
    .first()
    .waitFor();

  const reorderRow = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "fix bug B updated" })
    .first();
  await reorderRow.locator('[data-testid^="changelog-move-up-"]').click();
  await page.getByTestId("changelog-save-order-button").click();
  await page.locator("text=日志顺序已保存").waitFor();

  const firstRowTextAfterSave =
    (await page.locator(".ant-table-tbody tr").first().textContent()) || "";
  if (!firstRowTextAfterSave.includes("fix bug B updated")) {
    throw new Error("reordered changelog did not move to top before reload");
  }

  await page.reload();
  await page.locator(".ant-table-tbody tr").first().waitFor();
  const firstRowTextAfterReload =
    (await page.locator(".ant-table-tbody tr").first().textContent()) || "";
  if (!firstRowTextAfterReload.includes("fix bug B updated")) {
    throw new Error("reordered changelog order was not persisted after reload");
  }

  const deleteRow = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "support feature A" })
    .first();
  await deleteRow.locator('[data-testid^="changelog-delete-"]').click();
  await page.getByRole("button", { name: "确认删除" }).click();
  await page.locator("text=日志已删除").waitFor();

  return {
    firstRowAfterReload: firstRowTextAfterReload,
    remainingRows: await page.locator(".ant-table-tbody tr").count(),
  };
});

await record("publish_version_and_verify_readonly", async () => {
  await page.goto(`${baseUrl}/projects/${projectId}/versions`);
  const row = page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "2.0.0" })
    .first();
  await row.locator('[data-testid^="version-publish-"]').click();
  await page.getByRole("button", { name: "确认发布" }).click();
  await page.locator("text=版本 2.0.0 已发布").waitFor();
  await row.locator("text=已发布 / 只读").waitFor();

  const editButton = row.locator('[data-testid^="version-edit-"]');
  await editButton.click();
  await page.waitForURL(`**/projects/${projectId}/versions/*/edit`);

  const submitButtons = await page.getByTestId("version-submit-button").count();
  if (submitButtons !== 0) {
    throw new Error("published version should not show submit button");
  }

  await page.getByTestId("version-open-changelogs-button").click();
  await page.waitForURL(`**/projects/${projectId}/versions/*/changelogs`);
  await page
    .locator(".ant-table-tbody tr")
    .filter({ hasText: "fix bug B updated" })
    .first()
    .waitFor();

  const createButtonCount = await page
    .getByTestId("changelog-create-button")
    .count();
  const saveOrderButtonCount = await page
    .getByTestId("changelog-save-order-button")
    .count();
  const editActionCount = await page
    .locator('[data-testid^="changelog-edit-"]')
    .count();
  const deleteActionCount = await page
    .locator('[data-testid^="changelog-delete-"]')
    .count();
  const viewActionCount = await page
    .locator('[data-testid^="changelog-view-"]')
    .count();

  if (
    createButtonCount !== 0 ||
    saveOrderButtonCount !== 0 ||
    editActionCount !== 0 ||
    deleteActionCount !== 0 ||
    viewActionCount < 1
  ) {
    throw new Error("published version changelog page should be read-only");
  }

  return {
    createButtonCount,
    saveOrderButtonCount,
    editActionCount,
    deleteActionCount,
    viewActionCount,
  };
});

await cleanupProjects();
await browser.close();
console.log(JSON.stringify(results, null, 2));
