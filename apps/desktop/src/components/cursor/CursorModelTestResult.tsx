import type { ModelConnectivityResult } from "../../api";
import { Icon } from "../ui/Icon";
import { TooltipTrigger } from "../ui/TooltipTrigger";
import { informationOutlineIcon } from "../ui/icons";
import styles from "./CursorSettings.module.scss";

export type CursorModelTestState =
  | { status: "success"; result: ModelConnectivityResult }
  | { status: "error"; error: string };

export function CursorModelTestResult({ state }: { state: CursorModelTestState }) {
  const success = state.status === "success";
  const summary = success
    ? t("速度：{speed} tokens/s", { speed: formatSpeed(state.result.tokens_per_second) })
    : t("错误：{error}", { error: state.error });
  const detail = success
    ? t("速度 {speed} tokens/s · 首字 {firstText} ms · 总耗时 {duration} ms · 输出 {tokens} tokens{estimated} · 返回：{output}", {
      speed: formatSpeed(state.result.tokens_per_second),
      firstText: state.result.first_text_ms ?? "--",
      duration: state.result.duration_ms,
      tokens: state.result.output_tokens,
      estimated: state.result.tokens_estimated ? t("（估算）") : "",
      output: state.result.output || "--",
    })
    : t("测试失败：{error}", { error: state.error });

  return <div className={`${styles.testResult} ${success ? styles.testSuccess : styles.testError}`}>
    <span className={styles.testResultText}>{summary}</span>
    <TooltipTrigger label={detail}><span className={styles.testResultHint} tabIndex={0}><Icon icon={informationOutlineIcon} size="1.1em" /></span></TooltipTrigger>
  </div>;
}

function formatSpeed(value: number) {
  return Number.isFinite(value) ? value.toFixed(1) : "0.0";
}
