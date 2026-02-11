import { memo } from "react";
import { Handle, Position } from "@xyflow/react";
import type { NodeProps, Node } from "@xyflow/react";
import type { Table } from "../../api/types";

type TableNodeData = {
  table: Table;
  isRoot: boolean;
  isEmbedded: boolean;
};

type TableNodeType = Node<TableNodeData, "table">;

function formatNumber(n: number): string {
  return n.toLocaleString();
}

export const TableNode = memo(function TableNode({
  data,
}: NodeProps<TableNodeType>) {
  const { table, isRoot, isEmbedded } = data;

  const borderColor = isRoot
    ? "border-blue-400"
    : isEmbedded
      ? "border-green-400"
      : "border-gray-300";

  return (
    <div
      className={`rounded-lg border-2 bg-white shadow-sm min-w-[220px] ${borderColor}`}
    >
      <Handle type="target" position={Position.Top} className="!bg-gray-400" />

      <div
        className={`px-3 py-2 border-b text-sm font-semibold ${
          isRoot
            ? "bg-blue-50 text-blue-800"
            : isEmbedded
              ? "bg-green-50 text-green-800"
              : "bg-gray-50 text-gray-800"
        }`}
      >
        {table.name}
        <span className="ml-2 text-xs font-normal opacity-70">
          {formatNumber(table.row_count)} rows
        </span>
      </div>

      <div className="px-3 py-2 space-y-0.5 max-h-32 overflow-y-auto">
        {table.columns.slice(0, 8).map((col) => {
          const isPK = table.primary_key?.columns.includes(col.name);
          const isFK = table.foreign_keys?.some((fk) =>
            fk.columns.includes(col.name),
          );
          return (
            <div
              key={col.name}
              className="flex items-center gap-1.5 text-xs text-gray-600"
            >
              {isPK && (
                <span className="text-yellow-600 font-bold" title="Primary Key">
                  PK
                </span>
              )}
              {isFK && (
                <span className="text-blue-600 font-bold" title="Foreign Key">
                  FK
                </span>
              )}
              <span className={isPK ? "font-medium text-gray-900" : ""}>
                {col.name}
              </span>
              <span className="text-gray-400 ml-auto">{col.data_type}</span>
            </div>
          );
        })}
        {table.columns.length > 8 && (
          <div className="text-xs text-gray-400">
            +{table.columns.length - 8} more columns
          </div>
        )}
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-gray-400"
      />
    </div>
  );
});
