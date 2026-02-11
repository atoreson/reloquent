import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Layout } from "./components/Layout";
import { StepGuard } from "./components/StepGuard";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { useWebSocket } from "./api/useWebSocket";
import SourceConnection from "./pages/SourceConnection";
import TargetConnection from "./pages/TargetConnection";
import TableSelection from "./pages/TableSelection";
import DenormDesign from "./pages/DenormDesign";
import TypeMapping from "./pages/TypeMapping";
import Sizing from "./pages/Sizing";
import AWSSetup from "./pages/AWSSetup";
import PreMigration from "./pages/PreMigration";
import Review from "./pages/Review";
import Migration from "./pages/Migration";
import Validation from "./pages/Validation";
import IndexBuilds from "./pages/IndexBuilds";
import Readiness from "./pages/Readiness";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 10_000,
      retry: 1,
    },
  },
});

function GuardedStep({
  stepId,
  children,
}: {
  stepId: string;
  children: React.ReactNode;
}) {
  return <StepGuard stepId={stepId}>{children}</StepGuard>;
}

function WebSocketInit({ children }: { children: React.ReactNode }) {
  useWebSocket();
  return <>{children}</>;
}

function App() {
  return (
    <ErrorBoundary>
    <QueryClientProvider client={queryClient}>
      <WebSocketInit>
      <BrowserRouter>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<Navigate to="/source" replace />} />
            <Route
              path="/source"
              element={
                <GuardedStep stepId="source_connection">
                  <SourceConnection />
                </GuardedStep>
              }
            />
            <Route
              path="/target"
              element={
                <GuardedStep stepId="target_connection">
                  <TargetConnection />
                </GuardedStep>
              }
            />
            <Route
              path="/tables"
              element={
                <GuardedStep stepId="table_selection">
                  <TableSelection />
                </GuardedStep>
              }
            />
            <Route
              path="/design"
              element={
                <GuardedStep stepId="denormalization">
                  <DenormDesign />
                </GuardedStep>
              }
            />
            <Route
              path="/types"
              element={
                <GuardedStep stepId="type_mapping">
                  <TypeMapping />
                </GuardedStep>
              }
            />
            <Route
              path="/sizing"
              element={
                <GuardedStep stepId="sizing">
                  <Sizing />
                </GuardedStep>
              }
            />
            <Route
              path="/aws"
              element={
                <GuardedStep stepId="aws_setup">
                  <AWSSetup />
                </GuardedStep>
              }
            />
            <Route
              path="/prepare"
              element={
                <GuardedStep stepId="pre_migration">
                  <PreMigration />
                </GuardedStep>
              }
            />
            <Route
              path="/review"
              element={
                <GuardedStep stepId="review">
                  <Review />
                </GuardedStep>
              }
            />
            <Route
              path="/migration"
              element={
                <GuardedStep stepId="migration">
                  <Migration />
                </GuardedStep>
              }
            />
            <Route
              path="/validation"
              element={
                <GuardedStep stepId="validation">
                  <Validation />
                </GuardedStep>
              }
            />
            <Route
              path="/indexes"
              element={
                <GuardedStep stepId="index_builds">
                  <IndexBuilds />
                </GuardedStep>
              }
            />
            <Route
              path="/readiness"
              element={
                <GuardedStep stepId="complete">
                  <Readiness />
                </GuardedStep>
              }
            />
            <Route path="*" element={<Navigate to="/source" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
      </WebSocketInit>
    </QueryClientProvider>
    </ErrorBoundary>
  );
}

export default App;
