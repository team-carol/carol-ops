import { useEffect, useState } from "react";
import { api, type BotStatus, type Container, type ContainerDetail, type HostInfo } from "../api/client";
import KeyValueEditor from "../components/KeyValueEditor";

// carol-bot's container_name (see carol's docker-compose.yml) — stable and
// documented, so the dedicated log tab can query it directly without an
// extra round trip through the container list to find its id.
const BOT_CONTAINER_NAME = "carol-bot";

const TABS = ["상태", "컨테이너", "carol-bot 로그", "config.json", ".env"] as const;
type Tab = (typeof TABS)[number];

export default function DashboardPage() {
  const [tab, setTab] = useState<Tab>("상태");
  return (
    <div>
      <div className="page-heading">
        <h1>운영 대시보드</h1>
        <p>캐롤봇과 서버 상태를 확인하고 설정을 관리합니다.</p>
      </div>
      <div className="tabs">
        {TABS.map((t) => (
          <button key={t} className={`tab-btn ${tab === t ? "active" : ""}`} aria-pressed={tab === t} onClick={() => setTab(t)}>
            {t}
          </button>
        ))}
      </div>
      {tab === "상태" && <StatusPanel />}
      {tab === "컨테이너" && <ContainersPanel />}
      {tab === "carol-bot 로그" && <LogViewer id={BOT_CONTAINER_NAME} />}
      {tab === "config.json" && <ConfigPanel />}
      {tab === ".env" && <EnvPanel />}
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (!bytes) return "0 GB";
  const gb = bytes / 1024 ** 3;
  return `${gb.toFixed(1)} GB`;
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
    </div>
  );
}

function UsageBar({ label, used, total }: { label: string; used: number; total: number }) {
  const pct = total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0;
  return (
    <div style={{ marginTop: "1rem" }}>
      <div className="stat-label">
        {label} ({formatBytes(used)} / {formatBytes(total)}, {pct}%)
      </div>
      <div className="progress-bar">
        <div className="progress-bar-fill" style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

function StatusPanel() {
  const [status, setStatus] = useState<BotStatus | null>(null);
  const [host, setHost] = useState<HostInfo | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.getStatus().then(setStatus).catch((err) => setError(String(err)));
    api.getHost().then(setHost).catch(() => {});
  }, []);

  if (error) return <div className="notification notification-error">불러오기 실패: {error}</div>;
  if (!status) return <p className="spinner">불러오는 중...</p>;

  return (
    <>
      <div className="card">
        <div className="stat-grid">
          <Stat label="carol-bot 버전" value={status.version || "—"} />
          <Stat label="길드 수" value={String(status.guildCount)} />
          <Stat label="등록 유저 수" value={String(status.userCount)} />
          <Stat label="게이트웨이 핑" value={`${status.gatewayPingMs}ms`} />
          <Stat label="가동 시간" value={`${Math.floor(status.uptimeSeconds / 3600)}시간`} />
          <Stat label="마지막 동기화" value={status.lastSyncAt || "—"} />
        </div>
      </div>
      {host && (
        <div className="card" style={{ marginTop: "1rem" }}>
          <Stat label="서버 로드 (1분 평균)" value={host.loadAvg1.toFixed(2)} />
          <UsageBar label="메모리" used={host.memUsedBytes} total={host.memTotalBytes} />
          <UsageBar label="디스크" used={host.diskUsedBytes} total={host.diskTotalBytes} />
        </div>
      )}
    </>
  );
}

