import DashboardPage from "./pages/DashboardPage";

// No login here — this app is only reachable through a Cloudflare
// Access-protected tunnel hostname (no port is published to the host), so
// Access's per-email policy is the sole auth boundary. See
// internal/api/router.go for the backend side of that decision.
//
// The header is a normal in-flow element (not `position: fixed`) on purpose
// — a fixed header needs every page under it to reserve matching top
// padding or it silently overlaps content, which is exactly what happened
// with Carbon's <Header>.
export default function App() {
  return (
    <>
      <header className="app-header">
        <div className="app-header-inner">
          <div className="brand">
            <img src="/carolbot-avatar.png" alt="" />
            <span className="wordmark">캐롤봇 운영</span>
          </div>
          <span className="header-label">team carol</span>
        </div>
      </header>
      <main>
        <DashboardPage />
      </main>
    </>
  );
}
