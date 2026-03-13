import React, { Suspense, lazy } from "react";
import { Spin } from "antd";
import { Navigate, createBrowserRouter } from "react-router-dom";

import { RequireAuth } from "./guards/RequireAuth";

function lazyImport(factory, exportName) {
  return lazy(() =>
    factory().then((module) => ({
      default: module[exportName],
    })),
  );
}

function RouteLoading() {
  return (
    <div className="loading-shell">
      <Spin size="large" tip="页面加载中..." />
    </div>
  );
}

function withSuspense(node) {
  return <Suspense fallback={<RouteLoading />}>{node}</Suspense>;
}

const AdminLayout = lazyImport(
  () => import("./layouts/AdminLayout"),
  "AdminLayout",
);
const AnnouncementsPage = lazyImport(
  () => import("../pages/AnnouncementsPage"),
  "AnnouncementsPage",
);
const AnnouncementEditorPage = lazyImport(
  () => import("../pages/AnnouncementEditorPage"),
  "AnnouncementEditorPage",
);
const ChangelogEditorPage = lazyImport(
  () => import("../pages/ChangelogEditorPage"),
  "ChangelogEditorPage",
);
const IntegrationGuidePage = lazyImport(
  () => import("../pages/IntegrationGuidePage"),
  "IntegrationGuidePage",
);
const LoginPage = lazyImport(() => import("../pages/LoginPage"), "LoginPage");
const ProjectDetailPage = lazyImport(
  () => import("../pages/ProjectDetailPage"),
  "ProjectDetailPage",
);
const ProjectsPage = lazyImport(
  () => import("../pages/ProjectsPage"),
  "ProjectsPage",
);
const VersionComparePage = lazyImport(
  () => import("../pages/VersionComparePage"),
  "VersionComparePage",
);
const VersionEditPage = lazyImport(
  () => import("../pages/VersionEditPage"),
  "VersionEditPage",
);
const VersionListPage = lazyImport(
  () => import("../pages/VersionListPage"),
  "VersionListPage",
);

export const router = createBrowserRouter([
  {
    path: "/login",
    element: withSuspense(<LoginPage />),
  },
  {
    path: "/",
    element: (
      <RequireAuth>
        {withSuspense(<AdminLayout />)}
      </RequireAuth>
    ),
    children: [
      {
        index: true,
        element: <Navigate to="/projects" replace />,
      },
      {
        path: "projects",
        element: withSuspense(<ProjectsPage />),
      },
      {
        path: "projects/:projectId",
        element: withSuspense(<ProjectDetailPage />),
      },
      {
        path: "projects/:projectId/versions",
        element: withSuspense(<VersionListPage />),
      },
      {
        path: "projects/:projectId/versions/new",
        element: withSuspense(<VersionEditPage />),
      },
      {
        path: "projects/:projectId/versions/:versionId/edit",
        element: withSuspense(<VersionEditPage />),
      },
      {
        path: "projects/:projectId/versions/:versionId/changelogs",
        element: withSuspense(<ChangelogEditorPage />),
      },
      {
        path: "versions/:versionId/edit",
        element: withSuspense(<VersionEditPage />),
      },
      {
        path: "versions/:versionId/changelogs",
        element: withSuspense(<ChangelogEditorPage />),
      },
      {
        path: "versions/compare",
        element: withSuspense(<VersionComparePage />),
      },
      {
        path: "integration",
        element: withSuspense(<IntegrationGuidePage />),
      },
      {
        path: "announcements",
        element: withSuspense(<AnnouncementsPage />),
      },
      {
        path: "announcements/new",
        element: withSuspense(<AnnouncementEditorPage />),
      },
      {
        path: "announcements/:announcementId/edit",
        element: withSuspense(<AnnouncementEditorPage />),
      },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/" replace />,
  },
]);
