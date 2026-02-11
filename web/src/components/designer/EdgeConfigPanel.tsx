import { Button } from "../Button";

interface EdgeConfigPanelProps {
  sourceTable: string;
  targetTable: string;
  relationship: string;
  onChangeRelationship: (rel: string) => void;
  onRemove: () => void;
  onClose: () => void;
}

export function EdgeConfigPanel({
  sourceTable,
  targetTable,
  relationship,
  onChangeRelationship,
  onRemove,
  onClose,
}: EdgeConfigPanelProps) {
  return (
    <div className="absolute right-4 top-4 z-50 w-72 rounded-lg border border-gray-200 bg-white p-4 shadow-lg">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-gray-900">
          Relationship
        </h3>
        <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
          ✕
        </button>
      </div>

      <p className="text-xs text-gray-500 mb-3">
        {sourceTable} → {targetTable}
      </p>

      <div className="space-y-2 mb-4">
        <label className="flex items-center gap-2 text-sm cursor-pointer">
          <input
            type="radio"
            name="rel"
            value="array"
            checked={relationship === "array"}
            onChange={() => onChangeRelationship("array")}
            className="text-blue-600"
          />
          <span>
            <span className="font-medium">Embed as array</span>
            <span className="block text-xs text-gray-500">
              One-to-many: child rows become an array of subdocuments
            </span>
          </span>
        </label>

        <label className="flex items-center gap-2 text-sm cursor-pointer">
          <input
            type="radio"
            name="rel"
            value="single"
            checked={relationship === "single"}
            onChange={() => onChangeRelationship("single")}
            className="text-blue-600"
          />
          <span>
            <span className="font-medium">Embed as single</span>
            <span className="block text-xs text-gray-500">
              One-to-one: child becomes a single subdocument
            </span>
          </span>
        </label>

        <label className="flex items-center gap-2 text-sm cursor-pointer">
          <input
            type="radio"
            name="rel"
            value="reference"
            checked={relationship === "reference"}
            onChange={() => onChangeRelationship("reference")}
            className="text-blue-600"
          />
          <span>
            <span className="font-medium">Reference</span>
            <span className="block text-xs text-gray-500">
              Separate collection linked by ObjectId
            </span>
          </span>
        </label>
      </div>

      <Button variant="danger" onClick={onRemove} className="w-full text-xs">
        Remove Relationship
      </Button>
    </div>
  );
}
