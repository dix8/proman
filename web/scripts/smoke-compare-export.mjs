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
let version100Id = "";
let version110Id = "";

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
      description: "used by T607 smoke test",
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

async function createChangelogByApi(targetVersionId, type, content) {
  const token = await getToken();
  await page.request.post(
    `${apiBaseUrl}/api/versions/${targetVersionId}/changelogs`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      data: { type, content },
    },
  );
}

async function publishVersionByApi(targetVersionId) {
  const token = await getToken();
  await page.request.put(
    `${apiBaseUrl}/api/versions/${targetVersionId}/publish`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  );
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

async function selectProject(projectName) {
  const select = page.getByTestId("compare-project-select");
  await select.click({ force: true });
  const input = select.locator("input");
  await input.press("Control+A");
  await input.fill(projectName);
  await page.keyboard.press("Enter");
}

async function selectCompareFrom(version) {
  const select = page.getByTestId("compare-from-select");
  await select.click({ force: true });
  const input = select.locator("input");
  await input.press("Control+A");
  await input.fill(version);
  await page.keyboard.press("Enter");
}

async function selectCompareTo(version) {
  const select = page.getByTestId("compare-to-select");
  await select.click({ force: true });
  const input = select.locator("input");
  await input.press("Control+A");
  await input.fill(version);
  await page.keyboard.press("Enter");
}

async function selectExportFormat(formatLabel) {
  const select = page.getByTestId("export-format-select");
  const currentText = (await select.textContent()) || "";
  if (currentText.includes(formatLabel)) {
    return;
  }

  await select.click({ force: true });
  if (formatLabel === "JSON") {
    await page.keyboard.press("ArrowDown");
  } else {
    await page.keyboard.press("ArrowUp");
  }
  await page.keyboard.press("Enter");
}

async function selectExportVersion(label) {
  const select = page.getByTestId("export-version-select");
  await select.click({ force: true });
  const input = select.locator("input");
  await input.press("Control+A");
  await input.fill(label);
  await page.keyboard.press("Enter");
}

async function requestExportFile(
  targetProjectId,
  format,
  targetVersionId = "",
) {
  const token = await getToken();
  const response = await page.request.get(
    `${apiBaseUrl}/api/projects/${targetProjectId}/changelogs/export`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      params: {
        format,
        version_id: targetVersionId || undefined,
      },
    },
  );

  const content = await response.text();
  return {
    filename: response.headers()["content-disposition"] || "",
    content,
  };
}

await record("login_success", async () => {
  await loginViaUi();
  return { url: page.url() };
});

const stamp = new Date()
  .toISOString()
  .replace(/[-:TZ.]/g, "")
  .slice(0, 14);
const projectName = `Compare-Export-${stamp}`;

await record("prepare_compare_data", async () => {
  projectId = String(await createProjectByApi(projectName));

  version100Id = String(await createVersionByApi(projectId, 1, 0, 0));
  await createChangelogByApi(version100Id, "added", "base feature");
  await publishVersionByApi(version100Id);

  version110Id = String(await createVersionByApi(projectId, 1, 1, 0));
  await createChangelogByApi(version110Id, "fixed", "reverse order bug");
  await publishVersionByApi(version110Id);

  return { projectId, version100Id, version110Id };
});

await record("enter_compare_page", async () => {
  await page.getByText("版本对比", { exact: true }).click();
  await page.waitForURL("**/versions/compare");
  await selectProject(projectName);
  await page.locator("text=版本对比").first().waitFor();
  return { url: page.url() };
});

await record("reverse_order_compare_uses_server_normalization", async () => {
  await selectCompareFrom("1.1.0");
  await selectCompareTo("1.0.0");
  const compareResponse = page.waitForResponse(
    (response) =>
      response.url().includes("/versions/compare") &&
      response.request().method() === "GET",
  );
  await page.getByTestId("compare-submit-button").click();
  await compareResponse;
  await page
    .getByTestId("compare-normalized-from")
    .getByText("1.0.0", { exact: true })
    .waitFor();
  await page
    .getByTestId("compare-normalized-to")
    .getByText("1.1.0", { exact: true })
    .waitFor();
  await page
    .getByTestId("compare-versions-range")
    .getByText("1.0.0", { exact: true })
    .waitFor();
  await page
    .getByTestId("compare-versions-range")
    .getByText("1.1.0", { exact: true })
    .waitFor();
  await page
    .getByTestId("compare-group-added")
    .getByText("base feature")
    .waitFor();
  await page
    .getByTestId("compare-group-fixed")
    .getByText("reverse order bug")
    .waitFor();
  return {
    normalizedFrom: "1.0.0",
    normalizedTo: "1.1.0",
  };
});

