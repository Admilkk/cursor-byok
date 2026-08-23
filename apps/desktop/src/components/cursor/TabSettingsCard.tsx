import cursorIconUrl from "../../assets/icons/cursor.svg";
import type { TabMode, TabSettings } from "../../api";
import { Button } from "../ui/Button";
import { TextInput } from "../ui/FormControls";
import { Icon } from "../ui/Icon";
import { Select } from "../ui/Select";
import { TitledCard } from "../ui/TitledCard";
import styles from "./TabSettingsCard.module.scss";

export function TabSettingsCard({ settings, saving, onChange, onSave }: {
  settings: TabSettings;
  saving: boolean;
  onChange: (settings: TabSettings) => void;
  onSave: () => void;
}) {
  return <TitledCard
    title={<div className={styles.title}><Icon src={cursorIconUrl} size="1.1em" /><span>{t("TAB 设置")}</span></div>}
    action={<Button size="small" variant="primary" disabled={saving} onClick={onSave}>{saving ? t("保存中…") : t("保存")}</Button>}
  >
    <div className={styles.row}>
      <div className={styles.description}>
        <strong>{t("TAB 选择")}</strong>
        <small>{t("控制 Cursor TAB 相关接口的连接方式。")}</small>
      </div>
      <div className={styles.selectControl}>
        <Select
          value={settings.mode}
          ariaLabel={t("TAB 选择")}
          options={[
            { value: "public", label: t("使用公益服务") },
            { value: "direct", label: t("直连") },
            { value: "custom", label: t("自定义") },
          ]}
          onChange={(mode) => onChange({ ...settings, mode: mode as TabMode })}
        />
      </div>
    </div>
    {settings.mode === "custom" && <div className={styles.row}>
      <div className={styles.description}>
        <strong>{t("TAB 服务地址")}</strong>
        <small>{t("原接口路径会追加到此服务地址。")}</small>
      </div>
      <TextInput
        className={styles.addressInput}
        value={settings.address}
        placeholder="https://tab.leokun.cn"
        aria-label={t("TAB 服务地址")}
        onChange={(event) => onChange({ ...settings, address: event.target.value })}
        onKeyDown={(event) => { if (event.key === "Enter") onSave(); }}
      />
    </div>}
  </TitledCard>;
}
