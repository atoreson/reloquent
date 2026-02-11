import { Alert } from "../Alert";

interface SizeWarningProps {
  estimatedBytes: number;
  collectionName: string;
}

const BSON_LIMIT = 16 * 1024 * 1024; // 16MB

export function SizeWarning({
  estimatedBytes,
  collectionName,
}: SizeWarningProps) {
  if (estimatedBytes < BSON_LIMIT * 0.5) return null;

  const pct = ((estimatedBytes / BSON_LIMIT) * 100).toFixed(0);
  const isOver = estimatedBytes >= BSON_LIMIT;

  return (
    <Alert type={isOver ? "error" : "warning"}>
      <strong>{collectionName}:</strong> Estimated worst-case document size is{" "}
      {(estimatedBytes / (1024 * 1024)).toFixed(1)} MB ({pct}% of 16 MB BSON
      limit).
      {isOver
        ? " Documents will exceed the 16 MB limit. Consider using references instead of embedding."
        : " Consider monitoring document growth."}
    </Alert>
  );
}
