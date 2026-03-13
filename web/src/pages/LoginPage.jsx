import { Alert, Button, Card, Form, Input, Typography } from "antd";
import { useMemo, useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";

import { hasToken, setToken } from "../services/auth";
import { http } from "../services/http";

const { Paragraph, Title } = Typography;

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const nextPath = useMemo(
    () => location.state?.from?.pathname || "/projects",
    [location.state],
  );

  if (hasToken()) {
    return <Navigate to={nextPath} replace />;
  }

  async function handleFinish(values) {
    setSubmitting(true);
    setErrorMessage("");
    try {
      const response = await http.post("/api/auth/login", values);
      setToken(response.data.data.token);
      navigate(nextPath, { replace: true });
    } catch (error) {
      const message = error?.response?.data?.message || "登录失败";
      setErrorMessage(message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="login-shell">
      <Card className="login-card" bordered={false}>
        <div className="login-head">
          <Typography.Text className="login-kicker">
            Admin Access
          </Typography.Text>
          <Title level={2}>Proman 后台登录</Title>
          <Paragraph>
            使用当前后端管理员账号登录，进入项目、版本与公告的管理入口。
          </Paragraph>
        </div>

        {errorMessage ? (
          <Alert
            type="error"
            showIcon
            message={errorMessage}
            style={{ marginBottom: 16 }}
          />
        ) : null}

        <Form layout="vertical" onFinish={handleFinish} autoComplete="off">
          <Form.Item
            label="用户名"
            name="username"
            rules={[{ required: true, message: "请输入用户名" }]}
          >
            <Input placeholder="请输入用户名" />
          </Form.Item>
          <Form.Item
            label="密码"
            name="password"
            rules={[{ required: true, message: "请输入密码" }]}
          >
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={submitting}>
            登录
          </Button>
        </Form>
      </Card>
    </div>
  );
}
