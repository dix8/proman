import { Form, Input, Modal } from "antd";
import { useEffect } from "react";

import { useIsMobile } from "../hooks/useIsMobile";

const INITIAL_VALUES = {
  name: "",
  description: "",
};

export function ProjectFormModal({
  open,
  mode,
  initialValues,
  loading,
  onCancel,
  onSubmit,
}) {
  const [form] = Form.useForm();
  const isMobile = useIsMobile();

  useEffect(() => {
    if (!open) {
      form.resetFields();
      return;
    }

    form.setFieldsValue(initialValues || INITIAL_VALUES);
  }, [form, initialValues, open]);

  const title = mode === "edit" ? "编辑项目" : "新建项目";

  return (
    <Modal
      className="responsive-modal"
      title={title}
      open={open}
      width={isMobile ? "calc(100vw - 24px)" : 520}
      okText={mode === "edit" ? "保存修改" : "创建项目"}
      cancelText="取消"
      confirmLoading={loading}
      destroyOnClose
      onCancel={onCancel}
      onOk={() => form.submit()}
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={INITIAL_VALUES}
        onFinish={onSubmit}
      >
        <Form.Item
          label="项目名称"
          name="name"
          rules={[
            { required: true, message: "请输入项目名称" },
            { max: 100, message: "项目名称不能超过 100 个字符" },
          ]}
        >
          <Input placeholder="例如：OpenAPI 文档中心" />
        </Form.Item>
        <Form.Item
          label="项目描述"
          name="description"
          rules={[{ max: 1000, message: "项目描述不能超过 1000 个字符" }]}
        >
          <Input.TextArea
            rows={4}
            placeholder="输入项目用途、维护范围等描述信息"
            showCount
            maxLength={1000}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
