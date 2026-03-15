import { Form, Input, InputNumber, Space } from "antd";

export function VersionForm({ form, disabled = false, onFinish }) {
  return (
    <Form form={form} layout="vertical" onFinish={onFinish}>
      <Space
        wrap
        size={16}
        style={{ width: "100%" }}
        className="version-form-grid"
      >
        <Form.Item
          label="Major"
          name="major"
          rules={[
            { required: true, message: "请输入 major" },
            { type: "number", min: 0, message: "major 不能小于 0" },
          ]}
        >
          <InputNumber
            min={0}
            precision={0}
            disabled={disabled}
            style={{ width: "100%" }}
          />
        </Form.Item>
        <Form.Item
          label="Minor"
          name="minor"
          rules={[
            { required: true, message: "请输入 minor" },
            { type: "number", min: 0, message: "minor 不能小于 0" },
          ]}
        >
          <InputNumber
            min={0}
            precision={0}
            disabled={disabled}
            style={{ width: "100%" }}
          />
        </Form.Item>
        <Form.Item
          label="Patch"
          name="patch"
          rules={[
            { required: true, message: "请输入 patch" },
            { type: "number", min: 0, message: "patch 不能小于 0" },
          ]}
        >
          <InputNumber
            min={0}
            precision={0}
            disabled={disabled}
            style={{ width: "100%" }}
          />
        </Form.Item>
      </Space>
      <Form.Item
        label="发布地址"
        name="url"
        rules={[{ type: "url", message: "请输入有效的 URL" }]}
      >
        <Input
          placeholder="https://..."
          disabled={disabled}
          allowClear
        />
      </Form.Item>
    </Form>
  );
}
