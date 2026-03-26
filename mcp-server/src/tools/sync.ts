import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";

interface SyncManifest {
  since: string;
  changes: {
    tasks: unknown[];
    goals: unknown[];
    projects: unknown[];
    metrics: unknown[];
  };
  export_only: {
    signals: unknown[];
    recommendations: unknown[];
    working_memory: unknown;
  };
}

interface SyncResultRequest {
  storage_type: string;
  items_pushed: number;
  items_pulled: number;
  conflicts: number;
  errors?: string[];
  pulled_items?: Array<{
    entity_type: string;
    external_id: string;
    data: Record<string, unknown>;
  }>;
}

interface SyncConfig {
  storage_type: string;
  is_enabled: boolean;
  entity_types: string[];
  sync_frequency_minutes?: number;
  config?: Record<string, unknown>;
}

export async function getSyncManifest(
  client: ApiClient,
  storageType: string,
): Promise<CallToolResult> {
  try {
    const data = await client.get<SyncManifest>(
      `/api/v1/openclaw/sync/manifest?storage_type=${encodeURIComponent(storageType)}`,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function reportSyncResult(
  client: ApiClient,
  params: SyncResultRequest,
): Promise<CallToolResult> {
  try {
    const data = await client.post<Record<string, unknown>>(
      "/api/v1/openclaw/sync/result",
      params as unknown as Record<string, unknown>,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function configureSync(
  client: ApiClient,
  params: SyncConfig,
): Promise<CallToolResult> {
  try {
    const data = await client.put<Record<string, unknown>>(
      "/api/v1/openclaw/sync/config",
      params as unknown as Record<string, unknown>,
    );
    return {
      content: [{ type: "text", text: JSON.stringify(data, null, 2) }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}
