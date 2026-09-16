import { Header, HeaderName } from "@carbon/react";
import DashboardPage from "./pages/DashboardPage";

// No login here — this app is only reachable through a Cloudflare
// Access-protected tunnel hostname (no port is published to the host), so
// Access's per-email policy is the sole auth boundary. See
// internal/api/router.go for the backend side of that decision.
export default function App() {
  return (
    <>
      <Header aria-label="carol-ops">
        <HeaderName href="#" prefix="">
          carol-ops
        </HeaderName>
      </Header>
      <DashboardPage />
    </>
  );
}
