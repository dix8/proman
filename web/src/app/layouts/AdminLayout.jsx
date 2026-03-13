import { MenuOutlined } from "@ant-design/icons";
import { Button, Drawer, Layout, Menu, Spin, Typography } from "antd";
import { useEffect, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router-dom";

import { adminMenuItems } from "../menu";
import { clearToken } from "../../services/auth";
import { http } from "../../services/http";
import { useIsMobile } from "../../hooks/useIsMobile";

const { Header, Sider, Content } = Layout;
const { Title, Text } = Typography;

function resolveSelectedKey(pathname) {
  if (pathname.startsWith("/announcements")) {
    return "/announcements";
  }
  if (pathname.startsWith("/integration")) {
    return "/integration";
  }
  if (pathname.startsWith("/versions/compare")) {
    return "/versions/compare";
  }
  return "/projects";
}

function resolvePageTitle(pathname) {
  if (pathname.startsWith("/projects/") && pathname.endsWith("/versions/new")) {
    return "创建版本";
  }
  if (pathname.startsWith("/projects/") && pathname.endsWith("/versions")) {
    return "版本列表";
  }
  if (
    pathname.startsWith("/projects/") &&
    pathname.includes("/versions/") &&
    pathname.endsWith("/edit")
  ) {
    return "版本编辑";
  }
  if (
    pathname.startsWith("/projects/") &&
    pathname.includes("/versions/") &&
    pathname.endsWith("/changelogs")
  ) {
    return "更新日志编辑";
  }
  if (pathname.startsWith("/projects/")) {
    return "项目详情";
  }
  if (pathname.startsWith("/versions/") && pathname.endsWith("/edit")) {
    return "版本编辑";
  }
  if (pathname.startsWith("/versions/") && pathname.endsWith("/changelogs")) {
    return "更新日志编辑";
  }
  if (pathname.startsWith("/versions/compare")) {
    return "版本对比";
  }
  if (pathname.startsWith("/integration")) {
    return "接口接入";
  }
  if (pathname.startsWith("/announcements/") && pathname.endsWith("/edit")) {
    return "公告编辑";
  }
  if (pathname.startsWith("/announcements/new")) {
    return "公告编辑";
  }
  if (pathname.startsWith("/announcements")) {
    return "公告管理";
  }
  return "项目管理";
}

export function AdminLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const isMobile = useIsMobile();
  const [checkingSession, setCheckingSession] = useState(true);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const selectedKey = resolveSelectedKey(location.pathname);
  const pageTitle = resolvePageTitle(location.pathname);
  const menuItems = adminMenuItems.map((item) => ({
    key: item.key,
    icon: <item.icon />,
    label: item.label,
  }));

  function handleLogout() {
    clearToken();
    navigate("/login", { replace: true });
  }

  useEffect(() => {
    let cancelled = false;

    async function verifySession() {
      try {
        await http.get("/api/projects", {
          params: {
            page: 1,
            page_size: 1,
          },
        });
      } catch (error) {
        if (
          error?.response?.status !== 401 ||
          error?.response?.data?.code !== 40102
        ) {
          // Leave non-auth errors to the business pages later.
        }
      } finally {
        if (!cancelled) {
          setCheckingSession(false);
        }
      }
    }

    verifySession();

    return () => {
      cancelled = true;
    };
  }, []);

  if (checkingSession) {
    return (
      <div className="loading-shell">
        <Spin size="large" tip="校验登录状态..." />
      </div>
    );
  }

  return (
    <Layout className="admin-shell">
      {!isMobile ? (
        <Sider width={250} className="admin-sider" theme="light">
          <div className="admin-sider-inner">
            <div className="brand-block">
              <Text className="brand-kicker">Project Management</Text>
              <Title level={4} className="brand-title">
                Proman Admin
              </Title>
            </div>
            <Menu
              mode="inline"
              selectedKeys={[selectedKey]}
              items={menuItems}
              className="admin-side-menu"
              onClick={({ key }) => navigate(key)}
            />
            <div className="sidebar-logout">
              <Button block onClick={handleLogout}>
                退出登录
              </Button>
            </div>
          </div>
        </Sider>
      ) : null}
      <Layout>
        <Header className="admin-header">
          <div className="header-main">
            {isMobile ? (
              <Button
                type="text"
                icon={<MenuOutlined />}
                className="mobile-nav-trigger"
                onClick={() => setMobileMenuOpen(true)}
              />
            ) : null}
            <div>
              <Text className="header-kicker">后台控制台</Text>
              <Title level={3} className="header-title">
                {pageTitle}
              </Title>
            </div>
          </div>
        </Header>
        <Content className="admin-content">
          <Outlet />
        </Content>
      </Layout>
      {isMobile ? (
        <Drawer
          placement="left"
          width={280}
          open={mobileMenuOpen}
          onClose={() => setMobileMenuOpen(false)}
          title="Proman Admin"
          className="mobile-nav-drawer"
        >
          <div className="drawer-nav-content">
            <Menu
              mode="inline"
              selectedKeys={[selectedKey]}
              items={menuItems}
              className="drawer-nav-menu"
              onClick={({ key }) => {
                setMobileMenuOpen(false);
                navigate(key);
              }}
            />
            <div className="sidebar-logout">
              <Button
                block
                onClick={() => {
                  setMobileMenuOpen(false);
                  handleLogout();
                }}
              >
                退出登录
              </Button>
            </div>
          </div>
        </Drawer>
      ) : null}
    </Layout>
  );
}
