interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "danger";
  loading?: boolean;
}

const variants = {
  primary:
    "bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500 disabled:bg-blue-400",
  secondary:
    "bg-white text-gray-700 border border-gray-300 hover:bg-gray-50 focus:ring-blue-500",
  danger:
    "bg-red-600 text-white hover:bg-red-700 focus:ring-red-500 disabled:bg-red-400",
};

export function Button({
  variant = "primary",
  loading = false,
  disabled,
  children,
  className = "",
  ...props
}: ButtonProps) {
  return (
    <button
      disabled={disabled || loading}
      className={`inline-flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:cursor-not-allowed ${variants[variant]} ${className}`}
      {...props}
    >
      {loading && (
        <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
      )}
      {children}
    </button>
  );
}
