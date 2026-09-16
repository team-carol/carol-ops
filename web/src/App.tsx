import { useEffect, useState } from "react";
import { Header, HeaderGlobalAction, HeaderGlobalBar, HeaderName, Loading } from "@carbon/react";
import { Logout } from "@carbon/icons-react";
import { api, UnauthorizedError } from "./api/client";
import LoginPage from "./pages/LoginPage";
import DashboardPage from "./pages/DashboardPage";

type AuthState = "checking" | "loggedOut" | "loggedIn";

export default function App() {
  const [auth, setAuth] = useState<AuthState>("checking");

  useEffect(() => {
    // No dedicated "am I logged in" endpoint yet — any protected GET works
    // as a session probe since the auth middleware 401s without a cookie.
    api
      .getStatus()
      .then(() => setAuth("loggedIn"))
      .catch((err) => setAuth(err instanceof UnauthorizedError ? "loggedOut" : "loggedIn"));
  }, []);

  const logout = async () => {
    await api.logout();
    setAuth("loggedOut");
  };

  if (auth === "checking") return <Loading withOverlay />;
  if (auth === "loggedOut") return <LoginPage onLoggedIn={() => setAuth("loggedIn")} />;

  return (
    <>
      <Header aria-label="carol-ops">
        <HeaderName href="#" prefix="">
          carol-ops
        </HeaderName>
        <HeaderGlobalBar>
          <HeaderGlobalAction aria-label="로그아웃" onClick={logout}>
            <Logout size={20} />
          </HeaderGlobalAction>
        </HeaderGlobalBar>
      </Header>
      <DashboardPage />
    </>
  );
}
