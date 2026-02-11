import { NavLink } from "react-router-dom";
import { useWizardState } from "../api/hooks";
import { ALL_STEPS, STEP_ROUTES } from "../api/types";

const POINT_OF_NO_RETURN_STEPS = new Set([
  "migration",
  "validation",
  "index_builds",
]);

function stepStatusIcon(status: string | undefined, isCurrent: boolean) {
  if (isCurrent) return "●";
  if (status === "complete") return "✓";
  return "○";
}

function stepStatusClass(status: string | undefined, isCurrent: boolean) {
  if (isCurrent)
    return "bg-blue-50 text-blue-700 border-blue-200 font-medium";
  if (status === "complete") return "text-gray-500";
  return "text-gray-400";
}

export function Sidebar() {
  const { data: state } = useWizardState();
  const currentStep = state?.current_step;

  // Find current step order to determine if we've passed the point of no return
  const currentOrder = ALL_STEPS.find((s) => s.id === currentStep)?.order ?? 0;
  const pastPointOfNoReturn = currentOrder >= 10; // migration step

  return (
    <aside className="w-64 shrink-0 border-r border-gray-200 bg-white">
      <div className="p-4 border-b border-gray-200">
        <h1 className="text-xl font-bold text-gray-900">Reloquent</h1>
        <p className="text-xs text-gray-500 mt-0.5">Migration Wizard</p>
      </div>
      <nav className="p-2">
        <ol className="space-y-0.5">
          {ALL_STEPS.map((step, idx) => {
            const route = STEP_ROUTES[step.id];
            const stepState = state?.steps[step.id];
            const isCurrent = currentStep === step.id;
            const isNoReturn = POINT_OF_NO_RETURN_STEPS.has(step.id);

            // Show separator before point-of-no-return steps
            const showSeparator =
              isNoReturn &&
              idx > 0 &&
              !POINT_OF_NO_RETURN_STEPS.has(ALL_STEPS[idx - 1].id);

            return (
              <li key={step.id}>
                {showSeparator && (
                  <div className="flex items-center gap-2 px-3 py-1.5 my-1">
                    <div className="flex-1 h-px bg-orange-200" />
                    <span className="text-[10px] text-orange-500 font-medium uppercase tracking-wider">
                      {pastPointOfNoReturn ? "Active" : "Point of no return"}
                    </span>
                    <div className="flex-1 h-px bg-orange-200" />
                  </div>
                )}
                <NavLink
                  to={route}
                  className={({ isActive }) =>
                    `flex items-center gap-2 rounded px-3 py-2 text-sm transition-colors ${stepStatusClass(stepState?.status, isCurrent)} ${isActive ? "bg-gray-100" : "hover:bg-gray-50"}`
                  }
                >
                  <span className="w-4 text-center text-xs">
                    {stepStatusIcon(stepState?.status, isCurrent)}
                  </span>
                  <span className="flex-1">{step.label}</span>
                  <span className="text-xs text-gray-400">{step.order}</span>
                </NavLink>
              </li>
            );
          })}
        </ol>
      </nav>
    </aside>
  );
}
