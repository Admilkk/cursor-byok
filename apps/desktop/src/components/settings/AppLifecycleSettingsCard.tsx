import { useEffect, useRef, useState } from "react";
import type { Update } from "@tauri-apps/plugin-updater";
import {
  checkForUpdate,
  currentAppVersion,
  hasNativeAppLifecycle,
  installUpdate,
  readAutostart,
  writeAutostart,
} from "../../native/appLifecycle";
import { Button } from "../ui/Button";
import { Switch } from "../ui/Switch";
import { TitledCard } from "../ui/TitledCard";
import { useMessage } from "../ui/message";
import styles from "./AppLifecycleSettingsCard.module.scss";

export function AppLifecycleSettingsCard() {
  const message = useMessage();
  const native = hasNativeAppLifecycle();
  const updateRef = useRef<Update | null>(null);
  const [version, setVersion] = useState("…");
  const [autostart, setAutostart] = useState(false);
  const [loadingAutostart, setLoadingAutostart] = useState(native);
  const [checking, setChecking] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [availableVersion, setAvailableVersion] = useState<string | null>(null);

  useEffect(() => {
    let disposed = false;
    void currentAppVersion().then((next) => { if (!disposed) setVersion(next); });
    if (native) {
      void readAutostart()
        .then((enabled) => { if (!disposed) setAutostart(enabled); })
        .catch((cause) => message(cause instanceof Error ? cause.message : String(cause)))
        .finally(() => { if (!disposed) setLoadingAutostart(false); });
    }
    return () => {
      disposed = true;
      const update = updateRef.current;
      updateRef.current = null;
      if (update) void update.close();
    };
  }, [message, native]);

  const toggleAutostart = async (enabled: boolean) => {
    try {
      setLoadingAutostart(true);
      await writeAutostart(enabled);
      setAutostart(await readAutostart());
      message(enabled ? t("已开启开机启动") : t("已关闭开机启动"));
    } catch (cause) {
      message(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setLoadingAutostart(false);
    }
  };

  const checkUpdate = async () => {
    try {
      setChecking(true);
      const previous = updateRef.current;
      updateRef.current = null;
      if (previous) await previous.close();
      const update = await checkForUpdate();
      updateRef.current = update;
      setAvailableVersion(update?.version ?? null);
      message(update ? t("发现新版本 {version}", { version: update.version }) : t("当前已是最新版本"));
    } catch (cause) {
      message(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setChecking(false);
    }
  };

  const updateNow = async () => {
    const update = updateRef.current;
    if (!update) return;
    try {
      setInstalling(true);
      await installUpdate(update);
    } catch (cause) {
      setInstalling(false);
      message(cause instanceof Error ? cause.message : String(cause));
    }
  };

  return <TitledCard title={t("应用设置")}>
    <div className={styles.row}>
      <div>
        <strong>{t("开机启动")}</strong>
        <small>{t("登录系统后自动启动 Cursor BYOK。")}</small>
      </div>
      <Switch
        checked={autostart}
        disabled={!native || loadingAutostart}
        label={t("开机启动")}
        onChange={(enabled) => void toggleAutostart(enabled)}
      />
    </div>
    <div className={styles.row}>
      <div>
        <strong>{t("软件更新")}</strong>
        <small>{availableVersion
          ? t("版本 {version} 可以安装", { version: availableVersion })
          : t("当前版本 {version}", { version })}</small>
      </div>
      {availableVersion
        ? <Button size="small" variant="primary" disabled={installing} onClick={() => void updateNow()}>
            {installing ? t("安装中…") : t("下载并安装")}
          </Button>
        : <Button size="small" disabled={!native || checking} onClick={() => void checkUpdate()}>
            {checking ? t("检查中…") : t("检查更新")}
          </Button>}
    </div>
  </TitledCard>;
}
