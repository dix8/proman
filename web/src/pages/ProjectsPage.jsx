import {
  Button,
  Card,
  Empty,
  Input,
  Pagination,
  Popconfirm,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import dayjs from "dayjs";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { ProjectFormModal } from "../components/ProjectFormModal";
import { TokenRevealModal } from "../components/TokenRevealModal";
import { TokenRefreshConfirmButton } from "../components/TokenRefreshConfirmButton";
import { useIsMobile } from "../hooks/useIsMobile";
import {
  createProject,
  deleteProject,
  fetchProjects,
  refreshProjectToken,
  updateProject,
} from "../services/projects";

const { Paragraph, Title, Text } = Typography;

export function ProjectsPage() {
  const navigate = useNavigate();
  const isMobile = useIsMobile();
  const [messageApi, contextHolder] = message.useMessage();
  const [loading, setLoading] = useState(false);
  const [projects, setProjects] = useState([]);
  const [total, setTotal] = useState(0);
  const [keywordInput, setKeywordInput] = useState("");
  const [query, setQuery] = useState({ page: 1, pageSize: 5, keyword: "" });
  const [formModal, setFormModal] = useState({
    open: false,
    mode: "create",
    project: null,
  });
  const [formSubmitting, setFormSubmitting] = useState(false);
  const [tokenModal, setTokenModal] = useState({
    open: false,
    title: "",
    description: "",
    warning: "",
    token: "",
  });

  useEffect(() => {
    void loadProjects(query);
  }, [query.page, query.pageSize, query.keyword]);

  async function loadProjects(nextQuery = query) {
    setLoading(true);
    try {
      const data = await fetchProjects({
        page: nextQuery.page,
        page_size: nextQuery.pageSize,
        keyword: nextQuery.keyword || undefined,
      });
      setProjects(data.list);
      setTotal(data.total);
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "项目列表加载失败");
    } finally {
      setLoading(false);
    }
  }

  async function syncQuery(nextQuery) {
    const isSameQuery =
      nextQuery.page === query.page &&
      nextQuery.pageSize === query.pageSize &&
      nextQuery.keyword === query.keyword;

    setQuery(nextQuery);
    if (isSameQuery) {
      await loadProjects(nextQuery);
    }
  }

  async function applySearch(keyword) {
    const nextQuery = {
      ...query,
      page: 1,
      keyword,
    };
    await syncQuery(nextQuery);
  }

  function openCreateModal() {
    setFormModal({ open: true, mode: "create", project: null });
  }

  function openEditModal(project) {
    setFormModal({ open: true, mode: "edit", project });
  }

  async function handleSubmit(values) {
    setFormSubmitting(true);
    try {
      if (formModal.mode === "create") {
        const data = await createProject(values);
        messageApi.success("项目创建成功");
        setFormModal({ open: false, mode: "create", project: null });
        const nextQuery = { ...query, page: 1 };
        await syncQuery(nextQuery);
        setTokenModal({
          open: true,
          title: "项目 Token（仅显示一次）",
          description:
            "项目创建成功，以下 Token 只会显示这一次，请立即保存到安全位置。",
          warning: "关闭后页面不会再次展示明文 Token。",
          token: data.project_token,
        });
        return;
      }

      await updateProject(formModal.project.id, values);
      messageApi.success("项目已更新");
      setFormModal({ open: false, mode: "create", project: null });
      await loadProjects();
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "项目保存失败");
    } finally {
      setFormSubmitting(false);
    }
  }

  async function handleDelete(project) {
    try {
      await deleteProject(project.id);
      messageApi.success("项目已删除");
      if (projects.length === 1 && query.page > 1) {
        setQuery((current) => ({ ...current, page: current.page - 1 }));
      } else {
        await loadProjects();
      }
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "项目删除失败");
    }
  }

  async function handleRefreshToken(project) {
    try {
      const data = await refreshProjectToken(project.id);
      await loadProjects();
      messageApi.success("新 Token 已生成，旧 Token 已立即失效，请尽快更新调用方配置");
      setTokenModal({
        open: true,
        title: "新项目 Token（仅显示一次）",
        description: `项目「${project.name}」Token 已刷新，新 Token 只会显示这一次。`,
        warning: "旧 Token 已立即失效，请尽快通知调用方更新配置。",
        token: data.project_token,
      });
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "Token 刷新失败");
    }
  }

  const columns = [
    {
      title: "项目名称",
      dataIndex: "name",
      key: "name",
      render: (_, record) => (
        <Button
          type="link"
          onClick={() => navigate(`/projects/${record.id}`)}
          style={{ padding: 0 }}
        >
          {record.name}
        </Button>
      ),
    },
    {
      title: "描述",
      dataIndex: "description",
      key: "description",
      render: (value) => value || <Tag color="default">无描述</Tag>,
    },
    {
      title: "Token 更新时间",
      dataIndex: "token_updated_at",
      key: "token_updated_at",
      render: (value) => dayjs(value).format("YYYY-MM-DD HH:mm:ss"),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      render: (value) => dayjs(value).format("YYYY-MM-DD HH:mm:ss"),
    },
    {
      title: "操作",
      key: "actions",
      width: 280,
      render: (_, record) => (
        <Space wrap>
          <Button
            size="small"
            data-testid={`project-edit-${record.id}`}
            onClick={() => openEditModal(record)}
          >
            编辑
          </Button>
          <Button
            size="small"
            onClick={() => navigate(`/projects/${record.id}`)}
          >
            详情
          </Button>
          <TokenRefreshConfirmButton
            size="small"
            data-testid={`project-refresh-token-${record.id}`}
            onConfirm={() => handleRefreshToken(record)}
          >
            刷新 Token
          </TokenRefreshConfirmButton>
          <Popconfirm
            title="确认删除项目"
            description={`删除后，项目「${record.name}」及其关联数据将一起软删除。此操作不可撤销。`}
            okText="确认删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={() => handleDelete(record)}
          >
            <Button
              size="small"
              danger
              data-testid={`project-delete-${record.id}`}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      {contextHolder}
      <Card className="placeholder-card" bordered={false}>
        <Space direction="vertical" size={20} style={{ width: "100%" }}>
          <div className="page-toolbar">
            <div>
              <Title level={2} style={{ marginBottom: 8 }}>
                项目列表
              </Title>
              <Paragraph className="placeholder-description">
                支持项目分页、关键词搜索、创建、编辑、删除和 Token
                刷新。本轮已经把项目模块主流程接到真实后端。
              </Paragraph>
            </div>
            <Button type="primary" onClick={openCreateModal}>
              新建项目
            </Button>
          </div>

          <div className="table-toolbar">
            <Space.Compact block style={{ maxWidth: 420 }}>
              <Input
                allowClear
                placeholder="按项目名称搜索"
                value={keywordInput}
                onChange={(event) => setKeywordInput(event.target.value)}
                onPressEnter={() => applySearch(keywordInput.trim())}
              />
              <Button
                type="primary"
                data-testid="projects-search-button"
                onClick={() => applySearch(keywordInput.trim())}
              >
                搜索
              </Button>
              <Button
                data-testid="projects-reset-button"
                onClick={async () => {
                  setKeywordInput("");
                  await applySearch("");
                }}
              >
                重置
              </Button>
            </Space.Compact>
          </div>

          {!isMobile ? (
            <Table
              rowKey="id"
              loading={loading}
              columns={columns}
              dataSource={projects}
              locale={{
                emptyText: <Empty description="暂无项目数据" />,
              }}
              pagination={{
                current: query.page,
                pageSize: query.pageSize,
                total,
                showSizeChanger: true,
                pageSizeOptions: ["5", "10", "20", "50"],
                showTotal: (count) => `共 ${count} 项`,
                onChange: (page, pageSize) =>
                  setQuery((current) => ({
                    ...current,
                    page,
                    pageSize,
                  })),
              }}
            />
          ) : loading ? (
            <div className="page-inline-loading">
              <Spin tip="加载项目列表..." />
            </div>
          ) : projects.length > 0 ? (
            <>
              <div className="mobile-list">
                {projects.map((project) => (
                  <Card
                    key={project.id}
                    size="small"
                    className="mobile-list-card"
                  >
                    <Space
                      direction="vertical"
                      size={12}
                      style={{ width: "100%" }}
                    >
                      <Button
                        type="link"
                        style={{ padding: 0, textAlign: "left" }}
                        onClick={() => navigate(`/projects/${project.id}`)}
                      >
                        {project.name}
                      </Button>

                      <div className="mobile-list-meta">
                        <div className="mobile-list-row">
                          <Text className="mobile-list-label">描述</Text>
                          <Text className="mobile-list-value">
                            {project.description || "无描述"}
                          </Text>
                        </div>
                        <div className="mobile-list-row">
                          <Text className="mobile-list-label">
                            Token 更新时间
                          </Text>
                          <Text className="mobile-list-value">
                            {dayjs(project.token_updated_at).format(
                              "YYYY-MM-DD HH:mm:ss",
                            )}
                          </Text>
                        </div>
                        <div className="mobile-list-row">
                          <Text className="mobile-list-label">创建时间</Text>
                          <Text className="mobile-list-value">
                            {dayjs(project.created_at).format(
                              "YYYY-MM-DD HH:mm:ss",
                            )}
                          </Text>
                        </div>
                      </div>

                      <Space
                        direction="vertical"
                        size={8}
                        className="mobile-list-actions"
                      >
                        <Button
                          data-testid={`project-edit-${project.id}`}
                          onClick={() => openEditModal(project)}
                        >
                          编辑
                        </Button>
                        <Button
                          onClick={() => navigate(`/projects/${project.id}`)}
                        >
                          详情
                        </Button>
                        <TokenRefreshConfirmButton
                          data-testid={`project-refresh-token-${project.id}`}
                          onConfirm={() => handleRefreshToken(project)}
                        >
                          刷新 Token
                        </TokenRefreshConfirmButton>
                        <Popconfirm
                          title="确认删除项目"
                          description={`删除后，项目「${project.name}」及其关联数据将一起软删除。此操作不可撤销。`}
                          okText="确认删除"
                          cancelText="取消"
                          okButtonProps={{ danger: true }}
                          onConfirm={() => handleDelete(project)}
                        >
                          <Button
                            danger
                            data-testid={`project-delete-${project.id}`}
                          >
                            删除
                          </Button>
                        </Popconfirm>
                      </Space>
                    </Space>
                  </Card>
                ))}
              </div>
              <Pagination
                current={query.page}
                pageSize={query.pageSize}
                total={total}
                showSizeChanger
                pageSizeOptions={["5", "10", "20", "50"]}
                showTotal={(count) => `共 ${count} 项`}
                onChange={(page, pageSize) =>
                  setQuery((current) => ({
                    ...current,
                    page,
                    pageSize,
                  }))
                }
              />
            </>
          ) : (
            <Empty description="暂无项目数据" />
          )}
        </Space>
      </Card>

      <ProjectFormModal
        open={formModal.open}
        mode={formModal.mode}
        initialValues={
          formModal.project
            ? {
                name: formModal.project.name,
                description: formModal.project.description,
              }
            : undefined
        }
        loading={formSubmitting}
        onCancel={() =>
          setFormModal({ open: false, mode: "create", project: null })
        }
        onSubmit={handleSubmit}
      />

      <TokenRevealModal
        open={tokenModal.open}
        title={tokenModal.title}
        description={tokenModal.description}
        warning={tokenModal.warning}
        token={tokenModal.token}
        onClose={() =>
          setTokenModal({
            open: false,
            title: "",
            description: "",
            warning: "",
            token: "",
          })
        }
      />
    </>
  );
}
