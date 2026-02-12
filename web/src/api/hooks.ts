import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { useCallback } from "react";
import { api } from "./client";
import type {
  WizardState,
  ConnectionTestResult,
  Schema,
  TableInfo,
  Mapping,
  TypeMapEntry,
  SizingPlan,
  SourceConfig,
  TargetConfig,
  AWSConfig,
} from "./types";
import { STEP_ROUTES } from "./types";

export function useWizardState() {
  return useQuery<WizardState>({
    queryKey: ["state"],
    queryFn: () => api.get("/api/state"),
    refetchInterval: 5000,
  });
}

export function useSetStep() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (step: string) => api.put("/api/state/step", { step }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["state"] }),
  });
}

export function useNavigateToStep() {
  const setStep = useSetStep();
  const nav = useNavigate();
  return useCallback(
    (step: string) => {
      setStep.mutate(step, {
        onSuccess: () => nav(STEP_ROUTES[step]),
      });
    },
    [setStep, nav],
  );
}

export function useSourceConfig() {
  return useQuery<SourceConfig>({
    queryKey: ["sourceConfig"],
    queryFn: () => api.get("/api/source/config"),
  });
}

export function useTestSourceConnection() {
  return useMutation<ConnectionTestResult, Error, SourceConfig>({
    mutationFn: (cfg) => api.post("/api/source/test-connection", cfg),
  });
}

export function useDiscoverSchema() {
  const qc = useQueryClient();
  return useMutation<Schema, Error, SourceConfig>({
    mutationFn: (cfg) => api.post("/api/source/discover", cfg),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["schema"] });
      qc.invalidateQueries({ queryKey: ["tables"] });
    },
  });
}

export function useSchema() {
  return useQuery<Schema>({
    queryKey: ["schema"],
    queryFn: () => api.get("/api/source/schema"),
    retry: false,
  });
}

export function useTargetConfig() {
  return useQuery<TargetConfig>({
    queryKey: ["targetConfig"],
    queryFn: () => api.get("/api/target/config"),
  });
}

export function useTestTargetConnection() {
  return useMutation<ConnectionTestResult, Error, TargetConfig>({
    mutationFn: (cfg) => api.post("/api/target/test-connection", cfg),
  });
}

export function useDetectTopology() {
  return useMutation({
    mutationFn: (cfg: TargetConfig) =>
      api.post("/api/target/detect-topology", cfg),
  });
}

export function useTables() {
  return useQuery<TableInfo[]>({
    queryKey: ["tables"],
    queryFn: () => api.get("/api/tables"),
    retry: false,
  });
}

export function useSelectTables() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (tables: string[]) =>
      api.post("/api/tables/select", { tables }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tables"] }),
  });
}

export function useMapping() {
  return useQuery<Mapping>({
    queryKey: ["mapping"],
    queryFn: () => api.get("/api/mapping"),
    retry: false,
  });
}

export function useMappingPreview() {
  return useQuery<Mapping>({
    queryKey: ["mappingPreview"],
    queryFn: () => api.get("/api/mapping/preview"),
    retry: false,
  });
}

export function useSaveMapping() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (mapping: Mapping) => api.post("/api/mapping", mapping),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["mapping"] }),
  });
}

export function useTypeMap() {
  return useQuery<TypeMapEntry[]>({
    queryKey: ["typemap"],
    queryFn: () => api.get("/api/typemap"),
    retry: false,
  });
}

export function useSaveTypeMap() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (overrides: Record<string, string>) =>
      api.post("/api/typemap", overrides),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["typemap"] }),
  });
}

export function useSizing() {
  return useQuery<SizingPlan>({
    queryKey: ["sizing"],
    queryFn: () => api.get("/api/sizing"),
    retry: false,
  });
}

export function useConfigureAWS() {
  return useMutation({
    mutationFn: (cfg: AWSConfig) => api.post("/api/aws/configure", cfg),
  });
}
