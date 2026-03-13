export const TOKEN_PLACEHOLDER = "<project_token>";
export const VERSION_PLACEHOLDER = "{version}";

export function trimTrailingSlash(value) {
  return String(value || "").replace(/\/+$/, "");
}

export function resolvePublicAPIBaseURL(
  requestBaseURL,
  fallbackOrigin = "",
  explicitPublicBaseURL = "",
) {
  const explicitBase = trimTrailingSlash(explicitPublicBaseURL);
  if (explicitBase !== "") {
    return explicitBase;
  }

  const normalizedRequestBaseURL = String(requestBaseURL || "").trim();
  if (
    normalizedRequestBaseURL === "" ||
    normalizedRequestBaseURL === "/" ||
    normalizedRequestBaseURL.startsWith("/")
  ) {
    return trimTrailingSlash(fallbackOrigin);
  }

  return trimTrailingSlash(normalizedRequestBaseURL);
}

export function buildPublicURL(baseURL, path) {
  return `${trimTrailingSlash(baseURL)}${path}`;
}

export function buildCurlExample(url, token = TOKEN_PLACEHOLDER) {
  return `curl -X GET "${url}" \\
  -H "Authorization: Bearer ${token}"`;
}

export function buildFetchExample(url, token = TOKEN_PLACEHOLDER) {
  return `fetch("${url}", {
  method: "GET",
  headers: {
    Authorization: "Bearer ${token}",
  },
});`;
}

export function buildPublicApiIntegrationEndpoints(baseURL) {
  const resolvedBaseURL = resolvePublicAPIBaseURL(baseURL);
  const definitions = [
    {
      key: "project",
      title: "当前项目信息",
      method: "GET",
      path: "/v1/project",
      description: "获取当前项目的公开基础信息。",
    },
    {
      key: "versions",
      title: "已发布版本列表",
      method: "GET",
      path: "/v1/versions",
      description: "获取当前项目全部已发布版本，支持分页参数。",
    },
    {
      key: "changelogs",
      title: "指定版本日志",
      method: "GET",
      path: `/v1/versions/${VERSION_PLACEHOLDER}/changelogs`,
      description:
        "按版本号获取指定已发布版本的完整日志，`{version}` 请替换为如 `1.2.3` 的真实版本号。",
    },
    {
      key: "announcements",
      title: "已发布公告列表",
      method: "GET",
      path: "/v1/announcements",
      description: "获取当前项目全部已发布公告，支持分页参数。",
    },
  ];

  return definitions.map((endpoint) => {
    const fullURL = buildPublicURL(resolvedBaseURL, endpoint.path);
    return {
      ...endpoint,
      fullURL,
      curlExample: buildCurlExample(fullURL),
      fetchExample: buildFetchExample(fullURL),
    };
  });
}

export function buildTokenSnippetGroups(token, endpoints) {
  return [
    {
      title: "带当前新 Token 的接入示例",
      description:
        "这些示例仅用于当前项目公开接口接入，页面长期仍只展示占位符版本。",
      items: [
        {
          label: "鉴权 Header",
          value: `Authorization: Bearer ${token}`,
        },
        ...endpoints.map((endpoint) => ({
          label: `${endpoint.method} ${endpoint.path} curl`,
          value: buildCurlExample(endpoint.fullURL, token),
        })),
      ],
    },
  ];
}

export async function writeToClipboard(text) {
  if (navigator?.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }

  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "true");
  textarea.style.position = "absolute";
  textarea.style.left = "-9999px";
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand("copy");
  document.body.removeChild(textarea);
}
