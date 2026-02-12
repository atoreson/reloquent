import { useState } from "react";
import { FormField, Input, Select } from "../components/FormField";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import { useConfigureAWS, useNavigateToStep } from "../api/hooks";
import type { AWSConfig } from "../api/types";

const AWS_REGIONS = [
  "us-east-1",
  "us-east-2",
  "us-west-1",
  "us-west-2",
  "eu-west-1",
  "eu-west-2",
  "eu-central-1",
  "ap-southeast-1",
  "ap-southeast-2",
  "ap-northeast-1",
];

export default function AWSSetup() {
  const [form, setForm] = useState<AWSConfig>({
    region: "us-east-1",
    profile: "default",
    s3_bucket: "",
    platform: "emr",
  });

  const configureAWS = useConfigureAWS();
  const goToStep = useNavigateToStep();

  const update = (field: keyof AWSConfig, value: string) => {
    setForm((f) => ({ ...f, [field]: value }));
  };

  const handleSave = () => {
    configureAWS.mutate(form, {
      onSuccess: () => goToStep("pre_migration"),
    });
  };

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900">AWS Setup</h2>
      <p className="mt-2 text-gray-600">
        Configure AWS infrastructure for the Spark migration job.
      </p>

      <div className="mt-8 space-y-6 rounded-lg border border-gray-200 bg-white p-6">
        <div className="grid grid-cols-2 gap-4">
          <FormField label="AWS Region">
            <Select
              value={form.region}
              onChange={(e) => update("region", e.target.value)}
            >
              {AWS_REGIONS.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </Select>
          </FormField>
          <FormField label="AWS Profile" help="From ~/.aws/credentials">
            <Input
              value={form.profile}
              onChange={(e) => update("profile", e.target.value)}
              placeholder="default"
            />
          </FormField>
        </div>

        <FormField label="S3 Bucket" help="For PySpark scripts and artifacts">
          <Input
            value={form.s3_bucket}
            onChange={(e) => update("s3_bucket", e.target.value)}
            placeholder="my-migration-bucket"
          />
        </FormField>

        <FormField label="Platform">
          <Select
            value={form.platform}
            onChange={(e) => update("platform", e.target.value)}
          >
            <option value="emr">Amazon EMR</option>
            <option value="glue">AWS Glue</option>
          </Select>
        </FormField>

        {configureAWS.error && (
          <Alert type="error">{configureAWS.error.message}</Alert>
        )}

        <div className="flex gap-3 pt-4 border-t border-gray-200">
          <Button onClick={handleSave} loading={configureAWS.isPending}>
            Save & Continue
          </Button>
        </div>
      </div>
    </div>
  );
}
