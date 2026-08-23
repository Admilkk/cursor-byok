import { cloneElement, useState, type FocusEventHandler, type MouseEventHandler, type ReactElement } from "react";
import { Tooltip, type TooltipAnchor } from "./Tooltip";

function anchorFor(element: HTMLElement): TooltipAnchor {
  return { contextElement: element, getBoundingClientRect: () => element.getBoundingClientRect() };
}

type TriggerProps = {
  onMouseEnter?: MouseEventHandler<HTMLElement>;
  onMouseLeave?: MouseEventHandler<HTMLElement>;
  onFocus?: FocusEventHandler<HTMLElement>;
  onBlur?: FocusEventHandler<HTMLElement>;
};

export function TooltipTrigger({ label, children }: { label: string; children: ReactElement<TriggerProps> }) {
  const [anchor, setAnchor] = useState<TooltipAnchor | null>(null);
  const trigger = cloneElement(children, {
    onMouseEnter: (event) => {
      children.props.onMouseEnter?.(event);
      setAnchor(anchorFor(event.currentTarget));
    },
    onMouseLeave: (event) => {
      children.props.onMouseLeave?.(event);
      setAnchor(null);
    },
    onFocus: (event) => {
      children.props.onFocus?.(event);
      setAnchor(anchorFor(event.currentTarget));
    },
    onBlur: (event) => {
      children.props.onBlur?.(event);
      setAnchor(null);
    },
  });

  return <>
    {trigger}
    <Tooltip anchor={anchor}>{label}</Tooltip>
  </>;
}
