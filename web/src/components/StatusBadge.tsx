interface StatusBadgeProps {
  status: "pass" | "fail" | "pending" | "warning";
  label?: string;
}

const styles = {
  pass: "bg-green-100 text-green-800",
  fail: "bg-red-100 text-red-800",
  pending: "bg-gray-100 text-gray-600",
  warning: "bg-yellow-100 text-yellow-800",
};

const icons = {
  pass: "✓",
  fail: "✗",
  pending: "○",
  warning: "!",
};

export function StatusBadge({ status, label }: StatusBadgeProps) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${styles[status]}`}
    >
      <span>{icons[status]}</span>
      {label && <span>{label}</span>}
    </span>
  );
}