function ContainersPanel() {
  const [containers, setContainers] = useState<Container[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [confirmTarget, setConfirmTarget] = useState<Container | null>(null);
  const [detailTarget, setDetailTarget] = useState<Container | null>(null);

  const load = () => api.getContainers().then(setContainers).catch((err) => setError(String(err)));
  useEffect(() => {
    load();
  }, []);

  const act = async (id: string, action: "start" | "stop" | "restart") => {
    setBusyId(id);
    setError(null);
    try {
      await api.containerAction(id, action);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyId(null);
    }
  };

  if (error) return <div className="notification notification-error">불러오기 실패: {error}</div>;
  if (!containers) return <p className="spinner">불러오는 중...</p>;

  return (
    <div>
      {containers.map((c) => {
        const name = c.Names[0]?.replace(/^\//, "") ?? c.Id;
        return (
          <div key={c.Id} className="card" style={{ marginBottom: "1rem" }}>
            <p>
              <strong>{name}</strong> <span className={`tag ${c.State === "running" ? "tag-green" : "tag-gray"}`}>{c.State}</span>
            </p>
            <div style={{ display: "flex", gap: "0.5rem", marginTop: "0.75rem", flexWrap: "wrap" }}>
              <button
                className="btn btn-ghost btn-sm"
                disabled={busyId === c.Id || c.State !== "running"}
                onClick={() => act(c.Id, "restart")}
              >
                재시작
              </button>
              <button
                className="btn btn-danger btn-sm"
                disabled={busyId === c.Id || c.State !== "running"}
                onClick={() => setConfirmTarget(c)}
              >
                정지
              </button>
              <button
                className="btn btn-ghost btn-sm"
                disabled={busyId === c.Id || c.State === "running"}
                onClick={() => act(c.Id, "start")}
              >
                시작
              </button>
              <button className="btn btn-ghost btn-sm" onClick={() => setDetailTarget(c)}>
                상세 / 로그
              </button>
            </div>
          </div>
        );
      })}

      {confirmTarget && (
        <ConfirmModal
          title="컨테이너 정지"
          message={`"${confirmTarget.Names[0]?.replace(/^\//, "")}"을(를) 정말 정지할까요?`}
          onCancel={() => setConfirmTarget(null)}
          onConfirm={() => {
            const id = confirmTarget.Id;
            setConfirmTarget(null);
            act(id, "stop");
          }}
        />
      )}

      {detailTarget && (
        <ContainerDetailModal
          id={detailTarget.Id}
          name={detailTarget.Names[0]?.replace(/^\//, "") ?? detailTarget.Id}
          onClose={() => setDetailTarget(null)}
        />
      )}
    </div>
  );
}

function ConfirmModal({
  title,
  message,
  onCancel,
  onConfirm,
}: {
  title: string;
  message: string;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <div className="modal-backdrop" onClick={onCancel}>
      <div className="modal" style={{ maxWidth: 420 }} onClick={(e) => e.stopPropagation()}>
        <h3>{title}</h3>
        <p style={{ marginBottom: "1.5rem" }}>{message}</p>
        <div style={{ display: "flex", gap: "0.5rem", justifyContent: "flex-end" }}>
          <button className="btn btn-ghost btn-sm" onClick={onCancel}>
            취소
          </button>
          <button className="btn btn-danger btn-sm" onClick={onConfirm}>
            정지
          </button>
        </div>
      </div>
    </div>
  );
}

function ContainerDetailModal({ id, name, onClose }: { id: string; name: string; onClose: () => void }) {
  const [detail, setDetail] = useState<ContainerDetail | null>(null);
  const [logs, setLogs] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.getContainerDetail(id).then(setDetail).catch((err) => setError(String(err)));
    api
      .getContainerLogs(id)
      .then(setLogs)
      .catch((err) => setError((prev) => prev ?? String(err)));
  }, [id]);

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h3>{name}</h3>
        {error && <div className="notification notification-error">{error}</div>}
        {detail && (
          <div style={{ fontSize: "13px", marginBottom: "1rem", color: "var(--text-lo)" }}>
            <p>이미지: {detail.Config?.Image}</p>
            <p>생성: {detail.Created}</p>
            <p>재시작 횟수: {detail.RestartCount}</p>
            {detail.State?.Health && <p>헬스체크: {detail.State.Health.Status}</p>}
          </div>
        )}
        <pre>{logs ?? "로그 불러오는 중..."}</pre>
        <div style={{ display: "flex", justifyContent: "flex-end", marginTop: "1rem" }}>
          <button className="btn btn-ghost btn-sm" onClick={onClose}>
            닫기
          </button>
        </div>
      </div>
    </div>
  );
}

function LogViewer({ id }: { id: string }) {
  const [logs, setLogs] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const load = () => {
    setLoading(true);
    setError(null);
    api
      .getContainerLogs(id, 500)
      .then(setLogs)
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false));
  };
  useEffect(load, [id]);

  return (
    <div className="card">
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "0.75rem" }}>
        <strong>{id}</strong>
        <button className="btn btn-ghost btn-sm" onClick={load} disabled={loading}>
          {loading ? "불러오는 중..." : "새로고침"}
        </button>
      </div>
      {error && <div className="notification notification-error">{error}</div>}
      <pre style={{ maxHeight: "70vh" }}>{logs ?? "불러오는 중..."}</pre>
    </div>
  );
}

function ConfigPanel() {
  const [entries, setEntries] = useState<Awaited<ReturnType<typeof api.getConfig>> | null>(null);
  useEffect(() => {
    api.getConfig().then(setEntries);
  }, []);
  if (!entries) return <p className="spinner">불러오는 중...</p>;
  return <KeyValueEditor entries={entries} onSave={api.putConfig} />;
}

function EnvPanel() {
  const [entries, setEntries] = useState<Awaited<ReturnType<typeof api.getEnv>> | null>(null);
  useEffect(() => {
    api.getEnv().then(setEntries);
  }, []);
  if (!entries) return <p className="spinner">불러오는 중...</p>;
  return <KeyValueEditor entries={entries} onSave={api.putEnv} />;
}
