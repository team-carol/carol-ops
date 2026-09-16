import { useEffect, useState } from "react";
import {
  Button,
  Column,
  Grid,
  Loading,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Tag,
  Tile,
} from "@carbon/react";
import { api, type BotStatus, type Container } from "../api/client";
import KeyValueEditor from "../components/KeyValueEditor";

export default function DashboardPage() {
  return (
    <Grid style={{ marginTop: "2rem" }}>
      <Column lg={10} md={8} sm={4}>
        <Tabs>
          <TabList aria-label="carol-ops sections">
            <Tab>상태</Tab>
            <Tab>컨테이너</Tab>
            <Tab>config.json</Tab>
            <Tab>.env</Tab>
          </TabList>
          <TabPanels>
            <TabPanel>
              <StatusPanel />
            </TabPanel>
            <TabPanel>
              <ContainersPanel />
            </TabPanel>
            <TabPanel>
              <ConfigPanel />
            </TabPanel>
            <TabPanel>
              <EnvPanel />
            </TabPanel>
          </TabPanels>
        </Tabs>
      </Column>
    </Grid>
  );
}

function StatusPanel() {
  const [status, setStatus] = useState<BotStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.getStatus().then(setStatus).catch((err) => setError(String(err)));
  }, []);

  if (error) return <p>불러오기 실패: {error}</p>;
  if (!status) return <Loading small withOverlay={false} />;

  return (
    <Tile>
      <p>길드 수: {status.guildCount}</p>
      <p>게이트웨이 핑: {status.gatewayPingMs}ms</p>
      <p>마지막 동기화: {status.lastSyncAt}</p>
      <p>가동 시간: {Math.floor(status.uptimeSeconds / 3600)}시간</p>
    </Tile>
  );
}

function ContainersPanel() {
  const [containers, setContainers] = useState<Container[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

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

  if (error) return <p>불러오기 실패: {error}</p>;
  if (!containers) return <Loading small withOverlay={false} />;

  return (
    <div>
      {containers.map((c) => (
        <Tile key={c.Id} style={{ marginBottom: "1rem" }}>
          <p>
            <strong>{c.Names[0]?.replace(/^\//, "")}</strong> <Tag type={c.State === "running" ? "green" : "gray"}>{c.State}</Tag>
          </p>
          <div style={{ display: "flex", gap: "0.5rem", marginTop: "0.5rem" }}>
            <Button kind="tertiary" size="sm" disabled={busyId === c.Id} onClick={() => act(c.Id, "restart")}>
              재시작
            </Button>
            <Button kind="danger--tertiary" size="sm" disabled={busyId === c.Id} onClick={() => act(c.Id, "stop")}>
              정지
            </Button>
            <Button kind="tertiary" size="sm" disabled={busyId === c.Id} onClick={() => act(c.Id, "start")}>
              시작
            </Button>
          </div>
        </Tile>
      ))}
    </div>
  );
}

function ConfigPanel() {
  const [entries, setEntries] = useState<Awaited<ReturnType<typeof api.getConfig>> | null>(null);
  useEffect(() => {
    api.getConfig().then(setEntries);
  }, []);
  if (!entries) return <Loading small withOverlay={false} />;
  return <KeyValueEditor entries={entries} onSave={api.putConfig} />;
}

function EnvPanel() {
  const [entries, setEntries] = useState<Awaited<ReturnType<typeof api.getEnv>> | null>(null);
  useEffect(() => {
    api.getEnv().then(setEntries);
  }, []);
  if (!entries) return <Loading small withOverlay={false} />;
  return <KeyValueEditor entries={entries} onSave={api.putEnv} />;
}
