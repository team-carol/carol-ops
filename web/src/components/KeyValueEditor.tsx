import { useEffect, useState } from "react";
import { Button, IconButton, InlineNotification, TextInput, Stack } from "@carbon/react";
import { Add, TrashCan } from "@carbon/icons-react";
import type { Entry } from "../api/client";

const SECRET_KEY_PATTERN = /secret|token|password|key/i;

interface Props {
  entries: Entry[];
  onSave: (entries: Entry[]) => Promise<void>;
}

// A row's React key can't be its (editable, possibly-blank, possibly
// duplicate-while-typing) key text, so each row gets a separate stable id.
interface DraftRow extends Entry {
  id: number;
}

let nextRowId = 0;
function toDraftRows(entries: Entry[]): DraftRow[] {
  return entries.map((e) => ({ ...e, id: nextRowId++ }));
}

// Edits a flat key-value list: add/remove/rename/re-value rows, all as local
// draft state — nothing reaches the server until "저장" is clicked, at which
// point the full draft (including removed-row omissions) is PUT in one call.
// The backend (kvfile.WriteJSON/WriteEnv) treats that PUT body as the
// complete desired set, so an omitted key is what actually deletes it.
export default function KeyValueEditor({ entries, onSave }: Props) {
  const [draft, setDraft] = useState<DraftRow[]>(() => toDraftRows(entries));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => setDraft(toDraftRows(entries)), [entries]);

  const updateKey = (id: number, key: string) =>
    setDraft((prev) => prev.map((r) => (r.id === id ? { ...r, key } : r)));
  const updateValue = (id: number, value: string) =>
    setDraft((prev) => prev.map((r) => (r.id === id ? { ...r, value } : r)));
  const remove = (id: number) => setDraft((prev) => prev.filter((r) => r.id !== id));
  const add = () => setDraft((prev) => [...prev, { id: nextRowId++, key: "", value: "" }]);

  const save = async () => {
    setSaving(true);
    setError(null);
    setSaved(false);
    // Blank-key rows (e.g. an "항목 추가" click nobody filled in) are dropped
    // rather than sent — the backend would otherwise write a "" key.
    const deduped = new Map<string, string>();
    for (const row of draft) {
      const key = row.key.trim();
      if (key) deduped.set(key, row.value);
    }
    try {
      await onSave(Array.from(deduped, ([key, value]) => ({ key, value })));
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Stack gap={5}>
      {error && <InlineNotification kind="error" title="저장 실패" subtitle={error} onCloseButtonClick={() => setError(null)} />}
      {saved && <InlineNotification kind="success" title="저장됨" onCloseButtonClick={() => setSaved(false)} />}
      <Stack gap={4}>
        {draft.map((row) => (
          <div key={row.id} style={{ display: "flex", gap: "0.5rem", alignItems: "flex-end" }}>
            <TextInput
              id={`kv-key-${row.id}`}
              labelText="키"
              value={row.key}
              onChange={(e) => updateKey(row.id, e.target.value)}
              style={{ flex: 1 }}
            />
            <TextInput
              id={`kv-value-${row.id}`}
              labelText="값"
              type={SECRET_KEY_PATTERN.test(row.key) ? "password" : "text"}
              value={row.value}
              onChange={(e) => updateValue(row.id, e.target.value)}
              style={{ flex: 2 }}
            />
            <IconButton label="삭제" kind="ghost" onClick={() => remove(row.id)}>
              <TrashCan />
            </IconButton>
          </div>
        ))}
      </Stack>
      <div style={{ display: "flex", gap: "0.5rem" }}>
        <Button kind="tertiary" renderIcon={Add} onClick={add}>
          항목 추가
        </Button>
        <Button onClick={save} disabled={saving}>
          {saving ? "저장 중..." : "저장"}
        </Button>
      </div>
    </Stack>
  );
}
