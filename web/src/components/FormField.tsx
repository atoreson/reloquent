interface FormFieldProps {
  label: string;
  error?: string;
  help?: string;
  children: React.ReactNode;
}

export function FormField({ label, error, help, children }: FormFieldProps) {
  return (
    <div className="space-y-1">
      <label className="block text-sm font-medium text-gray-700">
        {label}
      </label>
      {children}
      {error && <p className="text-sm text-red-600">{error}</p>}
      {help && !error && <p className="text-sm text-gray-500">{help}</p>}
    </div>
  );
}

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  hasError?: boolean;
}

export function Input({ hasError, className = "", ...props }: InputProps) {
  return (
    <input
      className={`block w-full rounded-md border px-3 py-2 text-sm shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 ${
        hasError
          ? "border-red-300 focus:border-red-500"
          : "border-gray-300 focus:border-blue-500"
      } ${className}`}
      {...props}
    />
  );
}

interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  hasError?: boolean;
}

export function Select({ hasError, className = "", children, ...props }: SelectProps) {
  return (
    <select
      className={`block w-full rounded-md border px-3 py-2 text-sm shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 ${
        hasError
          ? "border-red-300 focus:border-red-500"
          : "border-gray-300 focus:border-blue-500"
      } ${className}`}
      {...props}
    >
      {children}
    </select>
  );
}
