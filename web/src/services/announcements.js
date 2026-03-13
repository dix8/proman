import { http } from "./http";

export const ANNOUNCEMENT_STATUS_DRAFT = "draft";
export const ANNOUNCEMENT_STATUS_PUBLISHED = "published";

export async function fetchAnnouncements(projectId, params) {
  const response = await http.get(`/api/projects/${projectId}/announcements`, {
    params,
  });
  return response.data.data;
}

export async function fetchAnnouncement(announcementId) {
  const response = await http.get(`/api/announcements/${announcementId}`);
  return response.data.data;
}

export async function createAnnouncement(projectId, payload) {
  const response = await http.post(
    `/api/projects/${projectId}/announcements`,
    payload,
  );
  return response.data.data;
}

export async function updateAnnouncement(announcementId, payload) {
  const response = await http.put(
    `/api/announcements/${announcementId}`,
    payload,
  );
  return response.data.data;
}

export async function deleteAnnouncement(announcementId) {
  const response = await http.delete(`/api/announcements/${announcementId}`);
  return response.data.data;
}

export async function publishAnnouncement(announcementId) {
  const response = await http.put(
    `/api/announcements/${announcementId}/publish`,
  );
  return response.data.data;
}

export async function revokeAnnouncement(announcementId) {
  const response = await http.put(
    `/api/announcements/${announcementId}/revoke`,
  );
  return response.data.data;
}
