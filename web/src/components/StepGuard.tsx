import { Navigate, useLocation } from "react-router-dom";
import { useWizardState } from "../api/hooks";
import { ALL_STEPS, STEP_ROUTES } from "../api/types";

interface StepGuardProps {
  stepId: string;
  children: React.ReactNode;
}

export function StepGuard({ stepId, children }: StepGuardProps) {
  const { data: state, isLoading } = useWizardState();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  if (!state) return null;

  const stepOrder = ALL_STEPS.map((s) => s.id);
  const currentIdx = stepOrder.indexOf(state.current_step);
  const targetIdx = stepOrder.indexOf(stepId);

  // Allow visiting current step or any completed step
  if (targetIdx <= currentIdx) {
    return <>{children}</>;
  }

  // Redirect to current step
  const currentRoute = STEP_ROUTES[state.current_step];
  if (location.pathname !== currentRoute) {
    return <Navigate to={currentRoute} replace />;
  }

  return <>{children}</>;
}
