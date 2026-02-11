import { useMemo } from "react";
import type { Mapping, Schema, Collection, Embedded } from "../api/types";

function buildPreview(
  collection: Collection,
  schema: Schema,
): Record<string, unknown> {
  const table = schema.tables.find(
    (t) => t.name === collection.source_table,
  );
  if (!table) return { _id: "ObjectId(...)" };

  const doc: Record<string, unknown> = { _id: "ObjectId(...)" };
  for (const col of table.columns) {
    doc[col.name] = sampleValue(col.data_type);
  }

  if (collection.embedded) {
    for (const emb of collection.embedded) {
      doc[emb.field_name] = buildEmbeddedPreview(emb, schema);
    }
  }

  if (collection.references) {
    for (const ref of collection.references) {
      doc[ref.field_name] = "ObjectId(...)";
    }
  }

  return doc;
}

function buildEmbeddedPreview(
  emb: Embedded,
  schema: Schema,
): unknown {
  const table = schema.tables.find((t) => t.name === emb.source_table);
  const subdoc: Record<string, unknown> = {};
  if (table) {
    for (const col of table.columns) {
      if (col.name !== emb.join_column) {
        subdoc[col.name] = sampleValue(col.data_type);
      }
    }
  }

  if (emb.embedded) {
    for (const nested of emb.embedded) {
      subdoc[nested.field_name] = buildEmbeddedPreview(nested, schema);
    }
  }

  return emb.relationship === "array" ? [subdoc, "..."] : subdoc;
}

function sampleValue(dataType: string): unknown {
  const lower = dataType.toLowerCase();
  if (lower.includes("int") || lower.includes("serial") || lower === "number")
    return 42;
  if (lower.includes("numeric") || lower.includes("decimal")) return "123.45";
  if (lower.includes("bool")) return true;
  if (lower.includes("date") || lower.includes("timestamp"))
    return "ISODate(...)";
  if (lower.includes("json")) return { key: "value" };
  if (lower.includes("bytea") || lower.includes("blob"))
    return "BinData(...)";
  return '"..."';
}

export function useDocumentPreview(
  mapping: Mapping | undefined,
  schema: Schema | undefined,
  selectedCollection?: string,
): string {
  return useMemo(() => {
    if (!mapping || !schema) return "{}";

    const collection = selectedCollection
      ? mapping.collections.find((c) => c.name === selectedCollection)
      : mapping.collections[0];

    if (!collection) return "{}";

    const preview = buildPreview(collection, schema);
    return JSON.stringify(preview, null, 2);
  }, [mapping, schema, selectedCollection]);
}
