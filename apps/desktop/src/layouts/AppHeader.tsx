import appIcon from "../../src-tauri/icons/32x32.png";
import type { DesktopPlatform } from "./AppFrame";
import { WindowControls } from "./WindowControls";
import styles from "./AppHeader.module.scss";

type AppHeaderProps = {
  platform: DesktopPlatform;
  nativeDesktop: boolean;
};

export function AppHeader({ platform, nativeDesktop }: AppHeaderProps) {
  const showNativeUi = nativeDesktop && platform !== "macos";

  return <header className={styles.root}>
    <div className={styles.dragLayer} data-tauri-drag-region aria-hidden="true" />
    <div className={styles.uiLayer}>
      {showNativeUi && <>
        <div className={styles.identity} aria-label="Cursor BYOK">
          <img src={appIcon} alt="" />
          <span>Cursor 助手 v0.1.0</span>
        </div>
        <WindowControls />
      </>}
    </div>
  </header>;
}
