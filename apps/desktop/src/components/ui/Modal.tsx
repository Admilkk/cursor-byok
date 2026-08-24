import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { ScrollableContent } from "../virtual/ScrollableContent";
import controls from "./Controls.module.scss";
import styles from "./Modal.module.scss";

export function Modal({ id, open, title, children, busy, wide, onClose, onSubmit, secondaryAction, closeLabel = t("取消"), submitLabel = t("保存") }: { id?: string; open: boolean; title: string; children: ReactNode; busy?: boolean; wide?: boolean; onClose: () => void; onSubmit?: () => void; secondaryAction?: ReactNode; closeLabel?: string; submitLabel?: string }) {
  const dialog = useRef<HTMLDivElement>(null);
  const closeRef = useRef(onClose);
  const busyRef = useRef(Boolean(busy));
  closeRef.current = onClose;
  busyRef.current = Boolean(busy);
  useEffect(() => {
    if (!open) return;
    const previous = document.activeElement as HTMLElement | null;
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape" && !busyRef.current) closeRef.current(); };
    document.addEventListener("keydown", onKey);
    requestAnimationFrame(() => dialog.current?.querySelector<HTMLElement>("input,button")?.focus());
    return () => {
      document.removeEventListener("keydown", onKey);
      if (previous && document.contains(previous)) previous.focus();
    };
  }, [open]);
  if (!open) return null;
  return createPortal(<div className={styles.mask}>
    <div className={styles.dragLayer} data-tauri-drag-region aria-hidden="true" />
    <div id={id} ref={dialog} className={[styles.dialog, wide && styles.wide].filter(Boolean).join(" ")} role="dialog" aria-modal="true" aria-label={title}>
      <header>{title}</header>
      <ScrollableContent alwaysShowVertical className={styles.body} viewportClassName={styles.bodyViewport} contentClassName={styles.bodyContent}>{children}</ScrollableContent>
      <footer>
        <button type="button" className={controls.primary} disabled={busy} onClick={onClose}>{closeLabel}</button>
        {secondaryAction}
        {onSubmit && <button type="button" className={controls.primary} disabled={busy} onClick={onSubmit}>{busy ? t("处理中…") : submitLabel}</button>}
      </footer>
    </div>
  </div>, document.body);
}
