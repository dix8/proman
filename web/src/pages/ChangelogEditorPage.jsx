import {
  ArrowDownOutlined,
  ArrowLeftOutlined,
  ArrowUpOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Empty,
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
import {
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";

import { ChangelogFormModal } from "../components/ChangelogFormModal";
import { useIsMobile } from "../hooks/useIsMobile";
import { fetchProject } from "../services/projects";
import {
  CHANGELOG_TYPE_OPTIONS,
  VERSION_STATUS_PUBLISHED,
  createChangelog,
  deleteChangelog,
  fetchAllChangelogs,
  fetchVersion,
  reorderChangelogs,
  updateChangelog,
} from "../services/versions";

const { Text, Title } = Typography;

function renderChangelogType(type) {
  const option = CHANGELOG_TYPE_OPTIONS.find((item) => item.value === type);
  return <Tag color="blue">{option?.label || type}</Tag>;
}

function moveItem(list, index, direction) {
  const targetIndex = index + direction;
  if (targetIndex < 0 || targetIndex >= list.length) {
    return list;
  }

  const nextList = [...list];
  const [currentItem] = nextList.splice(index, 1);
  nextList.splice(targetIndex, 0, currentItem);
  return nextList;
}

export function ChangelogEditorPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { projectId: routeProjectId, versionId } = useParams();
  const isMobile = useIsMobile();
  const [searchParams] = useSearchParams();
  const [messageApi, contextHolder] = message.useMessage();
  const [loading, setLoading] = useState(true);
  const [savingOrder, setSavingOrder] = useState(false);
  const [project, setProject] = useState(null);
  const [version, setVersion] = useState(null);
  const [changelogs, setChangelogs] = useState([]);
  const [missing, setMissing] = useState(false);
  const [filterType, setFilterType] = useState("");
  const [sortDirty, setSortDirty] = useState(false);
  const [modalState, setModalState] = useState({
    open: false,
    mode: "create",
    readOnly: false,
    item: null,
  });
  const [modalSubmitting, setModalSubmitting] = useState(false);

  const projectId = useMemo(
    () =>
      routeProjectId ||
      searchParams.get("projectId") ||
      location.state?.projectId ||
      "",
    [location.state?.projectId, routeProjectId, searchParams],
  );

  const isReadOnly = version?.status === VERSION_STATUS_PUBLISHED;
  const reorderDisabled =
    isReadOnly || filterType !== "" || changelogs.length <= 1;

  useEffect(() => {
    let cancelled = false;

    async function loadPage() {
      if (!projectId || !versionId) {
        setMissing(true);
        setLoading(false);
        return;
      }

      setLoading(true);
      setMissing(false);

      try {
        const [projectData, versionData] = await Promise.all([
          fetchProject(projectId),
          fetchVersion(versionId),
        ]);

        const changelogData = await fetchAllChangelogs(versionId, filterType);

        if (cancelled) {
          return;
        }

        setProject(projectData);
        setVersion(versionData);
        setChangelogs(changelogData.list);
        setSortDirty(false);
      } catch (error) {
        if (cancelled) {
          return;
        }

        if (error?.response?.status === 404) {
          setMissing(true);
        } else {
          messageApi.error(
            error?.response?.data?.message || "日志列表加载失败",
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
  }, [filterType, messageApi, projectId, versionId]);

  async function reloadChangelogs(nextType = filterType) {
    const changelogData = await fetchAllChangelogs(versionId, nextType);
    setChangelogs(changelogData.list);
    setSortDirty(false);
  }

  function openCreateModal() {
    setModalState({
      open: true,
      mode: "create",
      readOnly: false,
      item: null,
    });
  }

  function openEditModal(item, readOnly = false) {
    setModalState({
      open: true,
      mode: "edit",
      readOnly,
      item,
    });
  }

  async function handleModalSubmit(values) {
    setModalSubmitting(true);
    try {
      if (modalState.mode === "create") {
        await createChangelog(versionId, values);
        messageApi.success("日志已创建");
      } else {
        await updateChangelog(modalState.item.id, values);
        messageApi.success("日志已更新");
      }

      setModalState({
        open: false,
        mode: "create",
        readOnly: false,
        item: null,
      });
      await reloadChangelogs();
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "日志保存失败");
    } finally {
      setModalSubmitting(false);
    }
  }

  async function handleDelete(item) {
    try {
      await deleteChangelog(item.id);
      messageApi.success("日志已删除");
      await reloadChangelogs();
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "日志删除失败");
    }
  }

  async function handleSaveOrder() {
    setSavingOrder(true);
    try {
      await reorderChangelogs(
        versionId,
        changelogs.map((item, index) => ({
          id: item.id,
          sort_order: index + 1,
        })),
      );
      messageApi.success("日志顺序已保存");
      await reloadChangelogs("");
    } catch (error) {
      messageApi.error(error?.response?.data?.message || "日志排序保存失败");
    } finally {
      setSavingOrder(false);
    }
  }

  function handleMove(index, direction) {
    setChangelogs((current) => moveItem(current, index, direction));
    setSortDirty(true);
  }

  const columns = [
    {
      title: "顺序",
      key: "order",
      width: 140,
      render: (_, record, index) => (
        <Space size="small">
          <Text>{index + 1}</Text>
          {!reorderDisabled ? (
            <>
              <Button
                size="small"
                icon={<ArrowUpOutlined />}
                data-testid={`changelog-move-up-${record.id}`}
                disabled={index === 0}
                onClick={() => handleMove(index, -1)}
              />
              <Button
                size="small"
                icon={<ArrowDownOutlined />}
                data-testid={`changelog-move-down-${record.id}`}
                disabled={index === changelogs.length - 1}
                onClick={() => handleMove(index, 1)}
              />
            </>
          ) : null}
        </Space>
      ),
    },
    {
      title: "类型",
      dataIndex: "type",
      key: "type",
      width: 120,
      render: (value) => renderChangelogType(value),
    },
    {
      title: "内容",
      dataIndex: "content",
      key: "content",
      render: (value) => (
        <Typography.Paragraph
          ellipsis={{ rows: 3, expandable: true, symbol: "展开" }}
          style={{ marginBottom: 0 }}
        >
          {value}
        </Typography.Paragraph>
      ),
    },
    {
      title: "更新时间",
      dataIndex: "updated_at",
      key: "updated_at",
      width: 190,
      render: (value) => dayjs(value).format("YYYY-MM-DD HH:mm:ss"),
    },
    {
      title: "操作",
      key: "actions",
      width: 220,
      render: (_, record) => (
        <Space wrap>
          {isReadOnly ? (
            <Button
              size="small"
              data-testid={`changelog-view-${record.id}`}
              onClick={() => openEditModal(record, true)}
            >
              查看
            </Button>
          ) : (
            <>
              <Button
                size="small"
                data-testid={`changelog-edit-${record.id}`}
                onClick={() => openEditModal(record)}
              >
                编辑
              </Button>
              <Popconfirm
                title="确认删除日志"
                description="删除后该日志将不可恢复。"
                okText="确认删除"
                cancelText="取消"
                okButtonProps={{ danger: true }}
                onConfirm={() => handleDelete(record)}
              >
                <Button
                  size="small"
                  danger
                  data-testid={`changelog-delete-${record.id}`}
                >
                  删除
                </Button>
              </Popconfirm>
            </>
          )}
        </Space>
      ),
    },
  ];

  if (loading) {
    return (
      <>
        {contextHolder}
        <div className="loading-shell">
          <Spin size="large" tip="加载日志列表..." />
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
            title="版本不存在"
            subTitle="当前版本可能已被删除，或你没有权限访问它。"
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
                onClick={() =>
                  navigate(`/projects/${projectId}/versions/${versionId}/edit`)
                }
              >
                返回版本详情
              </Button>
              <Title level={2} style={{ marginBottom: 8 }}>
                {project?.name || "项目"} / {version?.version || "版本"} 日志
              </Title>
              <Typography.Paragraph className="placeholder-description">
                这里管理版本下的更新日志。草稿版本支持新增、编辑、删除和排序；已发布版本下的日志统一只读。
              </Typography.Paragraph>
            </div>
            <Space wrap>
              {!isReadOnly ? (
                <Button
                  type="primary"
                  data-testid="changelog-create-button"
                  onClick={openCreateModal}
                >
                  新增日志
                </Button>
              ) : null}
              <Button
                onClick={() => navigate(`/projects/${projectId}/versions`)}
              >
                返回版本列表
              </Button>
            </Space>
          </div>

          {isReadOnly ? (
            <Alert
              type="warning"
              showIcon
              message="当前版本已发布，日志进入只读状态。不能新增、编辑、删除或调整顺序。"
            />
          ) : null}

          {!isReadOnly && filterType ? (
            <Alert
              type="info"
              showIcon
              message="当前正在按日志类型筛选。为避免提交不完整排序，排序功能在筛选状态下已禁用。"
            />
          ) : null}

          <div className="table-toolbar">
            <Space wrap>
              <Text strong>类型筛选</Text>
              <Select
                value={filterType}
                style={{ width: 220 }}
                options={[
                  { label: "全部类型", value: "" },
                  ...CHANGELOG_TYPE_OPTIONS,
                ]}
                onChange={setFilterType}
              />
            </Space>
            {!reorderDisabled ? (
              <Button
                type="primary"
                data-testid="changelog-save-order-button"
                loading={savingOrder}
                disabled={!sortDirty}
                onClick={handleSaveOrder}
              >
                保存排序
              </Button>
            ) : null}
          </div>

          {!isMobile ? (
            <Table
              rowKey="id"
              dataSource={changelogs}
              columns={columns}
              pagination={false}
              locale={{
                emptyText: <Empty description="暂无日志数据" />,
              }}
            />
          ) : changelogs.length > 0 ? (
            <div className="mobile-list">
              {changelogs.map((item, index) => (
                <Card key={item.id} size="small" className="mobile-list-card">
                  <Space
                    direction="vertical"
                    size={12}
                    style={{ width: "100%" }}
                  >
                    <Space wrap>
                      <Tag color="geekblue">#{index + 1}</Tag>
                      {renderChangelogType(item.type)}
                    </Space>
                    <div className="mobile-list-meta">
                      <div className="mobile-list-row">
                        <Text className="mobile-list-label">更新时间</Text>
                        <Text className="mobile-list-value">
                          {dayjs(item.updated_at).format("YYYY-MM-DD HH:mm:ss")}
                        </Text>
                      </div>
                    </div>
                    <Typography.Paragraph
                      ellipsis={{ rows: 4, expandable: true, symbol: "展开" }}
                      style={{ marginBottom: 0 }}
                    >
                      {item.content}
                    </Typography.Paragraph>
                    <Space
                      direction="vertical"
                      size={8}
                      className="mobile-list-actions"
                    >
                      {!reorderDisabled ? (
                        <Space wrap>
                          <Button
                            size="small"
                            icon={<ArrowUpOutlined />}
                            data-testid={`changelog-move-up-${item.id}`}
                            disabled={index === 0}
                            onClick={() => handleMove(index, -1)}
                          >
                            上移
                          </Button>
                          <Button
                            size="small"
                            icon={<ArrowDownOutlined />}
                            data-testid={`changelog-move-down-${item.id}`}
                            disabled={index === changelogs.length - 1}
                            onClick={() => handleMove(index, 1)}
                          >
                            下移
                          </Button>
                        </Space>
                      ) : null}

                      {isReadOnly ? (
                        <Button
                          data-testid={`changelog-view-${item.id}`}
                          onClick={() => openEditModal(item, true)}
                        >
                          查看
                        </Button>
                      ) : (
                        <>
                          <Button
                            data-testid={`changelog-edit-${item.id}`}
                            onClick={() => openEditModal(item)}
                          >
                            编辑
                          </Button>
                          <Popconfirm
                            title="确认删除日志"
                            description="删除后该日志将不可恢复。"
                            okText="确认删除"
                            cancelText="取消"
                            okButtonProps={{ danger: true }}
                            onConfirm={() => handleDelete(item)}
                          >
                            <Button
                              danger
                              data-testid={`changelog-delete-${item.id}`}
                            >
                              删除
                            </Button>
                          </Popconfirm>
                        </>
                      )}
                    </Space>
                  </Space>
                </Card>
              ))}
            </div>
          ) : (
            <Empty description="暂无日志数据" />
          )}
        </Space>
      </Card>

      <ChangelogFormModal
        open={modalState.open}
        mode={modalState.mode}
        readOnly={modalState.readOnly}
        initialValues={
          modalState.item
            ? {
                type: modalState.item.type,
                content: modalState.item.content,
              }
            : undefined
        }
        loading={modalSubmitting}
        onCancel={() =>
          setModalState({
            open: false,
            mode: "create",
            readOnly: false,
            item: null,
          })
        }
        onSubmit={handleModalSubmit}
      />
    </>
  );
}
