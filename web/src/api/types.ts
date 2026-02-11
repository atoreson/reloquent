export interface StepState {
  status: string;
  completed_at?: string;
}

export interface WizardState {
  current_step: string;
  steps: Record<string, StepState>;
  last_updated: string;
}

export interface StepInfo {
  id: string;
  label: string;
  order: number;
}

export const ALL_STEPS: StepInfo[] = [
  { id: "source_connection", label: "Source Connection", order: 1 },
  { id: "target_connection", label: "Target Connection", order: 2 },
  { id: "table_selection", label: "Table Selection", order: 3 },
  { id: "denormalization", label: "Denormalization Design", order: 4 },
  { id: "type_mapping", label: "Type Mapping", order: 5 },
  { id: "sizing", label: "Sizing", order: 6 },
  { id: "aws_setup", label: "AWS Setup", order: 7 },
  { id: "pre_migration", label: "Pre-Migration", order: 8 },
  { id: "review", label: "Review", order: 9 },
  { id: "migration", label: "Migration", order: 10 },
  { id: "validation", label: "Validation", order: 11 },
  { id: "index_builds", label: "Index Builds", order: 12 },
];

export interface SourceConfig {
  type: string;
  host: string;
  port: number;
  database: string;
  schema?: string;
  username: string;
  password: string;
  ssl: boolean;
}

export interface TargetConfig {
  connection_string: string;
  database: string;
}

export interface ConnectionTestResult {
  success: boolean;
  message?: string;
  error?: string;
}

export interface TopologyInfo {
  type: string;
  is_atlas: boolean;
  shard_count: number;
  server_version: string;
}

export interface TableInfo {
  name: string;
  row_count: number;
  size_bytes: number;
  selected: boolean;
}

export interface Column {
  name: string;
  data_type: string;
  nullable: boolean;
  max_length?: number;
}

export interface Table {
  name: string;
  columns: Column[];
  primary_key?: { name: string; columns: string[] };
  foreign_keys?: ForeignKey[];
  row_count: number;
  size_bytes: number;
}

export interface ForeignKey {
  name: string;
  columns: string[];
  referenced_table: string;
  referenced_columns: string[];
}

export interface Schema {
  database_type: string;
  host: string;
  database: string;
  schema_name?: string;
  tables: Table[];
}

export interface Mapping {
  collections: Collection[];
}

export interface Collection {
  name: string;
  source_table: string;
  embedded?: Embedded[];
  references?: Reference[];
}

export interface Embedded {
  source_table: string;
  field_name: string;
  relationship: string;
  join_column: string;
  parent_column: string;
  embedded?: Embedded[];
}

export interface Reference {
  source_table: string;
  field_name: string;
  join_column: string;
  parent_column: string;
}

export interface TypeMapEntry {
  source_type: string;
  bson_type: string;
  overridden: boolean;
}

export interface SizingPlan {
  spark_plan: {
    platform: string;
    instance_type: string;
    worker_count: number;
    dpu_count: number;
    cost_estimate: string;
    cost_low: number;
    cost_high: number;
  };
  mongo_plan: {
    migration_tier: string;
    production_tier: string;
    storage_gb: number;
    migration_ram_gb: number;
    production_ram_gb: number;
  };
  estimated_time: string;
  explanations: { category: string; summary: string; detail: string }[];
}

export interface AWSConfig {
  region: string;
  profile: string;
  s3_bucket: string;
  platform: string;
}

// Step ID → route path mapping
export const STEP_ROUTES: Record<string, string> = {
  source_connection: "/source",
  target_connection: "/target",
  table_selection: "/tables",
  denormalization: "/design",
  type_mapping: "/types",
  sizing: "/sizing",
  aws_setup: "/aws",
  pre_migration: "/prepare",
  review: "/review",
  migration: "/migration",
  validation: "/validation",
  index_builds: "/indexes",
};
