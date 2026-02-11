import { useState } from "react";
import { Input } from "./FormField";

interface PasswordFieldProps {
  value: string;
  onChange: (value: string) => void;
  hasError?: boolean;
  placeholder?: string;
}

export function PasswordField({
  value,
  onChange,
  hasError,
  placeholder,
}: PasswordFieldProps) {
  const [visible, setVisible] = useState(false);

  return (
    <div className="relative">
      <Input
        type={visible ? "text" : "password"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        hasError={hasError}
        placeholder={placeholder}
        className="pr-10"
      />
      <button
        type="button"
        onClick={() => setVisible(!visible)}
        className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-gray-500 hover:text-gray-700"
      >
        {visible ? "Hide" : "Show"}
      </button>
    </div>
  );
}
