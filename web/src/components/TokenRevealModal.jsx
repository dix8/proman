import { Alert, Button, Modal, Space, Typography } from "antd";

import { useIsMobile } from "../hooks/useIsMobile";

const { Paragraph, Text } = Typography;

export function TokenRevealModal({
  open,
  title,
  description,
  token,
  warning,
  snippetGroups = [],
  onClose,
}) {
  const isMobile = useIsMobile();

  return (
    <Modal
      className="responsive-modal"
      title={title}
      open={open}
      width={isMobile ? "calc(100vw - 24px)" : 520}
      footer={
        <Button type="primary" onClick={onClose}>
          我已知晓
        </Button>
      }
      onCancel={onClose}
      destroyOnClose
    >
      <Space direction="vertical" size={16} style={{ width: "100%" }}>
        <Paragraph>{description}</Paragraph>
        {warning ? <Alert type="warning" showIcon message={warning} /> : null}
        <div className="token-block">
          <Text code copyable={{ text: token }} className="token-text">
            {token}
          </Text>
        </div>
        {snippetGroups.length > 0 ? (
          <>
            <Alert
              type="info"
              showIcon
              message="以下示例已带入当前新 Token，仅本次可见，关闭后不会再次展示。"
            />
            {snippetGroups.map((group) => (
              <Space
                key={group.title}
                direction="vertical"
                size={8}
                style={{ width: "100%" }}
              >
                <Text strong>{group.title}</Text>
                {group.description ? (
                  <Text type="secondary">{group.description}</Text>
                ) : null}
                {group.items.map((item) => (
                  <div key={item.label}>
                    <Text strong>{item.label}</Text>
                    <div className="snippet-block">
                      <Paragraph
                        copyable={{ text: item.value }}
                        className="snippet-text"
                      >
                        {item.value}
                      </Paragraph>
                    </div>
                  </div>
                ))}
              </Space>
            ))}
          </>
        ) : null}
      </Space>
    </Modal>
  );
}
