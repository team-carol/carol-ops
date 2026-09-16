// Thin fetch wrapper for the Go backend (internal/api/router.go). No login
// here — Cloudflare Access in front of the tunnel is the only auth boundary.
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!res.ok) throw new Error(`${init?.method ?? "GET"} ${path}: ${res.status} ${await res.text()}`);
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export interface Entry {
  key: string;
  value: string;
}

export interface Container {
  Id: string;
  Names: string[];
  State: string;
  Labels: Record<string, string>;
}

export interface BotStatus {
  guildCount: number;
  gatewayPingMs: number;
  lastSyncAt: string;
  uptimeSeconds: number;
  userCount: number;
}

export interface HostInfo {
  loadAvg1: number;
  memTotalBytes: number;
  memUsedBytes: number;
  diskTotalBytes: number;
  diskUsedBytes: number;
}

// Loose shape — this is `docker inspect` passed through verbatim (see
// dockerctl.Client.Inspect), only the few fields the detail view reads are
// named here.
export interface ContainerDetail {
  Name: string;
  Created: string;
  RestartCount: number;
  Config?: { Image?: string };
  State?: { Status?: string; Health?: { Status?: string } };
}

export const api = {
  getConfig: () => request<Entry[]>("/api/config"),
  putConfig: (entries: Entry[]) => request<void>("/api/config", { method: "PUT", body: JSON.stringify(entries) }),

  getEnv: () => request<Entry[]>("/api/env"),
  putEnv: (entries: Entry[]) => request<void>("/api/env", { method: "PUT", body: JSON.stringify(entries) }),

  getStatus: () => request<BotStatus>("/api/status"),
  getHost: () => request<HostInfo>("/api/host"),
  getContainers: () => request<Container[]>("/api/containers"),
  containerAction: (id: string, action: "start" | "stop" | "restart") =>
    request<void>(`/api/containers/${id}/${action}`, { method: "POST" }),
  getContainerDetail: (id: string) => request<ContainerDetail>(`/api/containers/${id}`),
  getContainerLogs: async (id: string, tail = 200): Promise<string> => {
    const res = await fetch(`/api/containers/${id}/logs?tail=${tail}`, { credentials: "include" });
    if (!res.ok) throw new Error(`GET .../logs: ${res.status} ${await res.text()}`);
    return res.text();
  },
};
