import { Select } from "./FormField";

const BSON_TYPES = [
  "NumberLong",
  "Decimal128",
  "String",
  "ISODate",
  "BinData",
  "Document",
  "Array",
  "Boolean",
  "Double",
];

interface TypeSelectProps {
  value: string;
  onChange: (value: string) => void;
}

export function TypeSelect({ value, onChange }: TypeSelectProps) {
  return (
    <Select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-36"
    >
      {BSON_TYPES.map((t) => (
        <option key={t} value={t}>
          {t}
        </option>
      ))}
    </Select>
  );
}
