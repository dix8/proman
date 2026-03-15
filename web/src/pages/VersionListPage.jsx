import { ArrowLeftOutlined, LinkOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Empty,
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
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { fetchProject } from "../services/projects";
import { useIsMobile } from "../hooks/useIsMobile";
import {
  VERSION_STATUS_DRAFT,
  VERSION_STATUS_PUBLISHED,
  deleteVersion,
  fetchVersions,
  publishVersion,
  unpublishVersion,
} from "../services/versions";

const { Paragraph, Title, Text } = Typography;

function renderVersionStatus(status) {
  if (status === VERSION_STATUS_PUBLISHED) {
    return <Tag color="success">已发布 / 只读</Tag>;
  }

  return <Tag color="processing">草稿</Tag>;
}

export function VersionListPage() {
  const navigate = useNavigate();
  const { projectId } = useParams();
  const isMobile = useIsMobile();
  const [messageApi, contextHolder] = message.useMessage();
  const [loading, setLoading] = useState(true);
  const [project, setProject] = useState(null);
  const [missing, setMissing] = useState(false);
  const [versions, setVersions] = useState([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState({ page: 1, pageSize: 10, status: "" });

  useEffect(() => {
    let cancelled = false;

    async function loadPage() {
      setLoading(true);
      setMissing(false);

      try {
        const [projectData, versionsData] = await Promise.all([
          fetchProject(projectId),
          fetchVersions(projectId, {
            page: query.page,
            page_size: query.pageSize,
            status: query.status || undefined,
          }),
        ]);

        if (cancelled) {
          return;
        }

        setProject(projectData);
        setVersions(versionsData.list);
        setTotal(versionsData.total);
      } catch (error) {
        if (cancelled) {
          return;
        }

        if (error?.response?.status === 404) {
          setMissing(true);
        } else {
          messageApi.error(
            error?.response?.data?.message || "版本列表加载失败",
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
  }, [messageApi, projectId, query.page, query.pageSize, query.status]);

  async function reloadCurrentPage() {
    const versionsData = await fetchVersions(projectId, {
      page: query.page,
      page_size: query.pageSize,
      status: query.status || undefined,
    });
    setVersions(versionsData.list);
    setTotal(versionsData.total);
  }

  async function handleDelete(record) {
    try {
      await deleteVersion(record.id);
      messageApi.success("草稿版本已删除");

      if (versions.length === 1 && query.page > 1) {
        setQuery((current) => ({ ...current, page: current.page - 1 }));
        return;
      }

      await reloadCurrentPage();
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "删除版本失败");
    }
  }

  async function handlePublish(record) {
    try {
      await publishVersion(record.id);
      messageApi.success(`版本 ${record.version} 已发布`);
      await reloadCurrentPage();
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "发布版本失败");
    }
  }

  async function handleUnpublish(record) {
    try {
      await unpublishVersion(record.id);
      messageApi.success(`版本 ${record.version} 已撤回发布`);
      await reloadCurrentPage();
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "撤回发布失败");
    }
  }

  const columns = [
    {
      title: "版本号",
      dataIndex: "version",
      key: "version",
      render: (value, record) => (
        <Space>
          <Button
            type="link"
            style={{ padding: 0 }}
            onClick={() =>
              navigate(`/projects/${projectId}/versions/${record.id}/edit`)
            }
          >
            {value}
          </Button>
          {record.url ? (
            <a href={record.url} target="_blank" rel="noopener noreferrer" title={record.url}>
              <LinkOutlined />
            </a>
          ) : null}
        </Space>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      render: (value) => renderVersionStatus(value),
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
      width: 320,
      render: (_, record) => {
        const isPublished = record.status === VERSION_STATUS_PUBLISHED;

        return (
          <Space wrap>
            <Button
              size="small"
              data-testid={`version-edit-${record.id}`}
              onClick={() =>
                navigate(`/projects/${projectId}/versions/${record.id}/edit`)
              }
            >
              {isPublished ? "查看详情" : "编辑版本"}
            </Button>
            <Button
              size="small"
              data-testid={`version-changelogs-${record.id}`}
              onClick={() =>
                navigate(
                  `/projects/${projectId}/versions/${record.id}/changelogs`,
                )
              }
            >
              日志管理
            </Button>
            {!isPublished ? (
              <Popconfirm
                title="确认发布版本"
                description={`发布后版本 ${record.version} 及其日志将进入只读状态，如需修改可撤回发布。`}
                okText="确认发布"
                cancelText="取消"
                onConfirm={() => handlePublish(record)}
              >
                <Button
                  size="small"
                  type="primary"
                  data-testid={`version-publish-${record.id}`}
                >
                  发布
                </Button>
              </Popconfirm>
            ) : (
              <Popconfirm
                title="确认撤回发布"
                description={`撤回后版本 ${record.version} 将恢复为草稿状态，可继续编辑。`}
                okText="确认撤回"
                cancelText="取消"
                onConfirm={() => handleUnpublish(record)}
              >
                <Button
                  size="small"
                  data-testid={`version-unpublish-${record.id}`}
                >
                  撤回发布
                </Button>
              </Popconfirm>
            )}
            {!isPublished ? (
              <Popconfirm
                title="确认删除草稿版本"
                description={`删除后版本 ${record.version} 及其关联日志会一起软删除，此操作不可撤销。`}
                okText="确认删除"
                cancelText="取消"
                okButtonProps={{ danger: true }}
                onConfirm={() => handleDelete(record)}
              >
                <Button
                  size="small"
                  danger
                  data-testid={`version-delete-${record.id}`}
                >
                  删除
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        );
      },
    },
  ];

  if (loading) {
    return (
      <>
        {contextHolder}
        <div className="loading-shell">
          <Spin size="large" tip="加载版本列表..." />
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
                onClick={() => navigate(`/projects/${projectId}`)}
              >
                返回项目详情
              </Button>
              <Title level={2} style={{ marginBottom: 8 }}>
                {project?.name || "项目版本"}
              </Title>
              <Paragraph className="placeholder-description">
                这里管理项目版本的创建、编辑、发布与删除。已发布版本会进入只读状态，如需修改可撤回发布。
              </Paragraph>
            </div>
            <Button
              type="primary"
              data-testid="version-create-button"
              onClick={() => navigate(`/projects/${projectId}/versions/new`)}
            >
              新建版本
            </Button>
          </div>

          <Alert
            type="info"
            showIcon
            message="版本发布后进入只读态，如需修改可撤回发布恢复为草稿。"
          />

          <div className="table-toolbar">
            <Space wrap>
              <Text strong>状态筛选</Text>
              <Select
                value={query.status}
                style={{ width: 200 }}
                options={[
                  { label: "全部状态", value: "" },
                  { label: "草稿", value: VERSION_STATUS_DRAFT },
                  { label: "已发布", value: VERSION_STATUS_PUBLISHED },
                ]}
                onChange={(status) =>
                  setQuery((current) => ({
                    ...current,
                    page: 1,
                    status,
                  }))
                }
              />
            </Space>
          </div>

          {!isMobile ? (
            <Table
              rowKey="id"
              dataSource={versions}
              columns={columns}
              locale={{
                emptyText: <Empty description="暂无版本数据" />,
              }}
              pagination={{
                current: query.page,
                pageSize: query.pageSize,
                total,
                showSizeChanger: true,
                pageSizeOptions: ["10", "20", "50"],
                showTotal: (count) => `共 ${count} 个版本`,
                onChange: (page, pageSize) =>
                  setQuery((current) => ({
                    ...current,
                    page,
                    pageSize,
                  })),
              }}
            />
          ) : versions.length > 0 ? (
            <>
              <div className="mobile-list">
                {versions.map((version) => {
                  const isPublished =
                    version.status === VERSION_STATUS_PUBLISHED;

                  return (
                    <Card
                      key={version.id}
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
                          onClick={() =>
                            navigate(
                              `/projects/${projectId}/versions/${version.id}/edit`,
                            )
                          }
                        >
                          {version.version}
                        </Button>
                        {version.url ? (
                          <a href={version.url} target="_blank" rel="noopener noreferrer" title={version.url}>
                            <LinkOutlined />
                          </a>
                        ) : null}
                        <div className="mobile-list-meta">
                          <div className="mobile-list-row">
                            <Text className="mobile-list-label">状态</Text>
                            <div className="mobile-list-value">
                              {renderVersionStatus(version.status)}
                            </div>
                          </div>
                          <div className="mobile-list-row">
                            <Text className="mobile-list-label">发布时间</Text>
                            <Text className="mobile-list-value">
                              {version.published_at
                                ? dayjs(version.published_at).format(
                                    "YYYY-MM-DD HH:mm:ss",
                                  )
                                : "未发布"}
                            </Text>
                          </div>
                          <div className="mobile-list-row">
                            <Text className="mobile-list-label">更新时间</Text>
                            <Text className="mobile-list-value">
                              {dayjs(version.updated_at).format(
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
                            data-testid={`version-edit-${version.id}`}
                            onClick={() =>
                              navigate(
                                `/projects/${projectId}/versions/${version.id}/edit`,
                              )
                            }
                          >
                            {isPublished ? "查看详情" : "编辑版本"}
                          </Button>
                          <Button
                            data-testid={`version-changelogs-${version.id}`}
                            onClick={() =>
                              navigate(
                                `/projects/${projectId}/versions/${version.id}/changelogs`,
                              )
                            }
                          >
                            日志管理
                          </Button>
                          {!isPublished ? (
                            <Popconfirm
                              title="确认发布版本"
                              description={`发布后版本 ${version.version} 及其日志将进入只读状态，如需修改可撤回发布。`}
                              okText="确认发布"
                              cancelText="取消"
                              onConfirm={() => handlePublish(version)}
                            >
                              <Button
                                type="primary"
                                data-testid={`version-publish-${version.id}`}
                              >
                                发布
                              </Button>
                            </Popconfirm>
                          ) : (
                            <Popconfirm
                              title="确认撤回发布"
                              description={`撤回后版本 ${version.version} 将恢复为草稿状态，可继续编辑。`}
                              okText="确认撤回"
                              cancelText="取消"
                              onConfirm={() => handleUnpublish(version)}
                            >
                              <Button
                                data-testid={`version-unpublish-${version.id}`}
                              >
                                撤回发布
                              </Button>
                            </Popconfirm>
                          )}
                          {!isPublished ? (
                            <Popconfirm
                              title="确认删除草稿版本"
                              description={`删除后版本 ${version.version} 及其关联日志会一起软删除，此操作不可撤销。`}
                              okText="确认删除"
                              cancelText="取消"
                              okButtonProps={{ danger: true }}
                              onConfirm={() => handleDelete(version)}
                            >
                              <Button
                                danger
                                data-testid={`version-delete-${version.id}`}
                              >
                                删除
                              </Button>
                            </Popconfirm>
                          ) : null}
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
                pageSizeOptions={["10", "20", "50"]}
                showTotal={(count) => `共 ${count} 个版本`}
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
            <Empty description="暂无版本数据" />
          )}
        </Space>
      </Card>
    </>
  );
}
