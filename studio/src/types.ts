export interface TemplateEntry {
  name: string;
  path: string;
  category: 'core' | 'calibration' | 'multimodal' | string;
  description: string;
  variables: string[];
  raw_source: string;
}

export interface GPUHealthStatus {
  gateway_status: string;
  upstream_url: string;
  gpu_state: 'warm_and_ready' | 'warming_up' | 'scaled_to_zero' | 'unreachable' | string;
  gpu_available: boolean;
  gpu_hardware: string;
  vram_gb: number;
  multimodal_vision: boolean;
  probe_latency_ms: number;
  warmup_in_progress?: boolean;
  warmup_elapsed_seconds?: number;
  warmup_phase?: string;
  warmup_phase_label?: string;
  warmup_bytes_staged_gb?: number;
  ewma_wake_seconds?: number;
  seconds_since_last_read?: number;
  idle_remaining_seconds?: number;
  last_readout_ms?: number;
  estimated_wake_seconds?: number;
  message: string;
  checked_at: string;
}

export interface AuthMeResponse {
  authenticated: boolean;
  email: string;
  user_id?: string;
  auth_mode: string;
}

export interface QuestionAnswer {
  type: string;
  label?: string;
  choice?: string;
  score?: number;
  level?: string;
  noul?: number;
  confidence: number;
  logprob?: number;
  entropy?: number;
  stderr?: number;
  agreement?: number;
  probabilities?: Record<string, number>;
}

export interface QuestionDiagnostic {
  entropy?: number | number[];
  first_read_max_entropy?: number;
  first_read_mean_entropy?: number;
}

export interface DecisionDiagnostics {
  hole?: string;
  steps?: number;
  timing?: {
    total_ms?: number;
    reads?: number;
    denoise_ms?: number;
    prefill_ms?: number;
    prompt_tokens?: number;
    reused_tokens?: number;
    rounds?: number;
    samples?: number;
    steps_run?: number;
  };
  questions?: Record<string, QuestionDiagnostic>;
}

export interface DecideAPIResponse {
  template: string;
  variables?: Record<string, string>;
  wall_time_ms: number;
  stats?: {
    model?: string;
    endpoint?: string;
    prompt_tokens?: number;
    output_tokens?: number;
    total_tokens?: number;
  };
  decision: {
    answers: Record<string, QuestionAnswer>;
    diagnostics?: DecisionDiagnostics;
  };
}

export interface PresetSample {
  id: string;
  title: string;
  badge: string;
  template: string;
  description: string;
  variables: Record<string, string>;
}

export interface MCPToolSpec {
  name: string;
  badge: string;
  description: string;
  defaultArgs: Record<string, unknown>;
}
