import { http } from "./http";

export async function fetchProjects(params) {
  const response = await http.get("/api/projects", { params });
  return response.data.data;
}

export async function fetchProject(projectId) {
  const response = await http.get(`/api/projects/${projectId}`);
  return response.data.data;
}

export async function createProject(payload) {
  const response = await http.post("/api/projects", payload);
  return response.data.data;
}

export async function updateProject(projectId, payload) {
  const response = await http.put(`/api/projects/${projectId}`, payload);
  return response.data.data;
}

export async function deleteProject(projectId) {
  const response = await http.delete(`/api/projects/${projectId}`);
  return response.data.data;
}

export async function refreshProjectToken(projectId) {
  const response = await http.post(`/api/projects/${projectId}/token/refresh`);
  return response.data.data;
}
