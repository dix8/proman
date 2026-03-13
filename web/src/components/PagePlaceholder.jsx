import { Button, Card, Space, Typography } from "antd";

const { Paragraph, Title, Text } = Typography;

export function PagePlaceholder({
  title,
  description,
  routePath,
  primaryActionLabel,
  onPrimaryAction,
}) {
  return (
    <Card className="placeholder-card" bordered={false}>
      <Space direction="vertical" size={16}>
        <Text className="placeholder-tag">Route Ready</Text>
        <Title level={2}>{title}</Title>
        <Paragraph className="placeholder-description">{description}</Paragraph>
        <Text type="secondary">当前路由：{routePath}</Text>
        {primaryActionLabel ? (
          <Button type="primary" onClick={onPrimaryAction}>
            {primaryActionLabel}
          </Button>
        ) : null}
      </Space>
    </Card>
  );
}
