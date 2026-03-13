import {
  Alert,
  Button,
  Empty,
  Form,
  Input,
  Modal,
  Select,
  Spin,
  Tabs,
  Typography,
} from "antd";
import { useEffect } from "react";

import { useIsMobile } from "../hooks/useIsMobile";
import { useMarkdownPreview } from "../hooks/useMarkdownPreview";
import { CHANGELOG_TYPE_OPTIONS } from "../services/versions";

const INITIAL_VALUES = {
  type: CHANGELOG_TYPE_OPTIONS[0].value,
  content: "",
};

export function ChangelogFormModal({
  open,
  mode,
  initialValues,
  loading,
  readOnly = false,
  onCancel,
  onSubmit,
}) {
  const [form] = Form.useForm();
  const isMobile = useIsMobile();
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

  useEffect(() => {
    if (!open) {
      form.resetFields();
      return;
    }

    form.setFieldsValue(initialValues || INITIAL_VALUES);
    handleTabChange("edit");
  }, [form, initialValues, open]);

  const isEdit = mode === "edit";

  return (
    <Modal
      className="responsive-modal"
      title={isEdit ? "编辑日志" : "新增日志"}
      open={open}
      width={isMobile ? "calc(100vw - 24px)" : 640}
      destroyOnClose
      confirmLoading={loading}
      okText={readOnly ? "关闭" : isEdit ? "保存修改" : "创建日志"}
      cancelText={readOnly ? null : "取消"}
      cancelButtonProps={readOnly ? { style: { display: "none" } } : undefined}
      onCancel={onCancel}
      onOk={() => {
        if (readOnly) {
          onCancel();
          return;
        }
        form.submit();
      }}
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={INITIAL_VALUES}
        onFinish={onSubmit}
      >
        <Form.Item
          label="日志类型"
          name="type"
          rules={[{ required: true, message: "请选择日志类型" }]}
        >
          <Select
            disabled={readOnly}
            options={CHANGELOG_TYPE_OPTIONS}
            placeholder="请选择日志类型"
          />
        </Form.Item>

        <Tabs
          activeKey={activeTab}
          onChange={handleTabChange}
          items={[
            {
              key: "edit",
              label: "编辑",
              children: (
                <Form.Item
                  label="Markdown 内容"
                  name="content"
                  rules={[
                    { required: true, message: "请输入日志内容" },
                    { max: 20000, message: "日志内容不能超过 20000 个字符" },
                  ]}
                >
                  <Input.TextArea
                    rows={8}
                    disabled={readOnly}
                    placeholder="输入更新日志 Markdown 原文"
                    showCount
                    maxLength={20000}
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
                  data-testid="changelog-preview-shell"
                >
                  {!hasPreviewContent ? (
                    <Empty description="输入日志内容后即可预览，空内容不会发起预览请求。" />
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
      </Form>
    </Modal>
  );
}
