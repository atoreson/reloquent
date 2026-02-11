import { useState, useEffect } from "react";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import { TypeSelect } from "../components/TypeSelect";
import { useTypeMap, useSaveTypeMap, useSetStep } from "../api/hooks";

export default function TypeMapping() {
  const { data: entries, isLoading, error } = useTypeMap();
  const saveTypeMap = useSaveTypeMap();
  const setStep = useSetStep();
  const [overrides, setOverrides] = useState<Record<string, string>>({});
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (entries && !initialized) {
      const o: Record<string, string> = {};
      for (const e of entries) {
        o[e.source_type] = e.bson_type;
      }
      setOverrides(o);
      setInitialized(true);
    }
  }, [entries, initialized]);

  const handleChange = (sourceType: string, bsonType: string) => {
    setOverrides((prev) => ({ ...prev, [sourceType]: bsonType }));
  };

  const handleRestore = (sourceType: string) => {
    if (!entries) return;
    const original = entries.find((e) => e.source_type === sourceType);
    if (original && !original.overridden) {
      // already at default — nothing to restore
      return;
    }
    // Remove from overrides to let server restore default
    setOverrides((prev) => {
      const next = { ...prev };
      delete next[sourceType];
      return next;
    });
  };

  const handleSave = () => {
    saveTypeMap.mutate(overrides, {
      onSuccess: () => setStep.mutate("sizing"),
    });
  };

  const isModified = (sourceType: string) => {
    if (!entries) return false;
    const original = entries.find((e) => e.source_type === sourceType);
    return original && overrides[sourceType] !== original.bson_type;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  if (error) {
    return <Alert type="error">{error.message}</Alert>;
  }

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Type Mapping</h2>
      <p className="mt-2 text-gray-600">
        Map source SQL types to MongoDB BSON types. Override any mapping by
        selecting a different BSON type.
      </p>

      <div className="mt-6 rounded-lg border border-gray-200 bg-white overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50">
              <th className="px-4 py-3 text-left font-medium text-gray-700">
                Source Type
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-700">
                BSON Type
              </th>
              <th className="px-4 py-3 text-left font-medium text-gray-700 w-24">
                Status
              </th>
              <th className="px-4 py-3 w-20" />
            </tr>
          </thead>
          <tbody>
            {entries?.map((entry) => (
              <tr
                key={entry.source_type}
                className={`border-b border-gray-100 ${isModified(entry.source_type) ? "bg-yellow-50" : ""}`}
              >
                <td className="px-4 py-2.5 font-mono text-gray-900">
                  {entry.source_type}
                </td>
                <td className="px-4 py-2.5">
                  <TypeSelect
                    value={overrides[entry.source_type] || entry.bson_type}
                    onChange={(v) => handleChange(entry.source_type, v)}
                  />
                </td>
                <td className="px-4 py-2.5">
                  {(entry.overridden || isModified(entry.source_type)) && (
                    <span className="text-xs text-yellow-700 bg-yellow-100 px-2 py-0.5 rounded-full">
                      modified
                    </span>
                  )}
                </td>
                <td className="px-4 py-2.5">
                  {(entry.overridden || isModified(entry.source_type)) && (
                    <button
                      onClick={() => handleRestore(entry.source_type)}
                      className="text-xs text-blue-600 hover:text-blue-800"
                    >
                      Restore
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {saveTypeMap.error && (
        <Alert type="error">{saveTypeMap.error.message}</Alert>
      )}

      <div className="mt-6 flex gap-3">
        <Button onClick={handleSave} loading={saveTypeMap.isPending}>
          Save & Continue
        </Button>
      </div>
    </div>
  );
}
