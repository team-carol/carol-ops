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
}

export const api = {
  getConfig: () => request<Entry[]>("/api/config"),
  putConfig: (entries: Entry[]) => request<void>("/api/config", { method: "PUT", body: JSON.stringify(entries) }),

  getEnv: () => request<Entry[]>("/api/env"),
  putEnv: (entries: Entry[]) => request<void>("/api/env", { method: "PUT", body: JSON.stringify(entries) }),

  getStatus: () => request<BotStatus>("/api/status"),
  getContainers: () => request<Container[]>("/api/containers"),
  containerAction: (id: string, action: "start" | "stop" | "restart") =>
    request<void>(`/api/containers/${id}/${action}`, { method: "POST" }),
};
