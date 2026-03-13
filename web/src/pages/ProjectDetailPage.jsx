import { ArrowLeftOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Descriptions,
  Empty,
  Result,
  Space,
  Spin,
  Typography,
  message,
} from "antd";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useIsMobile } from "../hooks/useIsMobile";
import { ProjectFormModal } from "../components/ProjectFormModal";
import { TokenRevealModal } from "../components/TokenRevealModal";
import { TokenRefreshConfirmButton } from "../components/TokenRefreshConfirmButton";
import { http } from "../services/http";
import {
  fetchProject,
  refreshProjectToken,
  updateProject,
} from "../services/projects";
import {
  buildPublicApiIntegrationEndpoints,
  buildTokenSnippetGroups,
  resolvePublicAPIBaseURL,
} from "../utils/publicApiIntegration";

const { Paragraph, Title } = Typography;

export function ProjectDetailPage() {
  const navigate = useNavigate();
  const { projectId } = useParams();
  const isMobile = useIsMobile();
  const [messageApi, contextHolder] = message.useMessage();
  const [loading, setLoading] = useState(true);
  const [project, setProject] = useState(null);
  const [missing, setMissing] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [formSubmitting, setFormSubmitting] = useState(false);
  const [tokenModal, setTokenModal] = useState({
    open: false,
    title: "",
    description: "",
    warning: "",
    token: "",
    snippetGroups: [],
  });
  const publicAPIBaseURL = useMemo(() => {
    const fallbackOrigin =
      typeof window !== "undefined" ? window.location.origin : "";
    return resolvePublicAPIBaseURL(
      http.defaults.baseURL,
      fallbackOrigin,
      import.meta.env.VITE_PUBLIC_BASE_URL,
    );
  }, []);
  const integrationEndpoints = useMemo(
    () => buildPublicApiIntegrationEndpoints(publicAPIBaseURL),
    [publicAPIBaseURL],
  );

  useEffect(() => {
    loadProject();
  }, [projectId]);

  async function loadProject() {
    setLoading(true);
    setMissing(false);
    try {
      const data = await fetchProject(projectId);
      setProject(data);
    } catch (error) {
      if (error?.response?.status === 404) {
        setMissing(true);
      } else {
        messageApi.error(error?.response?.data?.message || "项目详情加载失败");
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleEdit(values) {
    setFormSubmitting(true);
    try {
      const data = await updateProject(projectId, values);
      setProject(data);
      setFormOpen(false);
      messageApi.success("项目已更新");
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "项目更新失败");
    } finally {
      setFormSubmitting(false);
    }
  }

  async function handleRefreshToken() {
    try {
      const data = await refreshProjectToken(projectId);
      await loadProject();
      messageApi.success("新 Token 已生成，旧 Token 已立即失效，请尽快更新调用方配置");
      setTokenModal({
        open: true,
        title: "新项目 Token（仅显示一次）",
        description: `项目「${project?.name || ""}」Token 已刷新，新 Token 只会显示这一次。`,
        warning: "旧 Token 已立即失效，请尽快通知调用方更新配置。",
        token: data.project_token,
        snippetGroups: buildTokenSnippetGroups(
          data.project_token,
          integrationEndpoints,
        ),
      });
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "Token 刷新失败");
    }
  }

  if (loading) {
    return (
      <>
        {contextHolder}
        <div className="loading-shell">
          <Spin size="large" tip="加载项目详情..." />
        </div>
      </>
    );
  }

  if (missing) {
    return (
      <>
        {contextHolder}
        <Card className="placeholder-card" bordered={false}>
          <Result
            status="404"
            title="项目不存在"
            subTitle="当前项目可能已被删除，或你没有权限访问它。"
            extra={
              <Button type="primary" onClick={() => navigate("/projects")}>
                返回项目列表
              </Button>
            }
          />
        </Card>
      </>
    );
  }

  if (!project) {
    return (
      <>
        {contextHolder}
        <Card className="placeholder-card" bordered={false}>
          <Empty description="无法加载项目详情" />
        </Card>
      </>
    );
  }

  return (
    <>
      {contextHolder}
      <Card className="placeholder-card" bordered={false}>
        <Space direction="vertical" size={20} style={{ width: "100%" }}>
          <div className="page-toolbar">
            <div>
              <Button
                type="link"
                icon={<ArrowLeftOutlined />}
                onClick={() => navigate("/projects")}
                style={{ paddingInline: 0 }}
              >
                返回项目列表
              </Button>
              <Title level={2} style={{ marginBottom: 8 }}>
                {project.name}
              </Title>
              <Paragraph className="placeholder-description">
                这里展示项目基础信息，并提供编辑、版本入口和 Token
                刷新。项目公开接口的长期接入说明已收敛到全局“接口接入”页。
              </Paragraph>
            </div>
            <Space
              wrap
              direction={isMobile ? "vertical" : "horizontal"}
              className={isMobile ? "mobile-list-actions" : undefined}
            >
              <Button onClick={() => setFormOpen(true)}>编辑项目</Button>
              <Button
                onClick={() => navigate(`/projects/${projectId}/versions`)}
              >
                查看版本
              </Button>
              <Button onClick={() => navigate("/integration")}>
                查看接口接入说明
              </Button>
              <TokenRefreshConfirmButton
                type="primary"
                onConfirm={handleRefreshToken}
              >
                刷新 Token
              </TokenRefreshConfirmButton>
            </Space>
          </div>

          <Descriptions
            bordered
            column={1}
            items={[
              {
                key: "name",
                label: "项目名称",
                children: project.name,
              },
              {
                key: "description",
                label: "项目描述",
                children: project.description || "无描述",
              },
              {
                key: "token_updated_at",
                label: "Token 更新时间",
                children: dayjs(project.token_updated_at).format(
                  "YYYY-MM-DD HH:mm:ss",
                ),
              },
              {
                key: "created_at",
                label: "创建时间",
                children: dayjs(project.created_at).format(
                  "YYYY-MM-DD HH:mm:ss",
                ),
              },
              {
                key: "updated_at",
                label: "更新时间",
                children: dayjs(project.updated_at).format(
                  "YYYY-MM-DD HH:mm:ss",
                ),
              },
            ]}
          />
        </Space>
      </Card>

      <ProjectFormModal
        open={formOpen}
        mode="edit"
        initialValues={{
          name: project.name,
          description: project.description,
        }}
        loading={formSubmitting}
        onCancel={() => setFormOpen(false)}
        onSubmit={handleEdit}
      />

      <TokenRevealModal
        open={tokenModal.open}
        title={tokenModal.title}
        description={tokenModal.description}
        warning={tokenModal.warning}
        token={tokenModal.token}
        snippetGroups={tokenModal.snippetGroups}
        onClose={() =>
          setTokenModal({
            open: false,
            title: "",
            description: "",
            warning: "",
            token: "",
            snippetGroups: [],
          })
        }
      />
    </>
  );
}
