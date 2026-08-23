import type { Provider } from "../api";
import controls from "./ui/Controls.module.scss";
import { DataTable, type DataTableColumn } from "./ui/DataTable";
import { Icon } from "./ui/Icon";
import { TooltipTrigger } from "./ui/TooltipTrigger";
import { editIcon, trashIcon } from "./ui/icons";
import styles from "./ProviderTable.module.scss";

const json = (value: unknown) => JSON.stringify(value);

export function ProviderTable({ providers, onEdit, onDelete }: {
  providers: Provider[];
  onEdit: (provider: Provider) => void;
  onDelete: (provider: Provider) => void;
}) {
  const columns: DataTableColumn<Provider>[] = [
    { key: "name", header: t("名称"), render: (provider) => provider.name, title: (provider) => provider.name },
    { key: "type", header: t("协议"), render: (provider) => provider.provider_type },
    { key: "url", header: "Base URL", render: (provider) => provider.base_url, title: (provider) => provider.base_url },
    { key: "key", header: "API Key", render: (provider) => <span className={styles.badge}>{provider.has_api_key ? t("已配置") : t("未配置")}</span> },
    { key: "headers", header: "Headers JSON", render: (provider) => json(provider.custom_headers), title: (provider) => json(provider.custom_headers) },
    { key: "extra", header: t("额外参数 JSON"), render: (provider) => json(provider.extra_params), title: (provider) => json(provider.extra_params) },
    { key: "created", header: t("创建时间"), render: (provider) => new Date(provider.created_at_ms).toLocaleString() },
    { key: "updated", header: t("更新时间"), render: (provider) => new Date(provider.updated_at_ms).toLocaleString() },
    { key: "actions", header: t("操作"), sticky: "right", render: (provider) => <div className={styles.actions}>
      <TooltipTrigger label={t("编辑上游")}><button className={controls.iconButton} aria-label={t("编辑上游")} onClick={() => onEdit(provider)}><Icon icon={editIcon} size="1.1em" /></button></TooltipTrigger>
      <TooltipTrigger label={t("删除上游")}><button className={`${controls.iconButton} ${controls.danger}`} aria-label={t("删除上游")} onClick={() => onDelete(provider)}><Icon icon={trashIcon} size="1.1em" /></button></TooltipTrigger>
    </div> },
  ];
  return <DataTable rows={providers} columns={columns} rowKey={(provider) => provider.provider_id} minWidth={1400} />;
}
