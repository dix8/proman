import { Button, Popconfirm } from "antd";

export function TokenRefreshConfirmButton({
  children = "刷新 Token",
  onConfirm,
  ...buttonProps
}) {
  return (
    <Popconfirm
      title="确认刷新 Token"
      description="刷新后旧 Token 会立即失效，调用方需要尽快更新配置。系统只会返回一次新 Token 明文。"
      okText="确认刷新"
      cancelText="取消"
      onConfirm={onConfirm}
    >
      <Button {...buttonProps}>{children}</Button>
    </Popconfirm>
  );
}
