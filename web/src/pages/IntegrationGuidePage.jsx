import { Alert, Button, Card, Space, Typography, message } from "antd";
import { useMemo } from "react";

import { useIsMobile } from "../hooks/useIsMobile";
import { http } from "../services/http";
import {
  TOKEN_PLACEHOLDER,
  buildPublicApiIntegrationEndpoints,
  resolvePublicAPIBaseURL,
  writeToClipboard,
} from "../utils/publicApiIntegration";

const { Paragraph, Text, Title } = Typography;

export function IntegrationGuidePage() {
  const isMobile = useIsMobile();
  const [messageApi, contextHolder] = message.useMessage();
  const publicAPIBaseURL = useMemo(() => {
    const fallbackOrigin =
      typeof window !== "undefined" ? window.location.origin : "";
    return resolvePublicAPIBaseURL(
      http.defaults.baseURL,
      fallbackOrigin,
      import.meta.env.VITE_PUBLIC_BASE_URL,
    );
  }, []);
  const authHeaderExample = `Authorization: Bearer ${TOKEN_PLACEHOLDER}`;
  const integrationEndpoints = useMemo(
    () => buildPublicApiIntegrationEndpoints(publicAPIBaseURL),
    [publicAPIBaseURL],
  );

  async function handleCopy(text, label) {
    try {
      await writeToClipboard(text);
      messageApi.success(`已复制${label}`);
    } catch (error) {
      messageApi.error(`复制${label}失败，请手动复制`);
    }
  }

  return (
    <>
      {contextHolder}
      <Card className="placeholder-card" bordered={false}>
        <Space direction="vertical" size={20} style={{ width: "100%" }}>
          <div className="page-toolbar">
            <div>
              <Title level={2} style={{ marginBottom: 8 }}>
                接口接入
              </Title>
              <Paragraph className="placeholder-description">
                这里统一展示后台当前开放的 `/v1` 接入方式。长期展示的内容保持全局说明，
                不长期展示真实 Token；真实 `project_token`
                仍只会在项目创建或刷新 Token 后一次性出现。
              </Paragraph>
            </div>
          </div>

          <Alert
            type="info"
            showIcon
            message={`统一鉴权方式：${authHeaderExample}`}
            description="对外接口统一使用项目级 Token 鉴权，Header 形式为 `Authorization: Bearer <project_token>`。如需真实 token，请到项目详情页刷新 Token 后立即复制并保存。"
          />

          <div>
            <Text strong>接口基地址</Text>
            <div className="snippet-block">
              <Paragraph
                copyable={{ text: publicAPIBaseURL }}
                className="snippet-text"
              >
                {publicAPIBaseURL}
              </Paragraph>
            </div>
          </div>

          <div>
            <Text strong>鉴权 Header</Text>
            <div className="snippet-block">
              <Paragraph
                copyable={{ text: authHeaderExample }}
                className="snippet-text"
              >
                {authHeaderExample}
              </Paragraph>
            </div>
          </div>

          {integrationEndpoints.map((endpoint) => (
            <Card key={endpoint.key} size="small">
              <Space direction="vertical" size={16} style={{ width: "100%" }}>
                <div>
                  <Space wrap>
                    <Text strong>{endpoint.title}</Text>
                    <Text code>{endpoint.method}</Text>
                    <Text code>{endpoint.path}</Text>
                  </Space>
                  <Paragraph
                    type="secondary"
                    style={{ marginTop: 8, marginBottom: 0 }}
                  >
                    {endpoint.description}
                  </Paragraph>
                </div>

                <div>
                  <Text strong>完整 URL</Text>
                  <div className="snippet-block">
                    <Paragraph
                      copyable={{ text: endpoint.fullURL }}
                      className="snippet-text"
                    >
                      {endpoint.fullURL}
                    </Paragraph>
                  </div>
                </div>

                <Space
                  wrap
                  direction={isMobile ? "vertical" : "horizontal"}
                  className={isMobile ? "mobile-list-actions" : undefined}
                >
                  <Button
                    onClick={() =>
                      handleCopy(endpoint.fullURL, `${endpoint.title} URL`)
                    }
                  >
                    复制 URL
                  </Button>
                  <Button
                    onClick={() =>
                      handleCopy(
                        endpoint.curlExample,
                        `${endpoint.title} curl 示例`,
                      )
                    }
                  >
                    复制 curl
                  </Button>
                  <Button
                    onClick={() =>
                      handleCopy(
                        endpoint.fetchExample,
                        `${endpoint.title} fetch 示例`,
                      )
                    }
                  >
                    复制 fetch
                  </Button>
                </Space>
              </Space>
            </Card>
          ))}
        </Space>
      </Card>
    </>
  );
}
