interface AlertProps {
  type: "error" | "warning" | "success" | "info";
  children: React.ReactNode;
}

const styles = {
  error: "bg-red-50 border-red-200 text-red-800",
  warning: "bg-yellow-50 border-yellow-200 text-yellow-800",
  success: "bg-green-50 border-green-200 text-green-800",
  info: "bg-blue-50 border-blue-200 text-blue-800",
};

export function Alert({ type, children }: AlertProps) {
  return (
    <div className={`rounded-md border p-4 text-sm ${styles[type]}`}>
      {children}
    </div>
  );
}
