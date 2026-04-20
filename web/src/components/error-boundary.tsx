import { Button, Result, Typography } from "antd";
import { Component, type ErrorInfo, type ReactNode } from "react";

type Props = { children: ReactNode };

type State = { error: Error | null };

/**
 * 捕获子树渲染错误，避免整页白屏；便于运维平台在单页异常时仍可回到首页。
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // eslint-disable-next-line no-console
    console.error("ErrorBoundary", error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <Result
          status="error"
          title="页面渲染异常"
          subTitle={
            <Typography.Paragraph type="secondary" style={{ maxWidth: 560, margin: "0 auto" }}>
              {this.state.error.message || "未知错误"}
            </Typography.Paragraph>
          }
          extra={
            <Button
              type="primary"
              onClick={() => {
                this.setState({ error: null });
                window.location.assign("/");
              }}
            >
              返回首页
            </Button>
          }
        />
      );
    }
    return this.props.children;
  }
}
