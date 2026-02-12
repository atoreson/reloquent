import { useQuery } from "@tanstack/react-query";
import { ValidationResultCard } from "../components/ValidationResult";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import { PageContainer } from "../components/PageContainer";
import { useNavigateToStep } from "../api/hooks";
import { api } from "../api/client";

interface ValidationResults {
  status: string;
  collections: {
    name: string;
    row_count_check?: { passed: boolean };
    sample_check?: { passed: boolean };
    aggregate_check?: { passed: boolean };
    status: string;
  }[];
}

export default function Validation() {
  const goToStep = useNavigateToStep();

  const { data: results } = useQuery<ValidationResults>({
    queryKey: ["validation-results"],
    queryFn: () => api.get("/api/validation/results"),
    refetchInterval: 3000,
    retry: false,
  });

  const allPassed = results?.status === "pass";

  return (
    <PageContainer>
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Validation</h2>
      <p className="mt-2 text-gray-600">
        Validate the migrated data by comparing source and target.
      </p>

      {results && (
        <div className="mt-6 space-y-4">
          {allPassed ? (
            <Alert type="success">
              All validation checks passed. Data integrity verified.
            </Alert>
          ) : results.status === "fail" ? (
            <Alert type="error">
              Some validation checks failed. Review the results below.
            </Alert>
          ) : (
            <Alert type="info">Validation in progress...</Alert>
          )}

          <div className="grid grid-cols-2 gap-4">
            {results.collections.map((col) => (
              <ValidationResultCard
                key={col.name}
                collection={col.name}
                rowCountPassed={col.row_count_check?.passed}
                samplePassed={col.sample_check?.passed}
                aggregatePassed={col.aggregate_check?.passed}
                status={col.status}
              />
            ))}
          </div>

          {(allPassed || results.status === "fail") && (
            <div className="flex gap-3">
              <Button onClick={() => goToStep("index_builds")}>
                Continue to Index Builds
              </Button>
            </div>
          )}
        </div>
      )}

      {!results && (
        <div className="mt-6">
          <Alert type="info">
            Waiting for validation results...
          </Alert>
        </div>
      )}
    </div>
    </PageContainer>
  );
}
