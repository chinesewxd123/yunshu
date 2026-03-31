import { Breadcrumb, Space, Typography } from "antd";
import type { BreadcrumbProps } from "antd";
import type { ReactNode } from "react";

interface PageHeroProps {
  title: string;
  subtitle?: string;
  breadcrumbItems?: BreadcrumbProps["items"];
  extra?: ReactNode;
}

export function PageHero({ title, subtitle, breadcrumbItems, extra }: PageHeroProps) {
  return (
    <div className="page-hero">
      <div>
        {breadcrumbItems && breadcrumbItems.length > 0 ? (
          <Breadcrumb className="page-hero__breadcrumb" items={breadcrumbItems} />
        ) : null}
        <Typography.Title level={2} className="page-hero__title">
          {title}
        </Typography.Title>
        {subtitle ? (
          <Typography.Paragraph className="page-hero__subtitle">{subtitle}</Typography.Paragraph>
        ) : null}
      </div>
      {extra ? <Space wrap>{extra}</Space> : null}
    </div>
  );
}
