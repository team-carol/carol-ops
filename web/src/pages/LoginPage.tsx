import { useState } from "react";
import { Button, Form, InlineNotification, PasswordInput, Stack, TextInput } from "@carbon/react";
import { api } from "../api/client";

export default function LoginPage({ onLoggedIn }: { onLoggedIn: () => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      await api.login(username, password);
      onLoggedIn();
    } catch {
      setError("아이디 또는 비밀번호가 올바르지 않습니다.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div style={{ maxWidth: 320, margin: "15vh auto" }}>
      <h2 style={{ marginBottom: "1.5rem" }}>carol-ops</h2>
      <Form onSubmit={submit}>
        <Stack gap={5}>
          {error && <InlineNotification kind="error" title="로그인 실패" subtitle={error} onCloseButtonClick={() => setError(null)} />}
          <TextInput id="username" labelText="아이디" value={username} onChange={(e) => setUsername(e.target.value)} />
          <PasswordInput id="password" labelText="비밀번호" value={password} onChange={(e) => setPassword(e.target.value)} />
          <Button type="submit" disabled={submitting}>
            {submitting ? "로그인 중..." : "로그인"}
          </Button>
        </Stack>
      </Form>
    </div>
  );
}
