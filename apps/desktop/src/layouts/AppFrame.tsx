import { Outlet } from "react-router-dom";
import styles from "./AppFrame.module.scss";
import { AppHeader } from "./AppHeader";

const currentPlatform = desktopPlatform();
document.documentElement.dataset.platform = currentPlatform;

export function AppFrame() {
  const platform = currentPlatform;
  const platformClasses = platform === "windows" ? [styles.windows] : [];
  const nativeDesktop = "__TAURI_INTERNALS__" in window;

  return (
    <div className={[styles.shell, ...platformClasses].filter(Boolean).join(" ")}>
      <AppHeader platform={platform} nativeDesktop={nativeDesktop} />
      <Outlet />
    </div>
  );
}

export type DesktopPlatform = "macos" | "windows" | "linux";

function desktopPlatform(): DesktopPlatform {
  // return "windows";
  const agent = navigator.userAgent;
  if (/Macintosh|Mac OS X/.test(agent)) return "macos";
  if (/Windows/.test(agent)) return "windows";
  return "linux";
}
