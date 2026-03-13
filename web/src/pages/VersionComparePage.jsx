import {
  Alert,
  Button,
  Card,
  Descriptions,
  Empty,
  Result,
  Select,
  Space,
  Spin,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";

import { useIsMobile } from "../hooks/useIsMobile";
import { fetchProjects } from "../services/projects";
import {
  CHANGELOG_TYPE_OPTIONS,
  VERSION_STATUS_PUBLISHED,
  compareVersions,
  exportChangelogs,
  fetchAllVersions,
} from "../services/versions";
import { triggerBlobDownload } from "../utils/download";

const { Paragraph, Text, Title } = Typography;

function renderStatusTag(status) {
  return status === VERSION_STATUS_PUBLISHED ? (
    <Tag color="success">published</Tag>
  ) : (
    <Tag color="processing">{status}</Tag>
  );
}

export function VersionComparePage() {
  const isMobile = useIsMobile();
  const [messageApi, contextHolder] = message.useMessage();
  const [projectsLoading, setProjectsLoading] = useState(true);
  const [versionsLoading, setVersionsLoading] = useState(false);
  const [compareLoading, setCompareLoading] = useState(false);
  const [exportingWholeProject, setExportingWholeProject] = useState(false);
  const [exportingSingleVersion, setExportingSingleVersion] = useState(false);
  const [projects, setProjects] = useState([]);
  const [allVersions, setAllVersions] = useState([]);
  const [selectedProjectId, setSelectedProjectId] = useState("");
  const [fromVersionId, setFromVersionId] = useState("");
  const [toVersionId, setToVersionId] = useState("");
  const [exportVersionId, setExportVersionId] = useState("");
  const [exportFormat, setExportFormat] = useState("markdown");
  const [compareResult, setCompareResult] = useState(null);
  const [compareError, setCompareError] = useState("");

  const selectedProject = useMemo(
    () =>
      projects.find((project) => String(project.id) === selectedProjectId) ||
      null,
    [projects, selectedProjectId],
  );
  const publishedVersions = useMemo(
    () =>
      allVersions.filter(
        (version) => version.status === VERSION_STATUS_PUBLISHED,
      ),
    [allVersions],
  );
  const hasPublishedVersions = publishedVersions.length > 0;

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
        if (data.list.length > 0) {
          setSelectedProjectId(String(data.list[0].id));
        }
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
  }, [messageApi]);

  useEffect(() => {
    let cancelled = false;

    async function loadVersions() {
      if (!selectedProjectId) {
        setAllVersions([]);
        return;
      }

      setVersionsLoading(true);
      setCompareError("");
      setCompareResult(null);

      try {
        const data = await fetchAllVersions(selectedProjectId);
        if (cancelled) {
          return;
        }

        setAllVersions(data.list);

        const nextPublished = data.list.filter(
          (version) => version.status === VERSION_STATUS_PUBLISHED,
        );
        setFromVersionId(nextPublished[0] ? String(nextPublished[0].id) : "");
        setToVersionId(
          nextPublished[1]
            ? String(nextPublished[1].id)
            : nextPublished[0]
              ? String(nextPublished[0].id)
              : "",
        );
        setExportVersionId(data.list[0] ? String(data.list[0].id) : "");
      } catch (error) {
        if (!cancelled) {
          messageApi.error(
            error?.response?.data?.message || "版本列表加载失败",
          );
        }
      } finally {
        if (!cancelled) {
          setVersionsLoading(false);
        }
      }
    }

    void loadVersions();

    return () => {
      cancelled = true;
    };
  }, [messageApi, selectedProjectId]);

  async function handleCompare() {
    if (!selectedProjectId || !fromVersionId || !toVersionId) {
      messageApi.warning("请先选择项目和两个版本");
      return;
    }

    setCompareLoading(true);
    setCompareError("");
    try {
      const data = await compareVersions(
        selectedProjectId,
        fromVersionId,
        toVersionId,
      );
      setCompareResult(data);
    } catch (error) {
      const nextError = error?.response?.data?.message || "版本对比失败";
      setCompareError(nextError);
      setCompareResult(null);
      messageApi.error(nextError);
    } finally {
      setCompareLoading(false);
    }
  }

  async function handleExport(versionId) {
    if (!selectedProjectId) {
      messageApi.warning("请先选择项目");
      return;
    }

    const setLoading = versionId
      ? setExportingSingleVersion
      : setExportingWholeProject;
    setLoading(true);
    try {
      const file = await exportChangelogs(
        selectedProjectId,
        exportFormat,
        versionId,
      );
      triggerBlobDownload(file.blob, file.filename);
      messageApi.success(`已开始下载 ${file.filename}`);
    } catch (error) {
      messageApi.error(
        error?.response?.data?.message || error?.message || "导出失败",
      );
    } finally {
      setLoading(false);
    }
  }

  if (!projectsLoading && projects.length === 0) {
    return (
      <>
        {contextHolder}
        <Card className="placeholder-card" bordered={false}>
          <Result
            status="info"
            title="暂无项目可用于版本对比"
            subTitle="请先创建项目与版本，再进入版本对比与导出。"
          />
        </Card>
      </>
    );
  }

  return (
    <>
      {contextHolder}
      <Space direction="vertical" size={20} style={{ width: "100%" }}>
        <Card className="placeholder-card" bordered={false}>
          <Space direction="vertical" size={20} style={{ width: "100%" }}>
            <div className="page-toolbar">
              <div>
                <Title level={2} style={{ marginBottom: 8 }}>
                  版本对比
                </Title>
                <Paragraph className="placeholder-description">
                  先选择项目，再选择两个已发布版本发起对比。区间归一化与结果分组完全以服务端
                  compare 返回为准。
                </Paragraph>
              </div>
            </div>

            <div className="table-toolbar">
              <Space
                wrap
                direction={isMobile ? "vertical" : "horizontal"}
                className={isMobile ? "compare-control-group" : undefined}
              >
                <Text strong>项目</Text>
                <Select
                  value={selectedProjectId || undefined}
                  loading={projectsLoading}
                  style={{ width: isMobile ? "100%" : 280 }}
                  data-testid="compare-project-select"
                  showSearch
                  optionFilterProp="label"
                  options={projects.map((project) => ({
                    label: project.name,
                    value: String(project.id),
                  }))}
                  onChange={setSelectedProjectId}
                />
              </Space>

              <Space wrap>
                <Text strong>版本数量</Text>
                <Tag color="default">
                  {publishedVersions.length} 个已发布版本可对比
                </Tag>
              </Space>
            </div>

            {!versionsLoading && selectedProjectId && !hasPublishedVersions ? (
              <Alert
                type="info"
                showIcon
                message="当前项目暂无已发布版本可对比"
                description="请先发布至少一个版本，再返回这里进行版本对比。"
              />
            ) : null}

            <div className="table-toolbar compare-toolbar">
              <Space
                wrap
                direction={isMobile ? "vertical" : "horizontal"}
                className={isMobile ? "compare-control-group" : undefined}
              >
                <Text strong>From</Text>
                <Select
                  value={fromVersionId || undefined}
                  loading={versionsLoading}
                  disabled={!hasPublishedVersions}
                  placeholder={
                    hasPublishedVersions
                      ? "选择起始版本"
                      : "暂无已发布版本可选"
                  }
                  style={{ width: isMobile ? "100%" : 220 }}
                  data-testid="compare-from-select"
                  showSearch
                  optionFilterProp="label"
                  notFoundContent="暂无可对比版本"
                  options={publishedVersions.map((version) => ({
                    label: version.version,
                    value: String(version.id),
                  }))}
                  onChange={setFromVersionId}
                />
              </Space>

              <Space
                wrap
                direction={isMobile ? "vertical" : "horizontal"}
                className={isMobile ? "compare-control-group" : undefined}
              >
                <Text strong>To</Text>
                <Select
                  value={toVersionId || undefined}
                  loading={versionsLoading}
                  disabled={!hasPublishedVersions}
                  placeholder={
                    hasPublishedVersions ? "选择结束版本" : "暂无已发布版本可选"
                  }
                  style={{ width: isMobile ? "100%" : 220 }}
                  data-testid="compare-to-select"
                  showSearch
                  optionFilterProp="label"
                  notFoundContent="暂无可对比版本"
                  options={publishedVersions.map((version) => ({
                    label: version.version,
                    value: String(version.id),
                  }))}
                  onChange={setToVersionId}
                />
              </Space>

              <Button
                type="primary"
                block={isMobile}
                data-testid="compare-submit-button"
                loading={compareLoading}
                className={isMobile ? "compare-submit-button" : undefined}
                disabled={!selectedProjectId || !hasPublishedVersions}
                onClick={handleCompare}
              >
                发起对比
              </Button>
            </div>

            {compareError ? (
              <Alert type="error" showIcon message={compareError} />
            ) : null}

            {compareLoading ? (
              <div className="loading-shell" style={{ minHeight: 240 }}>
                <Spin tip="正在加载版本对比..." />
              </div>
            ) : null}

            {!compareLoading && !compareResult ? (
              <Empty description="选择项目与两个已发布版本后即可查看对比结果。支持同版本自比较和逆序选择。" />
            ) : null}

            {!compareLoading && compareResult ? (
              <Space direction="vertical" size={16} style={{ width: "100%" }}>
                <Card className="inner-card" bordered={false}>
                  <Descriptions
                    bordered
                    column={1}
                    items={[
                      {
                        key: "from",
                        label: "归一化 From",
                        children: (
                          <Space wrap data-testid="compare-normalized-from">
                            <Text strong>
                              {compareResult.from_version.version}
                            </Text>
                            {renderStatusTag(compareResult.from_version.status)}
                          </Space>
                        ),
                      },
                      {
                        key: "to",
                        label: "归一化 To",
                        children: (
                          <Space wrap data-testid="compare-normalized-to">
                            <Text strong>
                              {compareResult.to_version.version}
                            </Text>
                            {renderStatusTag(compareResult.to_version.status)}
                          </Space>
                        ),
                      },
                      {
                        key: "versions",
                        label: "区间 Versions",
                        children: (
                          <Space wrap data-testid="compare-versions-range">
                            {compareResult.versions.map((item) => (
                              <Tag key={item.id} color="blue">
                                {item.version}
                              </Tag>
                            ))}
                          </Space>
                        ),
                      },
                    ]}
                  />
                </Card>

                {CHANGELOG_TYPE_OPTIONS.map((group) => {
                  const items = compareResult.changelogs[group.value] || [];

                  return (
                    <Card
                      key={group.value}
                      className="inner-card"
                      bordered={false}
                      title={`${group.label} (${items.length})`}
                      data-testid={`compare-group-${group.value}`}
                    >
                      {items.length === 0 ? (
                        <Empty
                          image={Empty.PRESENTED_IMAGE_SIMPLE}
                          description="该分组暂无变更"
                        />
                      ) : (
                        <Space
                          direction="vertical"
                          size={12}
                          style={{ width: "100%" }}
                        >
                          {items.map((item) => (
                            <Card key={item.id} size="small">
                              <Space
                                direction="vertical"
                                size={6}
                                style={{ width: "100%" }}
                              >
                                <Space wrap>
                                  <Tag color="geekblue">{item.version}</Tag>
                                  <Text type="secondary">
                                    sort_order: {item.sort_order}
                                  </Text>
                                  <Text type="secondary">
                                    updated: {item.updated_at}
                                  </Text>
                                </Space>
                                <Typography.Paragraph
                                  style={{ marginBottom: 0 }}
                                >
                                  {item.content}
                                </Typography.Paragraph>
                              </Space>
                            </Card>
                          ))}
                        </Space>
                      )}
                    </Card>
                  );
                })}
              </Space>
            ) : null}
          </Space>
        </Card>

        <Card className="placeholder-card" bordered={false}>
          <Space direction="vertical" size={20} style={{ width: "100%" }}>
            <div className="page-toolbar">
              <div>
                <Title level={3} style={{ marginBottom: 8 }}>
                  导出入口
                </Title>
                <Paragraph className="placeholder-description">
                  支持导出整个项目，或导出指定版本。文件名直接使用后端
                  `Content-Disposition` 返回值，不在前端伪造。
                </Paragraph>
              </div>
            </div>

            <div className="table-toolbar">
              <Space wrap>
                <Text strong>当前项目</Text>
                <Tag color="default">
                  {selectedProject?.name || "未选择项目"}
                </Tag>
              </Space>
              <Space wrap>
                <Text strong>格式</Text>
                <Select
                  value={exportFormat}
                  style={{ width: 180 }}
                  data-testid="export-format-select"
                  options={[
                    { label: "Markdown", value: "markdown" },
                    { label: "JSON", value: "json" },
                  ]}
                  onChange={setExportFormat}
                />
              </Space>
            </div>

            <div className="table-toolbar">
              <Space wrap>
                <Text strong>指定版本</Text>
                <Select
                  value={exportVersionId || undefined}
                  loading={versionsLoading}
                  style={{ width: 260 }}
                  data-testid="export-version-select"
                  showSearch
                  optionFilterProp="label"
                  options={allVersions.map((version) => ({
                    label: `${version.version} (${version.status})`,
                    value: String(version.id),
                  }))}
                  onChange={setExportVersionId}
                />
              </Space>

              <Space wrap>
                <Button
                  block={isMobile}
                  data-testid="export-project-button"
                  loading={exportingWholeProject}
                  disabled={!selectedProjectId}
                  onClick={() => handleExport()}
                >
                  导出整个项目
                </Button>
                <Button
                  type="primary"
                  block={isMobile}
                  data-testid="export-version-button"
                  loading={exportingSingleVersion}
                  disabled={!selectedProjectId || !exportVersionId}
                  onClick={() => handleExport(exportVersionId)}
                >
                  导出指定版本
                </Button>
              </Space>
            </div>

            <Alert
              type="info"
              showIcon
              message="全项目导出会带出该项目全部未删除版本；指定版本导出则只导出选中版本。"
            />
          </Space>
        </Card>
      </Space>
    </>
  );
}
