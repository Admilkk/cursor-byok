import type { InputHTMLAttributes } from "react";
import { Icon } from "./Icon";
import { TooltipTrigger } from "./TooltipTrigger";
import { informationOutlineIcon } from "./icons";
import styles from "./FormControls.module.scss";

export function TextInput(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={[styles.input, props.className].filter(Boolean).join(" ")} />;
}

export function FormField({ label, hint, className, children }: { label: string; hint?: string; className?: string; children: React.ReactNode }) {
  return <label className={[styles.field, className].filter(Boolean).join(" ")}>
    <div className={styles.label}>
      <div>{label}</div>
      {hint && <TooltipTrigger label={hint}><div className={styles.hint}><Icon icon={informationOutlineIcon} size="1.1em" /></div></TooltipTrigger>}
    </div>
    {children}
  </label>;
}
