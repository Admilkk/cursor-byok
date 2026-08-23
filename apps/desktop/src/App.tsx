import { useEffect, useRef } from "react";
import { HashRouter, Navigate, Route, Routes } from "react-router-dom";
import { MessageProvider } from "./components/ui/MessageProvider";
import { useMessage } from "./components/ui/message";
import { AppFrame } from "./layouts/AppFrame";
import { AppLayout } from "./layouts/AppLayout";
import { CallsPage } from "./pages/CallsPage";
import { CallDetailsPage } from "./pages/CallDetailsPage";
import { CursorSettingsPage } from "./pages/CursorSettingsPage";
import { HomePage } from "./pages/HomePage";
import { ProvidersPage } from "./pages/ProvidersPage";
import { SettingsPage } from "./pages/SettingsPage";
import { checkForUpdate, hasNativeAppLifecycle } from "./native/appLifecycle";
import { useAppStore } from "./store/appStore";

export function App() {
  return (
    <>
      <HashRouter>
        <Routes>
          <Route path="calls/:callId" element={<CallDetailsPage />} />
          <Route element={<AppFrame />}>
            <Route element={<AppLayout />}>
              <Route index element={<HomePage />} />
              <Route path="providers" element={<ProvidersPage />} />
              <Route path="calls" element={<CallsPage />} />
              <Route path="harness/cursor" element={<CursorSettingsPage />} />
              <Route path="settings" element={<SettingsPage />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </HashRouter>
      <AppMessages />
    </>
  );
}

function AppMessages() {
  const { error } = useAppStore();
  const previousError = useRef<string | null>(null);
  const showMessage = useMessage();

  useEffect(() => {
    if (error && error !== previousError.current) showMessage(error);
    previousError.current = error;
  }, [error, showMessage]);

  useEffect(() => {
    if (!hasNativeAppLifecycle()) return;
    void checkForUpdate().then(async (update) => {
      if (!update) return;
      showMessage(t("发现新版本 {version}，可在设置中安装", { version: update.version }), { duration: 6_000 });
      await update.close();
    }).catch(() => {
      // Startup checks are best-effort; manual checks in Settings report errors.
    });
  }, [showMessage]);

  return <MessageProvider />;
}
