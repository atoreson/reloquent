import { useState, useEffect } from "react";
import { FormField, Input, Select } from "../components/FormField";
import { PasswordField } from "../components/PasswordField";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import { StatusBadge } from "../components/StatusBadge";
import {
  useSourceConfig,
  useTestSourceConnection,
  useDiscoverSchema,
  useSetStep,
} from "../api/hooks";
import type { SourceConfig } from "../api/types";

const DEFAULT_PORTS: Record<string, number> = {
  postgresql: 5432,
  oracle: 1521,
};

export default function SourceConnection() {
  const [form, setForm] = useState<SourceConfig>({
    type: "postgresql",
    host: "",
    port: 5432,
    database: "",
    schema: "public",
    username: "",
    password: "",
    ssl: false,
  });

  const sourceConfig = useSourceConfig();
  const testConn = useTestSourceConnection();
  const discover = useDiscoverSchema();
  const setStep = useSetStep();

  useEffect(() => {
    if (sourceConfig.data && sourceConfig.data.host) {
      setForm((f) => ({ ...f, ...sourceConfig.data }));
    }
  }, [sourceConfig.data]);

  const update = (field: keyof SourceConfig, value: string | number | boolean) => {
    setForm((f) => ({ ...f, [field]: value }));
  };

  const handleTypeChange = (type: string) => {
    setForm((f) => ({
      ...f,
      type,
      port: DEFAULT_PORTS[type] || f.port,
      schema: type === "postgresql" ? "public" : "",
    }));
  };

  const handleTestConnection = () => {
    testConn.mutate(form);
  };

  const handleDiscover = () => {
    discover.mutate(form, {
      onSuccess: () => {
        setStep.mutate("target_connection");
      },
    });
  };

  const isConnected = testConn.data?.success === true;

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Source Connection</h2>
      <p className="mt-2 text-gray-600">
        Configure the source relational database connection.
      </p>

      <div className="mt-8 space-y-6 rounded-lg border border-gray-200 bg-white p-6">
        <FormField label="Database Type">
          <Select
            value={form.type}
            onChange={(e) => handleTypeChange(e.target.value)}
          >
            <option value="postgresql">PostgreSQL</option>
            <option value="oracle">Oracle</option>
          </Select>
        </FormField>

        <div className="grid grid-cols-3 gap-4">
          <div className="col-span-2">
            <FormField label="Host">
              <Input
                value={form.host}
                onChange={(e) => update("host", e.target.value)}
                placeholder="localhost"
              />
            </FormField>
          </div>
          <FormField label="Port">
            <Input
              type="number"
              value={form.port}
              onChange={(e) => update("port", parseInt(e.target.value) || 0)}
            />
          </FormField>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <FormField label="Database">
            <Input
              value={form.database}
              onChange={(e) => update("database", e.target.value)}
              placeholder="mydb"
            />
          </FormField>
          <FormField label="Schema" help="PostgreSQL schema name">
            <Input
              value={form.schema || ""}
              onChange={(e) => update("schema", e.target.value)}
              placeholder="public"
            />
          </FormField>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <FormField label="Username">
            <Input
              value={form.username}
              onChange={(e) => update("username", e.target.value)}
            />
          </FormField>
          <FormField label="Password">
            <PasswordField
              value={form.password}
              onChange={(v) => update("password", v)}
            />
          </FormField>
        </div>

        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={form.ssl}
            onChange={(e) => update("ssl", e.target.checked)}
            className="rounded border-gray-300"
          />
          <span className="text-gray-700">Use SSL/TLS</span>
        </label>

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

        {testConn.error && (
          <Alert type="error">{testConn.error.message}</Alert>
        )}

        {discover.error && (
          <Alert type="error">{discover.error.message}</Alert>
        )}

        <div className="flex gap-3 pt-4 border-t border-gray-200">
          <Button
            variant="secondary"
            onClick={handleTestConnection}
            loading={testConn.isPending}
          >
            Test Connection
          </Button>
          <Button
            onClick={handleDiscover}
            loading={discover.isPending}
            disabled={!isConnected}
          >
            Discover Schema & Continue
          </Button>
        </div>
      </div>
    </div>
  );
}