await record("same_version_compare", async () => {
  await selectCompareFrom("1.0.0");
  await selectCompareTo("1.0.0");
  const compareResponse = page.waitForResponse(
    (response) =>
      response.url().includes("/versions/compare") &&
      response.request().method() === "GET",
  );
  await page.getByTestId("compare-submit-button").click();
  await compareResponse;
  await page
    .getByTestId("compare-normalized-from")
    .getByText("1.0.0", { exact: true })
    .waitFor();
  await page
    .getByTestId("compare-normalized-to")
    .getByText("1.0.0", { exact: true })
    .waitFor();
  const rangeTags = await page
    .getByTestId("compare-versions-range")
    .locator(".ant-tag")
    .count();
  return { rangeTags };
});

await record("project_markdown_and_json_export", async () => {
  const beforeProjectExportCaptures = await page.evaluate(
    () => window.__promanDownloads?.length || 0,
  );
  await selectExportFormat("Markdown");
  const markdownResponse = page.waitForResponse(
    (response) =>
      response.url().includes("/changelogs/export") &&
      response.request().method() === "GET",
  );
  await page.getByTestId("export-project-button").click();
  await markdownResponse;
  await page.waitForFunction(
    (count) => (window.__promanDownloads?.length || 0) > count,
    beforeProjectExportCaptures,
  );
  const markdownCapture = await page.evaluate(() =>
    window.__promanDownloads.at(-1),
  );
  const markdown = await requestExportFile(projectId, "markdown");

  await selectExportFormat("JSON");
  const jsonResponse = page.waitForResponse(
    (response) =>
      response.url().includes("/changelogs/export") &&
      response.request().method() === "GET",
  );
  await page.getByTestId("export-project-button").click();
  await jsonResponse;
  await page.waitForFunction(
    (count) => (window.__promanDownloads?.length || 0) > count,
    beforeProjectExportCaptures + 1,
  );
  const jsonCapture = await page.evaluate(() =>
    window.__promanDownloads.at(-1),
  );
  const json = await requestExportFile(projectId, "json");

  return {
    markdownFilename: markdownCapture?.filename || "",
    jsonFilename: jsonCapture?.filename || "",
    markdownHasProjectName: markdown.content.includes(projectName),
    jsonHasVersion110: json.content.includes('"version": "1.1.0"'),
    markdownUiSuccess: Boolean(markdownCapture?.filename),
    jsonUiSuccess: Boolean(jsonCapture?.filename),
  };
});

await record("single_version_markdown_and_json_export", async () => {
  const beforeSingleVersionExportCaptures = await page.evaluate(
    () => window.__promanDownloads?.length || 0,
  );
  await selectExportVersion("1.0.0 (published)");

  await selectExportFormat("Markdown");
  const markdownResponse = page.waitForResponse(
    (response) =>
      response.url().includes("/changelogs/export") &&
      response.request().method() === "GET",
  );
  await page.getByTestId("export-version-button").click();
  await markdownResponse;
  await page.waitForFunction(
    (count) => (window.__promanDownloads?.length || 0) > count,
    beforeSingleVersionExportCaptures,
  );
  const markdownCapture = await page.evaluate(() =>
    window.__promanDownloads.at(-1),
  );
  const markdown = await requestExportFile(projectId, "markdown", version100Id);

  await selectExportFormat("JSON");
  const jsonResponse = page.waitForResponse(
    (response) =>
      response.url().includes("/changelogs/export") &&
      response.request().method() === "GET",
  );
  await page.getByTestId("export-version-button").click();
  await jsonResponse;
  await page.waitForFunction(
    (count) => (window.__promanDownloads?.length || 0) > count,
    beforeSingleVersionExportCaptures + 1,
  );
  const jsonCapture = await page.evaluate(() =>
    window.__promanDownloads.at(-1),
  );
  const json = await requestExportFile(projectId, "json", version100Id);

  return {
    markdownFilename: markdownCapture?.filename || "",
    jsonFilename: jsonCapture?.filename || "",
    markdownContainsOnlyVersion100:
      markdown.content.includes("## 1.0.0") &&
      !markdown.content.includes("## 1.1.0"),
    jsonContainsOnlyVersion100:
      json.content.includes('"version": "1.0.0"') &&
      !json.content.includes('"version": "1.1.0"'),
    markdownUiSuccess: Boolean(markdownCapture?.filename),
    jsonUiSuccess: Boolean(jsonCapture?.filename),
  };
});

await cleanupProjects();
await browser.close();
console.log(JSON.stringify(results, null, 2));
