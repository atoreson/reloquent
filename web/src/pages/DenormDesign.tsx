import { useState, useCallback, useEffect } from "react";
import { ReactFlowProvider } from "@xyflow/react";
import { DesignerCanvas } from "../components/designer/DesignerCanvas";
import { EdgeConfigPanel } from "../components/designer/EdgeConfigPanel";
import { DocumentPreview } from "../components/designer/DocumentPreview";
import { DesignerToolbar } from "../components/designer/DesignerToolbar";
import { Alert } from "../components/Alert";
import { Button } from "../components/Button";
import {
  useSchema,
  useMapping,
  useMappingPreview,
  useSaveMapping,
  useNavigateToStep,
} from "../api/hooks";
import { useDesignerState } from "../hooks/useDesignerState";
import { useDocumentPreview } from "../hooks/useDocumentPreview";
import type { Mapping, Embedded, Reference } from "../api/types";

function RootCollectionPicker({
  tables,
  selected,
  onToggle,
  onConfirm,
}: {
  tables: { name: string; row_count: number; fk_count: number }[];
  selected: Set<string>;
  onToggle: (table: string) => void;
  onConfirm: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center h-full p-8">
      <div className="max-w-2xl w-full">
        <h2 className="text-2xl font-bold text-gray-900">
          Denormalization Design
        </h2>
        <p className="mt-2 text-gray-600">
          Choose which tables should become <strong>root MongoDB collections</strong>.
          All other tables will be embedded as subdocuments within these collections.
        </p>

        <div className="mt-6 rounded-lg border border-gray-200 bg-white overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-gray-50 border-b border-gray-200">
                <th className="px-4 py-2 text-left w-10"></th>
                <th className="px-4 py-2 text-left font-medium text-gray-700">Table</th>
                <th className="px-4 py-2 text-right font-medium text-gray-700">Rows</th>
                <th className="px-4 py-2 text-right font-medium text-gray-700">Foreign Keys</th>
              </tr>
            </thead>
            <tbody>
              {tables.map((t) => (
                <tr
                  key={t.name}
                  className={`border-b border-gray-100 cursor-pointer transition-colors ${
                    selected.has(t.name)
                      ? "bg-blue-50"
                      : "hover:bg-gray-50"
                  }`}
                  onClick={() => onToggle(t.name)}
                >
                  <td className="px-4 py-2.5">
                    <input
                      type="checkbox"
                      checked={selected.has(t.name)}
                      onChange={() => onToggle(t.name)}
                      className="rounded border-gray-300 text-blue-600"
                    />
                  </td>
                  <td className="px-4 py-2.5 font-mono text-gray-900">{t.name}</td>
                  <td className="px-4 py-2.5 text-right text-gray-600">
                    {t.row_count.toLocaleString()}
                  </td>
                  <td className="px-4 py-2.5 text-right text-gray-600">{t.fk_count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        <p className="mt-3 text-xs text-gray-500">
          Tip: Tables with no foreign keys (like <code>actor</code>, <code>category</code>) are
          natural roots. Tables with foreign keys (like <code>film_actor</code>) are usually
          embedded into the table they reference.
        </p>

        <div className="mt-4 flex gap-3">
          <Button onClick={onConfirm} disabled={selected.size === 0}>
            Generate Mapping ({selected.size} collection{selected.size !== 1 ? "s" : ""})
          </Button>
        </div>
      </div>
    </div>
  );
}

export default function DenormDesign() {
  const { data: schema, isLoading: schemaLoading } = useSchema();
  const { data: serverMapping, isLoading: mappingLoading } = useMapping();
  const saveMapping = useSaveMapping();
  const goToStep = useNavigateToStep();

  // Root collection selection state
  const [rootTables, setRootTables] = useState<Set<string>>(new Set());
  const [rootsConfirmed, setRootsConfirmed] = useState(false);

  // Only fetch preview after roots are confirmed
  const { data: previewMapping } = useMappingPreview(
    rootsConfirmed ? Array.from(rootTables) : undefined,
  );

  // Use saved mapping if available, otherwise use preview after roots confirmed
  const initialMapping = serverMapping ?? (rootsConfirmed ? previewMapping : undefined);

  // If server already has a mapping, skip the picker
  useEffect(() => {
    if (serverMapping && serverMapping.collections.length > 0) {
      setRootsConfirmed(true);
      setRootTables(
        new Set(serverMapping.collections.map((c) => c.source_table)),
      );
    }
  }, [serverMapping]);

  const {
    mapping,
    updateMapping,
    undo,
    redo,
    canUndo,
    canRedo,
  } = useDesignerState(initialMapping);

  const [selectedEdge, setSelectedEdge] = useState<{
    source: string;
    target: string;
    relationship: string;
  } | null>(null);

  const [selectedCollection, setSelectedCollection] = useState<string>();
  const previewJson = useDocumentPreview(mapping, schema, selectedCollection);

  const handleEdgeClick = useCallback(
    (source: string, target: string) => {
      let rel = "reference";
      for (const col of mapping.collections) {
        const emb = col.embedded?.find(
          (e) => e.source_table === target || e.source_table === source,
        );
        if (emb) {
          rel = emb.relationship;
          break;
        }
        const ref = col.references?.find(
          (r) => r.source_table === target || r.source_table === source,
        );
        if (ref) {
          rel = "reference";
          break;
        }
      }
      setSelectedEdge({ source, target, relationship: rel });
      const col = mapping.collections.find(
        (c) => c.source_table === source || c.source_table === target,
      );
      if (col) setSelectedCollection(col.name);
    },
    [mapping],
  );

  const handleConnect = useCallback(
    (source: string, target: string) => {
      updateMapping((m: Mapping) => {
        const cols = [...m.collections];
        const parentCol = cols.find((c) => c.source_table === source);
        if (parentCol) {
          const embedded: Embedded[] = [...(parentCol.embedded || [])];
          embedded.push({
            source_table: target,
            field_name: target.toLowerCase(),
            relationship: "array",
            join_column: `${source}_id`,
            parent_column: "id",
          });
          return {
            ...m,
            collections: cols.map((c) =>
              c === parentCol ? { ...c, embedded } : c,
            ),
          };
        }
        return { ...m, collections: cols };
      });
    },
    [updateMapping],
  );

  const handleChangeRelationship = useCallback(
    (rel: string) => {
      if (!selectedEdge) return;
      const { source, target } = selectedEdge;

      updateMapping((m: Mapping) => {
        const cols = m.collections.map((col) => {
          if (col.source_table !== source && col.source_table !== target)
            return col;

          if (rel === "reference") {
            const embedded = (col.embedded || []).filter(
              (e) => e.source_table !== target && e.source_table !== source,
            );
            const t = target === col.source_table ? source : target;
            const refs: Reference[] = [...(col.references || [])];
            if (!refs.some((r) => r.source_table === t)) {
              refs.push({
                source_table: t,
                field_name: `${t.toLowerCase()}_id`,
                join_column: `${col.source_table}_id`,
                parent_column: "id",
              });
            }
            return { ...col, embedded, references: refs };
          } else {
            const refs = (col.references || []).filter(
              (r) => r.source_table !== target && r.source_table !== source,
            );
            const t = target === col.source_table ? source : target;
            const existing = (col.embedded || []).find(
              (e) => e.source_table === t,
            );
            const embedded: Embedded[] = existing
              ? (col.embedded || []).map((e) =>
                  e.source_table === t ? { ...e, relationship: rel } : e,
                )
              : [
                  ...(col.embedded || []),
                  {
                    source_table: t,
                    field_name: t.toLowerCase(),
                    relationship: rel,
                    join_column: `${col.source_table}_id`,
                    parent_column: "id",
                  },
                ];
            return { ...col, embedded, references: refs };
          }
        });
        return { ...m, collections: cols };
      });

      setSelectedEdge((prev) => (prev ? { ...prev, relationship: rel } : null));
    },
    [selectedEdge, updateMapping],
  );

  const handleRemoveRelationship = useCallback(() => {
    if (!selectedEdge) return;
    const { source, target } = selectedEdge;

    updateMapping((m: Mapping) => {
      const cols = m.collections.map((col) => ({
        ...col,
        embedded: (col.embedded || []).filter(
          (e) => e.source_table !== source && e.source_table !== target,
        ),
        references: (col.references || []).filter(
          (r) => r.source_table !== source && r.source_table !== target,
        ),
      }));
      return { ...m, collections: cols };
    });
    setSelectedEdge(null);
  }, [selectedEdge, updateMapping]);

  const handleSave = () => {
    saveMapping.mutate(mapping, {
      onSuccess: () => goToStep("type_mapping"),
    });
  };

  if (schemaLoading || mappingLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  if (!schema) {
    return <Alert type="warning">No schema discovered yet. Complete Step 1 first.</Alert>;
  }

  // Show root collection picker if roots not yet confirmed
  if (!rootsConfirmed) {
    const tableInfo = schema.tables.map((t) => ({
      name: t.name,
      row_count: t.row_count || 0,
      fk_count: (t.foreign_keys || []).length,
    }));

    return (
      <RootCollectionPicker
        tables={tableInfo}
        selected={rootTables}
        onToggle={(name) =>
          setRootTables((prev) => {
            const next = new Set(prev);
            if (next.has(name)) next.delete(name);
            else next.add(name);
            return next;
          })
        }
        onConfirm={() => setRootsConfirmed(true)}
      />
    );
  }

  return (
    <div className="flex flex-col h-full p-4 gap-3">
      <div className="shrink-0 flex items-start justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">
            Denormalization Design
          </h2>
          <p className="mt-1 text-gray-600 text-sm">
            Design how source tables map to MongoDB documents. Click an edge to
            configure the relationship. Drag from one node to another to create a
            new relationship.
          </p>
        </div>
        <button
          onClick={() => setRootsConfirmed(false)}
          className="text-sm text-blue-600 hover:text-blue-800 whitespace-nowrap ml-4"
        >
          Change Root Tables
        </button>
      </div>

      <DesignerToolbar
        onZoomFit={() =>
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          (window as any).__designerFitView?.()
        }
        onAutoLayout={() =>
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          (window as any).__designerLayout?.()
        }
        onUndo={undo}
        onRedo={redo}
        canUndo={canUndo}
        canRedo={canRedo}
        onSave={handleSave}
        saving={saveMapping.isPending}
      />

      <div className="grid grid-cols-[1fr_300px] gap-4 min-h-0 flex-1">
        <div className="rounded-lg border border-gray-200 overflow-hidden relative">
          <ReactFlowProvider>
            <DesignerCanvas
              schema={schema}
              mapping={mapping}
              onEdgeClick={handleEdgeClick}
              onConnect={handleConnect}
              selectedEdge={selectedEdge}
            />
          </ReactFlowProvider>

          {selectedEdge && (
            <EdgeConfigPanel
              sourceTable={selectedEdge.source}
              targetTable={selectedEdge.target}
              relationship={selectedEdge.relationship}
              onChangeRelationship={handleChangeRelationship}
              onRemove={handleRemoveRelationship}
              onClose={() => setSelectedEdge(null)}
            />
          )}
        </div>

        <div className="space-y-4 overflow-y-auto">
          <DocumentPreview
            json={previewJson}
            collectionName={selectedCollection}
          />

          {mapping.collections.length > 0 && (
            <div className="rounded-lg border border-gray-200 bg-white p-3">
              <h3 className="text-sm font-medium text-gray-700 mb-2">
                Collections ({mapping.collections.length})
              </h3>
              <ul className="space-y-1">
                {mapping.collections.map((col) => (
                  <li key={col.name}>
                    <button
                      onClick={() => setSelectedCollection(col.name)}
                      className={`w-full text-left text-sm px-2 py-1 rounded ${
                        selectedCollection === col.name
                          ? "bg-blue-50 text-blue-700"
                          : "text-gray-600 hover:bg-gray-50"
                      }`}
                    >
                      {col.name}
                      {(col.embedded?.length ?? 0) > 0 && (
                        <span className="text-xs text-gray-400 ml-1">
                          ({col.embedded!.length} embedded)
                        </span>
                      )}
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </div>

      {saveMapping.error && (
        <Alert type="error">{saveMapping.error.message}</Alert>
      )}
    </div>
  );
}
