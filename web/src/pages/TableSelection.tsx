import { useState, useMemo } from "react";
import { Button } from "../components/Button";
import { Input } from "../components/FormField";
import { Alert } from "../components/Alert";
import { PageContainer } from "../components/PageContainer";
import { useTables, useSelectTables, useNavigateToStep } from "../api/hooks";

function formatBytes(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024)
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
  if (bytes >= 1024 * 1024)
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${bytes} B`;
}

function formatNumber(n: number): string {
  return n.toLocaleString();
}

type SortField = "name" | "row_count" | "size_bytes";
type SortDir = "asc" | "desc";

export default function TableSelection() {
  const { data: tables, isLoading, error } = useTables();
  const selectTables = useSelectTables();
  const goToStep = useNavigateToStep();
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [sortField, setSortField] = useState<SortField>("name");
  const [sortDir, setSortDir] = useState<SortDir>("asc");
  const [initialized, setInitialized] = useState(false);

  // Initialize selection from server state
  if (tables && !initialized) {
    const sel = new Set<string>();
    for (const t of tables) {
      if (t.selected) sel.add(t.name);
    }
    if (sel.size > 0) setSelected(sel);
    setInitialized(true);
  }

  const filtered = useMemo(() => {
    if (!tables) return [];
    let result = tables;
    if (search) {
      const lower = search.toLowerCase();
      result = result.filter((t) => t.name.toLowerCase().includes(lower));
    }
    result = [...result].sort((a, b) => {
      const aVal = a[sortField];
      const bVal = b[sortField];
      if (typeof aVal === "string" && typeof bVal === "string") {
        return sortDir === "asc"
          ? aVal.localeCompare(bVal)
          : bVal.localeCompare(aVal);
      }
      return sortDir === "asc"
        ? (aVal as number) - (bVal as number)
        : (bVal as number) - (aVal as number);
    });
    return result;
  }, [tables, search, sortField, sortDir]);

  const summary = useMemo(() => {
    if (!tables) return { rows: 0, size: 0 };
    let rows = 0;
    let size = 0;
    for (const t of tables) {
      if (selected.has(t.name)) {
        rows += t.row_count;
        size += t.size_bytes;
      }
    }
    return { rows, size };
  }, [tables, selected]);

  const toggleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortField(field);
      setSortDir("asc");
    }
  };

  const toggleAll = () => {
    if (filtered.length === 0) return;
    const allSelected = filtered.every((t) => selected.has(t.name));
    const next = new Set(selected);
    for (const t of filtered) {
      if (allSelected) {
        next.delete(t.name);
      } else {
        next.add(t.name);
      }
    }
    setSelected(next);
  };

  const toggle = (name: string) => {
    const next = new Set(selected);
    if (next.has(name)) {
      next.delete(name);
    } else {
      next.add(name);
    }
    setSelected(next);
  };

  const handleContinue = () => {
    selectTables.mutate([...selected], {
      onSuccess: () => {
        goToStep("denormalization");
      },
    });
  };

  const sortIcon = (field: SortField) => {
    if (sortField !== field) return "↕";
    return sortDir === "asc" ? "↑" : "↓";
  };

  if (isLoading) {
    return (
      <PageContainer>
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full" />
      </div>
      </PageContainer>
    );
  }

  if (error) {
    return <PageContainer><Alert type="error">{error.message}</Alert></PageContainer>;
  }

  return (
    <PageContainer>
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Table Selection</h2>
      <p className="mt-2 text-gray-600">
        Select which tables to include in the migration.
      </p>

      <div className="mt-6 flex items-center justify-between gap-4">
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search tables..."
          className="max-w-xs"
        />
        <div className="text-sm text-gray-600">
          <span className="font-medium">{selected.size}</span> of{" "}
          <span className="font-medium">{tables?.length || 0}</span> tables
          selected &middot;{" "}
          <span className="font-medium">{formatNumber(summary.rows)}</span>{" "}
          rows &middot;{" "}
          <span className="font-medium">{formatBytes(summary.size)}</span>
        </div>
      </div>

      <div className="mt-4 rounded-lg border border-gray-200 bg-white overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50">
              <th className="px-4 py-3 text-left w-8">
                <input
                  type="checkbox"
                  checked={
                    filtered.length > 0 &&
                    filtered.every((t) => selected.has(t.name))
                  }
                  onChange={toggleAll}
                  className="rounded border-gray-300"
                />
              </th>
              <th
                className="px-4 py-3 text-left font-medium text-gray-700 cursor-pointer select-none"
                onClick={() => toggleSort("name")}
              >
                Table {sortIcon("name")}
              </th>
              <th
                className="px-4 py-3 text-right font-medium text-gray-700 cursor-pointer select-none"
                onClick={() => toggleSort("row_count")}
              >
                Rows {sortIcon("row_count")}
              </th>
              <th
                className="px-4 py-3 text-right font-medium text-gray-700 cursor-pointer select-none"
                onClick={() => toggleSort("size_bytes")}
              >
                Size {sortIcon("size_bytes")}
              </th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((t) => (
              <tr
                key={t.name}
                className="border-b border-gray-100 hover:bg-gray-50 cursor-pointer"
                onClick={() => toggle(t.name)}
              >
                <td className="px-4 py-2.5">
                  <input
                    type="checkbox"
                    checked={selected.has(t.name)}
                    onChange={() => toggle(t.name)}
                    className="rounded border-gray-300"
                  />
                </td>
                <td className="px-4 py-2.5 font-mono text-gray-900">
                  {t.name}
                </td>
                <td className="px-4 py-2.5 text-right text-gray-600">
                  {formatNumber(t.row_count)}
                </td>
                <td className="px-4 py-2.5 text-right text-gray-600">
                  {formatBytes(t.size_bytes)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {filtered.length === 0 && (
          <div className="p-8 text-center text-gray-400">
            {search ? "No tables match your search" : "No tables discovered"}
          </div>
        )}
      </div>

      {selectTables.error && (
        <Alert type="error">{selectTables.error.message}</Alert>
      )}

      <div className="mt-6 flex gap-3">
        <Button
          onClick={handleContinue}
          loading={selectTables.isPending}
          disabled={selected.size === 0}
        >
          Select {selected.size} Tables & Continue
        </Button>
      </div>
    </div>
    </PageContainer>
  );
}
