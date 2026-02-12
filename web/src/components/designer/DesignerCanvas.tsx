import { useCallback, useEffect, useMemo, useRef } from "react";
import {
  ReactFlow,
  Controls,
  MiniMap,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  useReactFlow,
  type Node,
  type Edge,
  type OnConnect,
  type EdgeMouseHandler,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { TableNode } from "./TableNode";
import { layoutNodes } from "../../utils/layoutGraph";
import type { Schema, Mapping, Collection } from "../../api/types";

const nodeTypes = { table: TableNode };

interface DesignerCanvasProps {
  schema: Schema;
  mapping: Mapping;
  onEdgeClick: (sourceTable: string, targetTable: string) => void;
  onConnect: (sourceTable: string, targetTable: string) => void;
  selectedEdge?: { source: string; target: string } | null;
}

function buildNodesAndEdges(
  schema: Schema,
  mapping: Mapping,
): { nodes: Node[]; edges: Edge[] } {
  const rootTables = new Set(mapping.collections.map((c) => c.source_table));
  const embeddedTables = new Set<string>();

  function collectEmbedded(col: Collection) {
    for (const emb of col.embedded || []) {
      embeddedTables.add(emb.source_table);
    }
  }
  mapping.collections.forEach(collectEmbedded);

  const rawNodes: Node[] = schema.tables.map((table) => ({
    id: table.name,
    type: "table",
    position: { x: 0, y: 0 },
    data: {
      table,
      isRoot: rootTables.has(table.name),
      isEmbedded: embeddedTables.has(table.name),
    },
  }));

  const edges: Edge[] = [];

  // FK edges from schema
  for (const table of schema.tables) {
    for (const fk of table.foreign_keys || []) {
      const hasMapping = mapping.collections.some(
        (c) =>
          c.embedded?.some(
            (e) =>
              e.source_table === table.name ||
              e.source_table === fk.referenced_table,
          ) ||
          c.references?.some(
            (r) =>
              r.source_table === table.name ||
              r.source_table === fk.referenced_table,
          ),
      );

      // Check if there's a mapping-based edge for this FK
      const isEmbed = mapping.collections.some((c) =>
        c.embedded?.some(
          (e) =>
            (e.source_table === table.name &&
              e.join_column === fk.columns[0]) ||
            (e.source_table === fk.referenced_table &&
              e.parent_column === fk.columns[0]),
        ),
      );

      const isRef = mapping.collections.some((c) =>
        c.references?.some(
          (r) =>
            r.source_table === table.name ||
            r.source_table === fk.referenced_table,
        ),
      );

      edges.push({
        id: `fk-${table.name}-${fk.name}`,
        source: fk.referenced_table,
        target: table.name,
        style: {
          stroke: isEmbed ? "#22c55e" : isRef ? "#6366f1" : "#d1d5db",
          strokeDasharray: hasMapping ? undefined : "5 5",
          strokeWidth: hasMapping ? 2 : 1,
        },
        label: isEmbed ? "embed" : isRef ? "ref" : undefined,
        labelStyle: { fontSize: 10 },
      });
    }
  }

  const nodes = layoutNodes(rawNodes, edges);
  return { nodes, edges };
}

export function DesignerCanvas({
  schema,
  mapping,
  onEdgeClick,
  onConnect: onConnectProp,
  selectedEdge,
}: DesignerCanvasProps) {
  const { nodes: initialNodes, edges: initialEdges } = useMemo(
    () => buildNodesAndEdges(schema, mapping),
    [schema, mapping],
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const { fitView } = useReactFlow();
  const fitViewCalled = useRef(false);

  // Sync edges when mapping changes
  useEffect(() => {
    setEdges(initialEdges);
  }, [initialEdges, setEdges]);

  // Auto-fit on first render
  if (!fitViewCalled.current && nodes.length > 0) {
    fitViewCalled.current = true;
    setTimeout(() => fitView({ padding: 0.2 }), 100);
  }

  const handleEdgeClick: EdgeMouseHandler = useCallback(
    (_event, edge) => {
      onEdgeClick(edge.source, edge.target);
    },
    [onEdgeClick],
  );

  const handleConnect: OnConnect = useCallback(
    (params) => {
      if (params.source && params.target) {
        onConnectProp(params.source, params.target);
      }
    },
    [onConnectProp],
  );

  // Highlight selected edge
  const styledEdges = edges.map((e) => {
    if (
      selectedEdge &&
      e.source === selectedEdge.source &&
      e.target === selectedEdge.target
    ) {
      return {
        ...e,
        style: { ...e.style, stroke: "#2563eb", strokeWidth: 3 },
        animated: true,
      };
    }
    return e;
  });

  // Expose re-layout via ref
  const doLayout = useCallback(() => {
    const laid = layoutNodes(nodes, edges);
    setNodes(laid);
    setTimeout(() => fitView({ padding: 0.2 }), 50);
  }, [nodes, edges, setNodes, fitView]);

  // Attach doLayout to window for toolbar access
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).__designerLayout = doLayout;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).__designerFitView = () =>
    fitView({ padding: 0.2 });

  return (
    <ReactFlow
      nodes={nodes}
      edges={styledEdges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onEdgeClick={handleEdgeClick}
      onConnect={handleConnect}
      nodeTypes={nodeTypes}
      fitView
      className="bg-gray-50"
    >
      <Controls />
      <MiniMap
        nodeStrokeWidth={3}
        className="!bg-gray-100"
      />
      <Background variant={BackgroundVariant.Dots} gap={16} size={1} />
    </ReactFlow>
  );
}
