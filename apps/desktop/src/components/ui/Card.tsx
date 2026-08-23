import type { ElementType, ReactNode } from "react";
import styles from "./Card.module.scss";

type CardProps = {
  as?: ElementType;
  className?: string;
  children: ReactNode;
};

export function Card({ as: Component = "div", className, children }: CardProps) {
  return <Component className={[styles.root, className].filter(Boolean).join(" ")}>{children}</Component>;
}
