import { useEffect, useState } from "react";
import {
  currentAppVersion,
  hasNativeAppLifecycle,
  readAutostart,
  writeAutostart,
} from "../../native/appLifecycle";
import { updateStore, useUpdateStore } from "../../store/updateStore";
import { Button } from "../ui/Button";
import { Switch } from "../ui/Switch";
import { TitledCard } from "../ui/TitledCard";
import { useMessage } from "../ui/message";
import styles from "./AppLifecycleSettingsCard.module.scss";

export function AppLifecycleSettingsCard() {
  const message = useMessage();
  const native = hasNativeAppLifecycle();
  const { availableVersion, checking, installing } = useUpdateStore();
  const [version, setVersion] = useState("…");
  const [autostart, setAutostart] = useState(false);
  const [loadingAutostart, setLoadingAutostart] = useState(native);

  useEffect(() => {
    let disposed = false;
    void currentAppVersion().then((next) => { if (!disposed) setVersion(next); });
    if (native) {
      void readAutostart()
        .then((enabled) => { if (!disposed) setAutostart(enabled); })
        .catch((cause) => message(cause instanceof Error ? cause.message : String(cause)))
        .finally(() => { if (!disposed) setLoadingAutostart(false); });
    }
    return () => { disposed = true; };
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
      const nextVersion = await updateStore.check();
      message(nextVersion ? t("发现新版本 {version}", { version: nextVersion }) : t("当前已是最新版本"));
    } catch (cause) {
      message(cause instanceof Error ? cause.message : String(cause));
    }
  };

  const updateNow = async () => {
    try {
      await updateStore.install();
    } catch (cause) {
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
            <span className={styles.updateDot} aria-hidden="true" />
          </Button>
        : <Button size="small" disabled={!native || checking} onClick={() => void checkUpdate()}>
            {checking ? t("检查中…") : t("检查更新")}
          </Button>}
    </div>
  </TitledCard>;
}
