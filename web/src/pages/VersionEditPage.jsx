import { ArrowLeftOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Popconfirm,
  Result,
  Space,
  Spin,
  Tag,
  Typography,
  Form,
  message,
} from "antd";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import {
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";

import { VersionForm } from "../components/VersionForm";
import { useIsMobile } from "../hooks/useIsMobile";
import { fetchProject } from "../services/projects";
import {
  VERSION_STATUS_PUBLISHED,
  createVersion,
  fetchVersion,
  unpublishVersion,
  updateVersion,
} from "../services/versions";

const { Paragraph, Title, Text } = Typography;

function formatTimestamp(value) {
  return value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "未发布";
}

export function VersionEditPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const isMobile = useIsMobile();
  const { projectId: routeProjectId, versionId } = useParams();
  const [searchParams] = useSearchParams();
  const [messageApi, contextHolder] = message.useMessage();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [project, setProject] = useState(null);
  const [version, setVersion] = useState(null);
  const [missing, setMissing] = useState(false);

  const projectId = useMemo(
    () =>
      routeProjectId ||
      searchParams.get("projectId") ||
      location.state?.projectId ||
      "",
    [location.state?.projectId, routeProjectId, searchParams],
  );
  const isCreateMode = !versionId;
  const isReadOnly = version?.status === VERSION_STATUS_PUBLISHED;

  useEffect(() => {
    let cancelled = false;

    async function loadPage() {
      if (!projectId) {
        setMissing(true);
        setLoading(false);
        return;
      }

      setLoading(true);
      setMissing(false);

      try {
        const tasks = [fetchProject(projectId)];
        if (!isCreateMode) {
          tasks.push(fetchVersion(versionId));
        }

        const [projectData, versionData] = await Promise.all(tasks);
        if (cancelled) {
          return;
        }

        setProject(projectData);
        if (!isCreateMode) {
          setVersion(versionData);
          form.setFieldsValue({
            major: versionData.major,
            minor: versionData.minor,
            patch: versionData.patch,
            url: versionData.url || "",
          });
        } else {
          setVersion(null);
          form.setFieldsValue({ major: 1, minor: 0, patch: 0, url: "" });
        }
      } catch (error) {
        if (cancelled) {
          return;
        }

        if (error?.response?.status === 404) {
          setMissing(true);
        } else {
          messageApi.error(
            error?.response?.data?.message || "版本信息加载失败",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadPage();

    return () => {
      cancelled = true;
    };
  }, [form, isCreateMode, messageApi, projectId, versionId]);

  async function handleSubmit(values) {
    setSubmitting(true);
    try {
      if (isCreateMode) {
        const created = await createVersion(projectId, values);
        messageApi.success(`版本 ${created.version} 创建成功`);
        navigate(`/projects/${projectId}/versions/${created.id}/edit`, {
          replace: true,
        });
        return;
      }

      const updated = await updateVersion(versionId, values);
      setVersion(updated);
      form.setFieldsValue({
        major: updated.major,
        minor: updated.minor,
        patch: updated.patch,
        url: updated.url || "",
      });
      messageApi.success(`版本 ${updated.version} 已更新`);
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "版本保存失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleUnpublish() {
    try {
      const updated = await unpublishVersion(versionId);
      setVersion(updated);
      form.setFieldsValue({
        major: updated.major,
        minor: updated.minor,
        patch: updated.patch,
        url: updated.url || "",
      });
      messageApi.success(`版本 ${updated.version} 已撤回发布`);
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "撤回发布失败");
    }
  }

  if (loading) {
    return (
      <>
        {contextHolder}
        <div className="loading-shell">
          <Spin size="large" tip="加载版本信息..." />
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
            title={isCreateMode ? "项目不存在" : "版本不存在"}
            subTitle={
              isCreateMode
                ? "当前项目不存在，无法继续创建版本。"
                : "当前版本可能已被删除，或你没有权限访问它。"
            }
            extra={
              <Button
                type="primary"
                onClick={() =>
                  navigate(
                    projectId ? `/projects/${projectId}/versions` : "/projects",
                  )
                }
              >
                返回版本列表
              </Button>
            }
          />
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
                style={{ paddingInline: 0 }}
                onClick={() => navigate(`/projects/${projectId}/versions`)}
              >
                返回版本列表
              </Button>
              <Title level={2} style={{ marginBottom: 8 }}>
                {isCreateMode
                  ? `为 ${project?.name || "项目"} 创建版本`
                  : `版本 ${version?.version || ""}`}
              </Title>
              <Paragraph className="placeholder-description">
                {isCreateMode
                  ? "填写 major、minor、patch 后创建草稿版本。"
                  : "这里用于编辑草稿版本或查看已发布版本详情，并进入日志管理页面。"}
              </Paragraph>
            </div>
            {!isCreateMode ? (
              <Button
                type="primary"
                block={isMobile}
                data-testid="version-manage-changelogs-button"
                onClick={() =>
                  navigate(
                    `/projects/${projectId}/versions/${versionId}/changelogs`,
                  )
                }
              >
                管理日志
              </Button>
            ) : null}
          </div>

          {!isCreateMode && isReadOnly ? (
            <Alert
              type="warning"
              showIcon
              message="当前版本已发布，处于只读状态。如需修改可撤回发布恢复为草稿。"
              action={
                <Popconfirm
                  title="确认撤回发布"
                  description={`撤回后版本 ${version?.version} 将恢复为草稿状态，可继续编辑。`}
                  okText="确认撤回"
                  cancelText="取消"
                  onConfirm={handleUnpublish}
                >
                  <Button size="small">撤回发布</Button>
                </Popconfirm>
              }
            />
          ) : null}

          <Card className="inner-card" bordered={false}>
            <Space direction="vertical" size={20} style={{ width: "100%" }}>
              <div>
                <Text strong>所属项目：</Text>
                <Text>{project?.name || "-"}</Text>
                {!isCreateMode ? (
                  <Space style={{ marginLeft: 16 }}>
                    {isReadOnly ? (
                      <Tag color="success">已发布 / 只读</Tag>
                    ) : (
                      <Tag color="processing">草稿</Tag>
                    )}
                  </Space>
                ) : null}
              </div>

              <VersionForm
                form={form}
                disabled={isReadOnly}
                onFinish={handleSubmit}
              />

              {!isCreateMode ? (
                <Descriptions
                  bordered
                  size="small"
                  column={1}
                  items={[
                    {
                      key: "status",
                      label: "版本状态",
                      children: isReadOnly ? "published" : "draft",
                    },
                    {
                      key: "url",
                      label: "发布地址",
                      children: version?.url ? (
                        <a href={version.url} target="_blank" rel="noopener noreferrer">
                          {version.url}
                        </a>
                      ) : (
                        <Text type="secondary">未设置</Text>
                      ),
                    },
                    {
                      key: "published_at",
                      label: "发布时间",
                      children: formatTimestamp(version?.published_at),
                    },
                    {
                      key: "created_at",
                      label: "创建时间",
                      children: formatTimestamp(version?.created_at),
                    },
                    {
                      key: "updated_at",
                      label: "更新时间",
                      children: formatTimestamp(version?.updated_at),
                    },
                  ]}
                />
              ) : null}

              <Space
                wrap
                direction={isMobile ? "vertical" : "horizontal"}
                className={isMobile ? "mobile-list-actions" : undefined}
              >
                {!isReadOnly ? (
                  <Button
                    type="primary"
                    block={isMobile}
                    data-testid="version-submit-button"
                    loading={submitting}
                    onClick={() => form.submit()}
                  >
                    {isCreateMode ? "创建版本" : "保存修改"}
                  </Button>
                ) : null}
                {!isCreateMode ? (
                  <Button
                    block={isMobile}
                    data-testid="version-open-changelogs-button"
                    onClick={() =>
                      navigate(
                        `/projects/${projectId}/versions/${versionId}/changelogs`,
                      )
                    }
                  >
                    {isReadOnly ? "查看日志" : "进入日志管理"}
                  </Button>
                ) : null}
              </Space>
            </Space>
          </Card>
        </Space>
      </Card>
    </>
  );
}
