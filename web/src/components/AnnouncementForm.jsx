import {
  Alert,
  Button,
  Empty,
  Form,
  Input,
  Select,
  Spin,
  Switch,
  Tabs,
  Typography,
} from "antd";

import { useMarkdownPreview } from "../hooks/useMarkdownPreview";

export function AnnouncementForm({
  form,
  projects = [],
  createMode = false,
  disabled = false,
  onFinish,
}) {
  const contentValue = Form.useWatch("content", form);
  const {
    activeTab,
    handleTabChange,
    hasPreviewContent,
    previewError,
    previewHtml,
    previewLoading,
    retryPreview,
  } = useMarkdownPreview(contentValue);

  return (
    <Form form={form} layout="vertical" onFinish={onFinish}>
      {createMode ? (
        <Form.Item
          label="所属项目"
          name="projectId"
          rules={[{ required: true, message: "请选择所属项目" }]}
        >
          <Select
            disabled={disabled}
            className="responsive-full-control"
            options={projects.map((project) => ({
              label: project.name,
              value: String(project.id),
            }))}
            placeholder="请选择所属项目"
            showSearch
            optionFilterProp="label"
          />
        </Form.Item>
      ) : null}

      <Form.Item
        label="公告标题"
        name="title"
        rules={[
          { required: true, message: "请输入公告标题" },
          { max: 150, message: "公告标题不能超过 150 个字符" },
        ]}
      >
        <Input
          disabled={disabled}
          placeholder="例如：服务升级通知"
          maxLength={150}
          showCount
        />
      </Form.Item>

      <Form.Item label="Markdown 内容">
        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          items={[
            {
              key: "edit",
              label: "编辑",
              children: (
                <Form.Item
                  name="content"
                  rules={[
                    { required: true, message: "请输入公告内容" },
                    { max: 20000, message: "公告内容不能超过 20000 个字符" },
                  ]}
                >
                  <Input.TextArea
                    disabled={disabled}
                    placeholder="输入公告 Markdown 原文"
                    rows={12}
                    maxLength={20000}
                    showCount
                  />
                </Form.Item>
              ),
            },
            {
              key: "preview",
              label: "预览",
              children: (
                <div
                  className="markdown-preview-shell"
                  data-testid="announcement-preview-shell"
                >
                  {!hasPreviewContent ? (
                    <Empty description="输入公告内容后即可预览，空内容不会发起预览请求。" />
                  ) : null}

                  {hasPreviewContent && previewLoading ? (
                    <div className="markdown-preview-state">
                      <Spin tip="正在加载服务端预览..." />
                    </div>
                  ) : null}

                  {hasPreviewContent && !previewLoading && previewError ? (
                    <Alert
                      type="error"
                      showIcon
                      message={previewError}
                      action={
                        <Button size="small" onClick={retryPreview}>
                          重试
                        </Button>
                      }
                    />
                  ) : null}

                  {hasPreviewContent && !previewLoading && !previewError ? (
                    <>
                      <Typography.Text
                        type="secondary"
                        className="markdown-preview-hint"
                      >
                        当前预览 HTML 来自服务端 `/api/markdown/preview`。
                      </Typography.Text>
                      <div
                        className="markdown-preview-pane"
                        data-testid="markdown-preview-pane"
                        dangerouslySetInnerHTML={{ __html: previewHtml }}
                      />
                    </>
                  ) : null}
                </div>
              ),
            },
          ]}
        />
      </Form.Item>

      <Form.Item label="置顶公告" name="is_pinned" valuePropName="checked">
        <Switch
          disabled={disabled}
          checkedChildren="置顶"
          unCheckedChildren="普通"
        />
      </Form.Item>
    </Form>
  );
}
