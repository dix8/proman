import { useEffect, useRef, useState } from "react";

import { fetchMarkdownPreview } from "../services/markdownPreview";

export function useMarkdownPreview(content, options = {}) {
  const { defaultTab = "edit" } = options;

  const [activeTab, setActiveTab] = useState(defaultTab);
  const [previewHtml, setPreviewHtml] = useState("");
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState("");
  const lastPreviewContentRef = useRef("");

  const trimmedContent = (content || "").trim();
  const hasPreviewContent = trimmedContent !== "";

  useEffect(() => {
    if (activeTab !== "preview" || !hasPreviewContent) {
      if (!hasPreviewContent) {
        setPreviewHtml("");
        setPreviewError("");
        setPreviewLoading(false);
      }
      return;
    }

    if (lastPreviewContentRef.current === trimmedContent && previewHtml) {
      return;
    }

    let cancelled = false;

    async function loadPreview() {
      setPreviewLoading(true);
      setPreviewError("");
      setPreviewHtml("");
      try {
        const data = await fetchMarkdownPreview(trimmedContent);
        if (cancelled) {
          return;
        }
        setPreviewHtml(data.html);
        lastPreviewContentRef.current = trimmedContent;
      } catch (error) {
        if (cancelled) {
          return;
        }
        setPreviewError(
          error?.response?.data?.message || "Markdown 预览加载失败",
        );
      } finally {
        if (!cancelled) {
          setPreviewLoading(false);
        }
      }
    }

    void loadPreview();

    return () => {
      cancelled = true;
    };
  }, [activeTab, hasPreviewContent, previewHtml, trimmedContent]);

  async function retryPreview() {
    if (!hasPreviewContent) {
      return;
    }

    setPreviewLoading(true);
    setPreviewError("");
    setPreviewHtml("");
    try {
      const data = await fetchMarkdownPreview(trimmedContent);
      setPreviewHtml(data.html);
      lastPreviewContentRef.current = trimmedContent;
    } catch (error) {
      setPreviewError(
        error?.response?.data?.message || "Markdown 预览加载失败",
      );
    } finally {
      setPreviewLoading(false);
    }
  }

  function handleTabChange(nextTab) {
    setActiveTab(nextTab);
  }

  return {
    activeTab,
    handleTabChange,
    hasPreviewContent,
    previewError,
    previewHtml,
    previewLoading,
    retryPreview,
  };
}
