import { useState } from "react";

interface CodePreviewProps {
  code: string;
  language?: string;
  title?: string;
  collapsible?: boolean;
}

export function CodePreview({
  code,
  title,
  collapsible = false,
}: CodePreviewProps) {
  const [collapsed, setCollapsed] = useState(collapsible);

  return (
    <div className="rounded-lg border border-gray-200 bg-white overflow-hidden">
      {title && (
        <div
          className={`border-b border-gray-200 bg-gray-50 px-4 py-2 flex items-center justify-between ${
            collapsible ? "cursor-pointer" : ""
          }`}
          onClick={() => collapsible && setCollapsed(!collapsed)}
        >
          <h3 className="text-sm font-medium text-gray-700">{title}</h3>
          {collapsible && (
            <span className="text-xs text-gray-500">
              {collapsed ? "Show" : "Hide"}
            </span>
          )}
        </div>
      )}
      {!collapsed && (
        <pre className="p-4 text-xs text-gray-800 overflow-auto max-h-96 font-mono bg-gray-900 text-gray-100">
          {code}
        </pre>
      )}
    </div>
  );
}
