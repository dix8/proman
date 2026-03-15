import { http } from "./http";
import { parseContentDispositionFilename } from "../utils/download";

export const VERSION_STATUS_DRAFT = "draft";
export const VERSION_STATUS_PUBLISHED = "published";

export const CHANGELOG_TYPE_OPTIONS = [
  { label: "新增", value: "added" },
  { label: "修改", value: "changed" },
  { label: "修复", value: "fixed" },
  { label: "优化", value: "improved" },
  { label: "废弃", value: "deprecated" },
  { label: "移除", value: "removed" },
];

export async function fetchVersions(projectId, params) {
  const response = await http.get(`/api/projects/${projectId}/versions`, {
    params,
  });
  return response.data.data;
}

export async function fetchAllVersions(projectId, status = "") {
  const pageSize = 100;
  const firstPage = await fetchVersions(projectId, {
    page: 1,
    page_size: pageSize,
    status: status || undefined,
  });

  const list = [...firstPage.list];
  const totalPages = Math.ceil(firstPage.total / pageSize);

  for (let page = 2; page <= totalPages; page += 1) {
    const nextPage = await fetchVersions(projectId, {
      page,
      page_size: pageSize,
      status: status || undefined,
    });
    list.push(...nextPage.list);
  }

  return {
    ...firstPage,
    list,
  };
}

export async function fetchVersion(versionId) {
  const response = await http.get(`/api/versions/${versionId}`);
  return response.data.data;
}

export async function createVersion(projectId, payload) {
  const response = await http.post(
    `/api/projects/${projectId}/versions`,
    payload,
  );
  return response.data.data;
}

export async function updateVersion(versionId, payload) {
  const response = await http.put(`/api/versions/${versionId}`, payload);
  return response.data.data;
}

export async function deleteVersion(versionId) {
  const response = await http.delete(`/api/versions/${versionId}`);
  return response.data.data;
}

export async function publishVersion(versionId) {
  const response = await http.put(`/api/versions/${versionId}/publish`);
  return response.data.data;
}

export async function unpublishVersion(versionId) {
  const response = await http.put(`/api/versions/${versionId}/unpublish`);
  return response.data.data;
}

export async function fetchChangelogs(versionId, params) {
  const response = await http.get(`/api/versions/${versionId}/changelogs`, {
    params,
  });
  return response.data.data;
}

export async function fetchAllChangelogs(versionId, changelogType = "") {
  const pageSize = 100;
  const firstPage = await fetchChangelogs(versionId, {
    page: 1,
    page_size: pageSize,
    type: changelogType || undefined,
  });

  const list = [...firstPage.list];
  const totalPages = Math.ceil(firstPage.total / pageSize);

  for (let page = 2; page <= totalPages; page += 1) {
    const nextPage = await fetchChangelogs(versionId, {
      page,
      page_size: pageSize,
      type: changelogType || undefined,
    });
    list.push(...nextPage.list);
  }

  return {
    ...firstPage,
    list,
  };
}

export async function createChangelog(versionId, payload) {
  const response = await http.post(
    `/api/versions/${versionId}/changelogs`,
    payload,
  );
  return response.data.data;
}

export async function updateChangelog(changelogId, payload) {
  const response = await http.put(`/api/changelogs/${changelogId}`, payload);
  return response.data.data;
}

export async function deleteChangelog(changelogId) {
  const response = await http.delete(`/api/changelogs/${changelogId}`);
  return response.data.data;
}

export async function reorderChangelogs(versionId, items) {
  const response = await http.put(
    `/api/versions/${versionId}/changelogs/reorder`,
    { items },
  );
  return response.data.data;
}

export async function compareVersions(projectId, fromVersionId, toVersionId) {
  const response = await http.get(
    `/api/projects/${projectId}/versions/compare`,
    {
      params: {
        from_version_id: fromVersionId,
        to_version_id: toVersionId,
      },
    },
  );
  return response.data.data;
}

export async function exportChangelogs(projectId, format, versionId) {
  const response = await http.get(
    `/api/projects/${projectId}/changelogs/export`,
    {
      params: {
        format,
        version_id: versionId || undefined,
      },
      responseType: "blob",
    },
  );

  const filename = parseContentDispositionFilename(
    response.headers["content-disposition"],
  );
  if (!filename) {
    throw new Error("导出响应缺少文件名");
  }

  return {
    blob: response.data,
    filename,
    contentType: response.headers["content-type"] || "",
  };
}
