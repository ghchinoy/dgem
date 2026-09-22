export interface TemplateCatalogEntry {
  id: string;
  name: string;
  category: string;
  path: string;
  variables: string[];
  sample_vars: Record<string, string>;
  multimodal: boolean;
  raw_template: string;
}

export interface QuestionAnswer {
  type: string;
  label: string;
  value?: unknown;
  choice?: string;
  confidence: number;
  logprob?: number;
  entropy?: number;
  stderr: number;
  agreement: number;
  probabilities?: Record<string, number>;
}

export interface QuestionDiagnostic {
  argmax_token: string;
  entropy: number;
  label_mass: number;
}

export interface GatewayDecideResponse {
  template: string;
  answers: Record<string, QuestionAnswer>;
  diagnostics?: {
    questions?: Record<string, QuestionDiagnostic>;
  };
  max_entropy: number;
  wall_time_ms: number;
  warmup_attempts: number;
  model: string;
  upstream_url: string;
}

export interface GatewayStatusResponse {
  status: string;
  reachable: boolean;
  http_status?: number;
  upstream_url: string;
  detail?: string;
}
