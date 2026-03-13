import { http } from "./http";

export async function fetchMarkdownPreview(content) {
  const response = await http.post("/api/markdown/preview", { content });
  return response.data.data;
}
