import { useState, useEffect } from "react";
import { FormField, Input } from "../components/FormField";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import { StatusBadge } from "../components/StatusBadge";
import {
  useTargetConfig,
  useTestTargetConnection,
  useDetectTopology,
  useNavigateToStep,
} from "../api/hooks";
import type { TargetConfig, TopologyInfo } from "../api/types";

export default function TargetConnection() {
  const [form, setForm] = useState<TargetConfig>({
    connection_string: "",
    database: "",
  });

  const [topology, setTopology] = useState<TopologyInfo | null>(null);

  const targetConfig = useTargetConfig();
  const testConn = useTestTargetConnection();
  const detectTopo = useDetectTopology();
  const goToStep = useNavigateToStep();

  useEffect(() => {
    if (targetConfig.data && targetConfig.data.connection_string) {
      setForm((f) => ({ ...f, ...targetConfig.data }));
    }
  }, [targetConfig.data]);

  const handleTestConnection = () => {
    testConn.mutate(form, {
      onSuccess: (result) => {
        if (result.success) {
          detectTopo.mutate(form, {
            onSuccess: (topo) => setTopology(topo as TopologyInfo),
          });
        }
      },
    });
  };

  const handleContinue = () => {
    goToStep("aws_setup");
  };

  const isConnected = testConn.data?.success === true;

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Target Connection</h2>
      <p className="mt-2 text-gray-600">
        Configure the target MongoDB connection.
      </p>

      <div className="mt-8 space-y-6 rounded-lg border border-gray-200 bg-white p-6">
        <FormField
          label="Connection String"
          help="mongodb:// or mongodb+srv:// URI"
        >
          <Input
            value={form.connection_string}
            onChange={(e) =>
              setForm((f) => ({ ...f, connection_string: e.target.value }))
            }
            placeholder="mongodb://localhost:27017"
          />
        </FormField>

        <FormField label="Database Name">
          <Input
            value={form.database}
            onChange={(e) =>
              setForm((f) => ({ ...f, database: e.target.value }))
            }
            placeholder="my_database"
          />
        </FormField>

        {testConn.data && (
          <div>
            {testConn.data.success ? (
              <Alert type="success">
                <StatusBadge status="pass" /> Connection successful
              </Alert>
            ) : (
              <Alert type="error">{testConn.data.error}</Alert>
            )}
          </div>
        )}

        {topology && (
          <div className="rounded-md border border-gray-200 bg-gray-50 p-4">
            <h3 className="text-sm font-medium text-gray-700 mb-2">
              Topology
            </h3>
            <dl className="grid grid-cols-2 gap-2 text-sm">
              <dt className="text-gray-500">Type</dt>
              <dd className="text-gray-900">{topology.type}</dd>
              <dt className="text-gray-500">Version</dt>
              <dd className="text-gray-900">{topology.server_version}</dd>
              {topology.is_atlas && (
                <>
                  <dt className="text-gray-500">Platform</dt>
                  <dd className="text-gray-900">MongoDB Atlas</dd>
                </>
              )}
              {topology.shard_count > 0 && (
                <>
                  <dt className="text-gray-500">Shards</dt>
                  <dd className="text-gray-900">{topology.shard_count}</dd>
                </>
              )}
            </dl>
          </div>
        )}

        {testConn.error && (
          <Alert type="error">{testConn.error.message}</Alert>
        )}

        <div className="flex gap-3 pt-4 border-t border-gray-200">
          <Button
            variant="secondary"
            onClick={handleTestConnection}
            loading={testConn.isPending || detectTopo.isPending}
          >
            Test Connection
          </Button>
          <Button onClick={handleContinue} disabled={!isConnected}>
            Continue to AWS Setup
          </Button>
        </div>
      </div>
    </div>
  );
}
