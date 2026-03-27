export interface Server {
  id: string;
  name: string;
  host?: string;
  api_key?: string;
  status: string;
  last_seen_at?: string;
  notes?: string;
}

export interface AlertRule {
  id: string;
  server_id?: string;
  metric: string;
  operator: string;
  threshold: number;
  duration_seconds?: number;
  cooldown_seconds?: number;
  channel_id?: string;
  docker_container?: string;
  enabled: boolean;
}

export interface NotificationChannel {
  id: string;
  type: string;
  name: string;
  config: string;
}

export interface LogSource {
  id: string;
  server_id: string;
  name: string;
  type: string;
  path?: string;
  container?: string;
  format?: string;
  enabled: boolean;
  ingest_levels?: string;
}

export interface UptimeCheck {
  id: string;
  name: string;
  url: string;
  method: string;
  interval_seconds: number;
  timeout_seconds: number;
  expected_status: number;
  enabled: boolean;
}
