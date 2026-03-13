import {
  Alert,
  Button,
  Card,
  Empty,
  Input,
  Pagination,
  Popconfirm,
  Result,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import { fetchProjects } from "../services/projects";
import { useIsMobile } from "../hooks/useIsMobile";
import {
  ANNOUNCEMENT_STATUS_DRAFT,
  ANNOUNCEMENT_STATUS_PUBLISHED,
  deleteAnnouncement,
  fetchAnnouncements,
  publishAnnouncement,
  revokeAnnouncement,
} from "../services/announcements";

const { Paragraph, Title, Text } = Typography;

function renderStatus(status) {
  if (status === ANNOUNCEMENT_STATUS_PUBLISHED) {
    return <Tag color="success">已发布</Tag>;
  }

  return <Tag color="processing">草稿</Tag>;
}

export function AnnouncementsPage() {
  const navigate = useNavigate();
  const isMobile = useIsMobile();
  const [searchParams, setSearchParams] = useSearchParams();
  const [messageApi, contextHolder] = message.useMessage();
  const [projectsLoading, setProjectsLoading] = useState(true);
  const [loading, setLoading] = useState(false);
  const [projects, setProjects] = useState([]);
  const [announcements, setAnnouncements] = useState([]);
  const [total, setTotal] = useState(0);
  const [keywordInput, setKeywordInput] = useState("");
  const [query, setQuery] = useState({
    page: 1,
    pageSize: 5,
    keyword: "",
    status: "",
  });

  const selectedProjectId = searchParams.get("projectId") || "";
  const selectedProject = useMemo(
    () =>
      projects.find((project) => String(project.id) === selectedProjectId) ||
      null,
    [projects, selectedProjectId],
  );

  useEffect(() => {
    let cancelled = false;

    async function loadProjects() {
      setProjectsLoading(true);
      try {
        const data = await fetchProjects({
          page: 1,
          page_size: 100,
        });

        if (cancelled) {
          return;
        }

        setProjects(data.list);

        const currentProjectExists = data.list.some(
          (project) => String(project.id) === selectedProjectId,
        );
        if (selectedProjectId && currentProjectExists) {
          return;
        }

        if (data.list.length > 0) {
          setSearchParams(
            { projectId: String(data.list[0].id) },
            { replace: true },
          );
          return;
        }

        setSearchParams({}, { replace: true });
      } catch (error) {
        if (!cancelled) {
          messageApi.error(
            error?.response?.data?.message || "项目列表加载失败",
          );
        }
      } finally {
        if (!cancelled) {
          setProjectsLoading(false);
        }
      }
    }

    void loadProjects();

    return () => {
      cancelled = true;
    };
  }, [messageApi, selectedProjectId, setSearchParams]);

  useEffect(() => {
    let cancelled = false;

    async function loadAnnouncements() {
      if (!selectedProjectId) {
        setAnnouncements([]);
        setTotal(0);
        return;
      }

      setLoading(true);
      try {
        const data = await fetchAnnouncements(selectedProjectId, {
          page: query.page,
          page_size: query.pageSize,
          keyword: query.keyword || undefined,
          status: query.status || undefined,
        });

        if (cancelled) {
          return;
        }

        setAnnouncements(data.list);
        setTotal(data.total);
      } catch (error) {
        if (!cancelled) {
          messageApi.error(
            error?.response?.data?.message || "公告列表加载失败",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadAnnouncements();

    return () => {
      cancelled = true;
    };
  }, [
    messageApi,
    query.keyword,
    query.page,
    query.pageSize,
    query.status,
    selectedProjectId,
  ]);

  async function reloadCurrentPage() {
    if (!selectedProjectId) {
      return;
    }

    const data = await fetchAnnouncements(selectedProjectId, {
      page: query.page,
      page_size: query.pageSize,
      keyword: query.keyword || undefined,
      status: query.status || undefined,
    });
    setAnnouncements(data.list);
    setTotal(data.total);
  }

  async function syncQuery(nextQuery) {
    const isSameQuery =
      nextQuery.page === query.page &&
      nextQuery.pageSize === query.pageSize &&
      nextQuery.keyword === query.keyword &&
      nextQuery.status === query.status;

    setQuery(nextQuery);
    if (isSameQuery) {
      await reloadCurrentPage();
    }
  }

  async function handleSearch(keyword) {
    await syncQuery({
      ...query,
      page: 1,
      keyword,
    });
  }

  async function handlePublish(record) {
    try {
      await publishAnnouncement(record.id);
      messageApi.success(`公告「${record.title}」已发布`);
      await reloadCurrentPage();
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "发布公告失败");
    }
  }

  async function handleRevoke(record) {
    try {
      await revokeAnnouncement(record.id);
      messageApi.success(`公告「${record.title}」已撤回为草稿`);
      await reloadCurrentPage();
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "撤回公告失败");
    }
  }

  async function handleDelete(record) {
    try {
      await deleteAnnouncement(record.id);
      messageApi.success("公告已删除");

      if (announcements.length === 1 && query.page > 1) {
        setQuery((current) => ({ ...current, page: current.page - 1 }));
        return;
      }

      await reloadCurrentPage();
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "删除公告失败");
    }
  }

  const columns = [
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      render: (value, record) => (
        <Space wrap>
          <Button
            type="link"
            style={{ padding: 0 }}
            onClick={() =>
              navigate(
                `/announcements/${record.id}/edit?projectId=${record.project_id}`,
              )
            }
          >
            {value}
          </Button>
          {record.is_pinned ? <Tag color="gold">置顶</Tag> : null}
        </Space>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      render: (value) => renderStatus(value),
    },
    {
      title: "发布时间",
      dataIndex: "published_at",
      key: "published_at",
      render: (value) =>
        value ? (
          dayjs(value).format("YYYY-MM-DD HH:mm:ss")
        ) : (
          <Text type="secondary">未发布</Text>
        ),
    },
    {
      title: "更新时间",
      dataIndex: "updated_at",
      key: "updated_at",
      render: (value) => dayjs(value).format("YYYY-MM-DD HH:mm:ss"),
    },
    {
      title: "操作",
      key: "actions",
      width: 360,
      render: (_, record) => {
        const isPublished = record.status === ANNOUNCEMENT_STATUS_PUBLISHED;

        return (
          <Space wrap>
            <Button
              size="small"
              data-testid={`announcement-edit-${record.id}`}
              onClick={() =>
                navigate(
                  `/announcements/${record.id}/edit?projectId=${record.project_id}`,
                )
              }
            >
              {isPublished ? "编辑已发布" : "编辑草稿"}
            </Button>

            {!isPublished ? (
              <Popconfirm
                title="确认发布公告"
                description={`发布后公告「${record.title}」将对外可见，但仍允许后续继续编辑。`}
                okText="确认发布"
                cancelText="取消"
                onConfirm={() => handlePublish(record)}
              >
                <Button
                  size="small"
                  type="primary"
                  data-testid={`announcement-publish-${record.id}`}
                >
                  发布
                </Button>
              </Popconfirm>
            ) : (
              <Popconfirm
                title="确认撤回公告"
                description={`撤回后公告「${record.title}」将回到草稿状态，对外接口不再返回。`}
                okText="确认撤回"
                cancelText="取消"
                onConfirm={() => handleRevoke(record)}
              >
                <Button
                  size="small"
                  data-testid={`announcement-revoke-${record.id}`}
                >
                  撤回
                </Button>
              </Popconfirm>
            )}

            <Popconfirm
              title="确认删除公告"
              description={`删除后公告「${record.title}」将不可恢复。若当前已发布，也会立即从对外接口中消失。`}
              okText="确认删除"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              onConfirm={() => handleDelete(record)}
            >
              <Button
                size="small"
                danger
                data-testid={`announcement-delete-${record.id}`}
              >
                删除
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  if (!projectsLoading && projects.length === 0) {
    return (
      <>
        {contextHolder}
        <Card className="placeholder-card" bordered={false}>
          <Result
            status="info"
            title="暂无项目可管理公告"
            subTitle="公告属于项目维度，请先创建项目后再进入公告管理。"
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
              <Title level={2} style={{ marginBottom: 8 }}>
                公告管理
              </Title>
              <Paragraph className="placeholder-description">
                公告按项目维度管理。草稿和已发布状态在列表中会明确区分，已发布公告仍允许继续编辑。
              </Paragraph>
            </div>
            <Button
              type="primary"
              data-testid="announcement-create-button"
              disabled={!selectedProjectId}
              onClick={() =>
                navigate(`/announcements/new?projectId=${selectedProjectId}`)
              }
            >
              新建公告
            </Button>
          </div>

          <Alert
            type="info"
            showIcon
            message="已发布公告仍可继续编辑；保存后仍保持 published。撤回和删除都需要确认。"
          />

          <div className="table-toolbar">
            <Space wrap>
              <Text strong>选择项目</Text>
              <Select
                value={selectedProjectId || undefined}
                loading={projectsLoading}
                style={{ width: 260 }}
                data-testid="announcement-project-select"
                options={projects.map((project) => ({
                  label: project.name,
                  value: String(project.id),
                }))}
                onChange={(projectId) => {
                  setSearchParams({ projectId }, { replace: true });
                  setKeywordInput("");
                  setQuery({ page: 1, pageSize: 5, keyword: "", status: "" });
                }}
              />
            </Space>

            <Space wrap>
              <Text strong>状态筛选</Text>
              <Select
                value={query.status}
                style={{ width: 180 }}
                data-testid="announcement-status-select"
                options={[
                  { label: "全部状态", value: "" },
                  { label: "草稿", value: ANNOUNCEMENT_STATUS_DRAFT },
                  { label: "已发布", value: ANNOUNCEMENT_STATUS_PUBLISHED },
                ]}
                onChange={(status) => {
                  void syncQuery({
                    ...query,
                    page: 1,
                    status,
                  });
                }}
              />
            </Space>
          </div>

          <div className="table-toolbar">
            <Space.Compact block style={{ maxWidth: 480 }}>
              <Input
                allowClear
                placeholder="按公告标题搜索"
                value={keywordInput}
                onChange={(event) => setKeywordInput(event.target.value)}
                onPressEnter={() => handleSearch(keywordInput.trim())}
              />
              <Button
                type="primary"
                data-testid="announcements-search-button"
                onClick={() => handleSearch(keywordInput.trim())}
              >
                搜索
              </Button>
              <Button
                data-testid="announcements-reset-button"
                onClick={() => {
                  setKeywordInput("");
                  void handleSearch("");
                }}
              >
                重置
              </Button>
            </Space.Compact>
          </div>

          {!isMobile ? (
            <Table
              rowKey="id"
              loading={loading || projectsLoading}
              dataSource={announcements}
              columns={columns}
              locale={{
                emptyText: (
                  <Empty
                    description={
                      selectedProject
                        ? `项目「${selectedProject.name}」暂无公告`
                        : "暂无公告数据"
                    }
                  />
                ),
              }}
              pagination={{
                current: query.page,
                pageSize: query.pageSize,
                total,
                showSizeChanger: true,
                pageSizeOptions: ["5", "10", "20", "50"],
                showTotal: (count) => `共 ${count} 条公告`,
                onChange: (page, pageSize) =>
                  setQuery((current) => ({
                    ...current,
                    page,
                    pageSize,
                  })),
              }}
            />
          ) : loading || projectsLoading ? (
            <div className="page-inline-loading">
              <Spin tip="加载公告列表..." />
            </div>
          ) : announcements.length > 0 ? (
            <>
              <div className="mobile-list">
                {announcements.map((announcement) => {
                  const isPublished =
                    announcement.status === ANNOUNCEMENT_STATUS_PUBLISHED;

                  return (
                    <Card
                      key={announcement.id}
                      size="small"
                      className="mobile-list-card"
                    >
                      <Space
                        direction="vertical"
                        size={12}
                        style={{ width: "100%" }}
                      >
                        <Space wrap>
                          <Button
                            type="link"
                            style={{ padding: 0, textAlign: "left" }}
                            onClick={() =>
                              navigate(
                                `/announcements/${announcement.id}/edit?projectId=${announcement.project_id}`,
                              )
                            }
                          >
                            {announcement.title}
                          </Button>
                          {announcement.is_pinned ? (
                            <Tag color="gold">置顶</Tag>
                          ) : null}
                        </Space>

                        <div className="mobile-list-meta">
                          <div className="mobile-list-row">
                            <Text className="mobile-list-label">状态</Text>
                            <div className="mobile-list-value">
                              {renderStatus(announcement.status)}
                            </div>
                          </div>
                          <div className="mobile-list-row">
                            <Text className="mobile-list-label">发布时间</Text>
                            <Text className="mobile-list-value">
                              {announcement.published_at
                                ? dayjs(announcement.published_at).format(
                                    "YYYY-MM-DD HH:mm:ss",
                                  )
                                : "未发布"}
                            </Text>
                          </div>
                          <div className="mobile-list-row">
                            <Text className="mobile-list-label">更新时间</Text>
                            <Text className="mobile-list-value">
                              {dayjs(announcement.updated_at).format(
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
                            data-testid={`announcement-edit-${announcement.id}`}
                            onClick={() =>
                              navigate(
                                `/announcements/${announcement.id}/edit?projectId=${announcement.project_id}`,
                              )
                            }
                          >
                            {isPublished ? "编辑已发布" : "编辑草稿"}
                          </Button>
                          {!isPublished ? (
                            <Popconfirm
                              title="确认发布公告"
                              description={`发布后公告「${announcement.title}」将对外可见，但仍允许后续继续编辑。`}
                              okText="确认发布"
                              cancelText="取消"
                              onConfirm={() => handlePublish(announcement)}
                            >
                              <Button
                                type="primary"
                                data-testid={`announcement-publish-${announcement.id}`}
                              >
                                发布
                              </Button>
                            </Popconfirm>
                          ) : (
                            <Popconfirm
                              title="确认撤回公告"
                              description={`撤回后公告「${announcement.title}」将回到草稿状态，对外接口不再返回。`}
                              okText="确认撤回"
                              cancelText="取消"
                              onConfirm={() => handleRevoke(announcement)}
                            >
                              <Button
                                data-testid={`announcement-revoke-${announcement.id}`}
                              >
                                撤回
                              </Button>
                            </Popconfirm>
                          )}
                          <Popconfirm
                            title="确认删除公告"
                            description={`删除后公告「${announcement.title}」将不可恢复。若当前已发布，也会立即从对外接口中消失。`}
                            okText="确认删除"
                            cancelText="取消"
                            okButtonProps={{ danger: true }}
                            onConfirm={() => handleDelete(announcement)}
                          >
                            <Button
                              danger
                              data-testid={`announcement-delete-${announcement.id}`}
                            >
                              删除
                            </Button>
                          </Popconfirm>
                        </Space>
                      </Space>
                    </Card>
                  );
                })}
              </div>
              <Pagination
                current={query.page}
                pageSize={query.pageSize}
                total={total}
                showSizeChanger
                pageSizeOptions={["5", "10", "20", "50"]}
                showTotal={(count) => `共 ${count} 条公告`}
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
            <Empty
              description={
                selectedProject
                  ? `项目「${selectedProject.name}」暂无公告`
                  : "暂无公告数据"
              }
            />
          )}
        </Space>
      </Card>
    </>
  );
}
