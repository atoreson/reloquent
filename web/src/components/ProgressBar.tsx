interface ProgressBarProps {
  percent: number;
  label?: string;
  color?: "blue" | "green" | "yellow" | "red";
  animated?: boolean;
}

const colors = {
  blue: "bg-blue-500",
  green: "bg-green-500",
  yellow: "bg-yellow-500",
  red: "bg-red-500",
};

export function ProgressBar({
  percent,
  label,
  color = "blue",
  animated = false,
}: ProgressBarProps) {
  const clamped = Math.min(100, Math.max(0, percent));

  return (
    <div>
      {label && (
        <div className="flex justify-between text-xs text-gray-600 mb-1">
          <span>{label}</span>
          <span>{clamped.toFixed(1)}%</span>
        </div>
      )}
      <div className="h-2 w-full rounded-full bg-gray-200 overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ${colors[color]} ${
            animated ? "animate-pulse" : ""
          }`}
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  );
}
