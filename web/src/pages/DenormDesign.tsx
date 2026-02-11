import { useState, useCallback } from "react";
import { ReactFlowProvider } from "@xyflow/react";
import { DesignerCanvas } from "../components/designer/DesignerCanvas";
import { EdgeConfigPanel } from "../components/designer/EdgeConfigPanel";
import { DocumentPreview } from "../components/designer/DocumentPreview";
import { DesignerToolbar } from "../components/designer/DesignerToolbar";
import { Alert } from "../components/Alert";
import {
  useSchema,
  useMapping,
  useSaveMapping,
  useSetStep,
} from "../api/hooks";
import { useDesignerState } from "../hooks/useDesignerState";
import { useDocumentPreview } from "../hooks/useDocumentPreview";
import type { Mapping, Embedded, Reference } from "../api/types";

export default function DenormDesign() {
  const { data: schema, isLoading: schemaLoading } = useSchema();
  const { data: serverMapping, isLoading: mappingLoading } = useMapping();
  const saveMapping = useSaveMapping();
  const setStep = useSetStep();

  const {
    mapping,
    updateMapping,
    undo,
    redo,
    canUndo,
    canRedo,
  } = useDesignerState(serverMapping);

  const [selectedEdge, setSelectedEdge] = useState<{
    source: string;
    target: string;
    relationship: string;
  } | null>(null);

  const [selectedCollection, setSelectedCollection] = useState<string>();
  const previewJson = useDocumentPreview(mapping, schema, selectedCollection);

  const handleEdgeClick = useCallback(
    (source: string, target: string) => {
      // Find what relationship this edge represents
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
      // Set collection for preview
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
          const embedded: Embedded[] = parentCol.embedded || [];
          embedded.push({
            source_table: target,
            field_name: target.toLowerCase(),
            relationship: "array",
            join_column: `${source}_id`,
            parent_column: "id",
          });
          parentCol.embedded = embedded;
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
            // Convert to reference
            const embedded = (col.embedded || []).filter(
              (e) => e.source_table !== target && e.source_table !== source,
            );
            const refs: Reference[] = col.references || [];
            const t = target === col.source_table ? source : target;
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
            // Convert to embed
            const refs = (col.references || []).filter(
              (r) => r.source_table !== target && r.source_table !== source,
            );
            const embedded: Embedded[] = col.embedded || [];
            const t = target === col.source_table ? source : target;
            const existing = embedded.find((e) => e.source_table === t);
            if (existing) {
              existing.relationship = rel;
            } else {
              embedded.push({
                source_table: t,
                field_name: t.toLowerCase(),
                relationship: rel,
                join_column: `${col.source_table}_id`,
                parent_column: "id",
              });
            }
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
      onSuccess: () => setStep.mutate("type_mapping"),
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

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-bold text-gray-900">
          Denormalization Design
        </h2>
        <p className="mt-2 text-gray-600">
          Design how source tables map to MongoDB documents. Click an edge to
          configure the relationship. Drag from one node to another to create a
          new relationship.
        </p>
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

      <div className="grid grid-cols-3 gap-4">
        <div className="col-span-2 h-[500px] rounded-lg border border-gray-200 overflow-hidden relative">
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

        <div className="space-y-4">
          <DocumentPreview
            json={previewJson}
            collectionName={selectedCollection}
          />

          {mapping.collections.length > 1 && (
            <div className="rounded-lg border border-gray-200 bg-white p-3">
              <h3 className="text-sm font-medium text-gray-700 mb-2">
                Collections
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
