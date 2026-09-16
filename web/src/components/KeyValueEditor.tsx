import { useEffect, useState } from "react";
import { Button, InlineNotification, TextInput, Stack } from "@carbon/react";
import type { Entry } from "../api/client";

const SECRET_KEY_PATTERN = /secret|token|password|key/i;

interface Props {
  entries: Entry[];
  onSave: (entries: Entry[]) => Promise<void>;
}

// Edits a flat key-value list in place. Values whose key looks secret
// (token/password/secret/key) render masked, matching how config.json's
// own fields (carolStatusSecret, adminPassword, ...) should be handled.
export default function KeyValueEditor({ entries, onSave }: Props) {
  const [draft, setDraft] = useState(entries);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => setDraft(entries), [entries]);

  const update = (key: string, value: string) =>
    setDraft((prev) => prev.map((e) => (e.key === key ? { ...e, value } : e)));

  const save = async () => {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      await onSave(draft);
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
        {draft.map((entry) => (
          <TextInput
            key={entry.key}
            id={`kv-${entry.key}`}
            labelText={entry.key}
            type={SECRET_KEY_PATTERN.test(entry.key) ? "password" : "text"}
            value={entry.value}
            onChange={(e) => update(entry.key, e.target.value)}
          />
        ))}
      </Stack>
      <Button onClick={save} disabled={saving}>
        {saving ? "저장 중..." : "저장"}
      </Button>
    </Stack>
  );
}
