import { CallTable } from "../components/CallTable";
import { PageContent } from "../components/layout/PageContent";
import { appStore, useAppStore } from "../store/appStore";
import styles from "./CallsPage.module.scss";

export function CallsPage() {
  const { calls } = useAppStore();
  const content = <div className={styles.page}><CallTable calls={calls} onDetails={(call) => void appStore.openCallDetails(call.call_id)} /></div>;
  return <PageContent fixed title={t("调用")} contentClassName={styles.pageContent} sections={[{ key: "calls", estimatedHeight: 720, content }]} />;
}
