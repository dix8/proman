import { ArrowLeftOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Descriptions,
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

import { AnnouncementForm } from "../components/AnnouncementForm";
import { useIsMobile } from "../hooks/useIsMobile";
import { fetchProjects, fetchProject } from "../services/projects";
import {
  ANNOUNCEMENT_STATUS_PUBLISHED,
  createAnnouncement,
  fetchAnnouncement,
  updateAnnouncement,
} from "../services/announcements";

const { Paragraph, Title, Text } = Typography;

function formatTimestamp(value) {
  return value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "未发布";
}

export function AnnouncementEditorPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const isMobile = useIsMobile();
  const { announcementId } = useParams();
  const [searchParams] = useSearchParams();
  const [messageApi, contextHolder] = message.useMessage();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [projects, setProjects] = useState([]);
  const [project, setProject] = useState(null);
  const [announcement, setAnnouncement] = useState(null);
  const [missing, setMissing] = useState(false);

  const isCreateMode = !announcementId;
  const queryProjectId =
    searchParams.get("projectId") || location.state?.projectId || "";
  const activeProjectId = announcement?.project_id
    ? String(announcement.project_id)
    : queryProjectId;
  const isPublished = announcement?.status === ANNOUNCEMENT_STATUS_PUBLISHED;

  useEffect(() => {
    let cancelled = false;

    async function loadPage() {
      setLoading(true);
      setMissing(false);

      try {
        const projectsData = await fetchProjects({
          page: 1,
          page_size: 100,
        });

        if (cancelled) {
          return;
        }

        setProjects(projectsData.list);

        if (isCreateMode) {
          if (projectsData.list.length === 0) {
            setProject(null);
            form.setFieldsValue({
              projectId: undefined,
              title: "",
              content: "",
              is_pinned: false,
            });
            return;
          }

          const preferredProject =
            projectsData.list.find(
              (item) => String(item.id) === queryProjectId,
            ) || projectsData.list[0];

          setProject(preferredProject);
          form.setFieldsValue({
            projectId: String(preferredProject.id),
            title: "",
            content: "",
            is_pinned: false,
          });
          return;
        }

        const announcementData = await fetchAnnouncement(announcementId);
        if (cancelled) {
          return;
        }

        setAnnouncement(announcementData);
        form.setFieldsValue({
          title: announcementData.title,
          content: announcementData.content,
          is_pinned: announcementData.is_pinned,
        });

        const matchedProject =
          projectsData.list.find(
            (item) => item.id === announcementData.project_id,
          ) || (await fetchProject(announcementData.project_id));

        if (!cancelled) {
          setProject(matchedProject);
        }
      } catch (error) {
        if (cancelled) {
          return;
        }

        if (error?.response?.status === 404) {
          setMissing(true);
        } else {
          messageApi.error(
            error?.response?.data?.message || "公告信息加载失败",
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
  }, [announcementId, form, isCreateMode, messageApi, queryProjectId]);

  async function handleSubmit(values) {
    setSubmitting(true);
    try {
      if (isCreateMode) {
        const created = await createAnnouncement(values.projectId, {
          title: values.title,
          content: values.content,
          is_pinned: values.is_pinned || false,
        });
        messageApi.success(`公告「${created.title}」创建成功`);
        navigate(
          `/announcements/${created.id}/edit?projectId=${created.project_id}`,
          { replace: true },
        );
        return;
      }

      const updated = await updateAnnouncement(announcementId, {
        title: values.title,
        content: values.content,
        is_pinned: values.is_pinned || false,
      });

      setAnnouncement(updated);
      form.setFieldsValue({
        title: updated.title,
        content: updated.content,
        is_pinned: updated.is_pinned,
      });
      messageApi.success(`公告「${updated.title}」已保存`);
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "公告保存失败");
    } finally {
      setSubmitting(false);
    }
  }

  if (loading) {
    return (
      <>
        {contextHolder}
        <div className="loading-shell">
          <Spin size="large" tip="加载公告信息..." />
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
            title={isCreateMode ? "项目不存在" : "公告不存在"}
            subTitle={
              isCreateMode
                ? "当前项目不存在，无法继续创建公告。"
                : "当前公告可能已被删除，或你没有权限访问它。"
            }
            extra={
              <Button
                type="primary"
                onClick={() =>
                  navigate(
                    activeProjectId
                      ? `/announcements?projectId=${activeProjectId}`
                      : "/announcements",
                  )
                }
              >
                返回公告列表
              </Button>
            }
          />
        </Card>
      </>
    );
  }

  if (isCreateMode && projects.length === 0) {
    return (
      <>
        {contextHolder}
        <Card className="placeholder-card" bordered={false}>
          <Result
            status="info"
            title="暂无项目可创建公告"
            subTitle="公告必须归属于项目，请先创建项目。"
            extra={
              <Button type="primary" onClick={() => navigate("/projects")}>
                前往项目管理
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
                onClick={() =>
                  navigate(
                    activeProjectId
                      ? `/announcements?projectId=${activeProjectId}`
                      : "/announcements",
                  )
                }
              >
                返回公告列表
              </Button>
              <Title level={2} style={{ marginBottom: 8 }}>
                {isCreateMode
                  ? "新建公告"
                  : `编辑公告：${announcement?.title || ""}`}
              </Title>
              <Paragraph className="placeholder-description">
                {isCreateMode
                  ? "创建公告时先选择所属项目，默认创建为草稿。"
                  : "公告支持草稿编辑，也支持在 published 状态下继续编辑。"}
              </Paragraph>
            </div>
          </div>

          {isPublished ? (
            <Alert
              type="info"
              showIcon
              message="当前公告已发布。你仍可继续编辑并保存，保存后状态仍保持 published，published_at 不会因此回退。"
            />
          ) : null}

          {!isCreateMode && !isPublished ? (
            <Alert
              type="warning"
              showIcon
              message="当前公告为草稿。发布和撤回操作请在公告列表页完成。"
            />
          ) : null}

          <Card className="inner-card" bordered={false}>
            <Space direction="vertical" size={20} style={{ width: "100%" }}>
              <div>
                <Text strong>所属项目：</Text>
                <Text>{project?.name || "-"}</Text>
                {!isCreateMode ? (
                  <Space style={{ marginLeft: 16 }}>
                    {isPublished ? (
                      <Tag color="success">已发布</Tag>
                    ) : (
                      <Tag color="processing">草稿</Tag>
                    )}
                  </Space>
                ) : null}
              </div>

              <AnnouncementForm
                key={announcementId || `new-${activeProjectId || "default"}`}
                form={form}
                projects={projects}
                createMode={isCreateMode}
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
                      label: "当前状态",
                      children: announcement?.status || "-",
                    },
                    {
                      key: "published_at",
                      label: "发布时间",
                      children: formatTimestamp(announcement?.published_at),
                    },
                    {
                      key: "created_at",
                      label: "创建时间",
                      children: formatTimestamp(announcement?.created_at),
                    },
                    {
                      key: "updated_at",
                      label: "更新时间",
                      children: formatTimestamp(announcement?.updated_at),
                    },
                  ]}
                />
              ) : null}

              <Space
                wrap
                direction={isMobile ? "vertical" : "horizontal"}
                className={isMobile ? "mobile-list-actions" : undefined}
              >
                <Button
                  type="primary"
                  block={isMobile}
                  data-testid="announcement-submit-button"
                  loading={submitting}
                  onClick={() => form.submit()}
                >
                  {isCreateMode ? "创建公告" : "保存公告"}
                </Button>
                <Button
                  block={isMobile}
                  onClick={() =>
                    navigate(
                      activeProjectId
                        ? `/announcements?projectId=${activeProjectId}`
                        : "/announcements",
                    )
                  }
                >
                  返回列表
                </Button>
              </Space>
            </Space>
          </Card>
        </Space>
      </Card>
    </>
  );
}
