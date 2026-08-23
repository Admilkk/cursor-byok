import { createContext, useContext, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useKeepAliveContext } from "keepalive-for-react";

export const PageActionsTarget = createContext<HTMLElement | null>(null);

export function PageActions({ children }: { children: ReactNode }) {
  const target = useContext(PageActionsTarget);
  const { active } = useKeepAliveContext();
  return active && target ? createPortal(children, target) : null;
}
