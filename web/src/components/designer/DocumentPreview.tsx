interface DocumentPreviewProps {
  json: string;
  collectionName?: string;
}

export function DocumentPreview({
  json,
  collectionName,
}: DocumentPreviewProps) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white">
      <div className="border-b border-gray-200 bg-gray-50 px-4 py-2">
        <h3 className="text-sm font-medium text-gray-700">
          Document Preview
          {collectionName && (
            <span className="ml-2 text-gray-500 font-normal">
              {collectionName}
            </span>
          )}
        </h3>
      </div>
      <pre className="p-4 text-xs text-gray-800 overflow-auto max-h-96 font-mono">
        {json}
      </pre>
    </div>
  );
}
