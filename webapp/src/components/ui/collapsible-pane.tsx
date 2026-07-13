import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface CollapsiblePaneProps {
  id?: string;
  open: boolean;
  children: ReactNode;
  className?: string;
  /** Additional class names applied to the inner overflow container. */
  innerClassName?: string;
}

/**
 * Smoothly animates between collapsed and expanded states using the
 * `grid-template-rows: 0fr <-> 1fr` interpolation technique. Modern browsers
 * (Chrome 117+, Firefox 132+, Safari 17.4+) interpolate this for free; older
 * browsers will simply snap, which is acceptable here.
 *
 * The element stays in the DOM while collapsed so that React state inside it
 * is preserved across toggles, but `aria-hidden` and `inert` keep it out of
 * the accessibility tree and tab order when hidden.
 */
export function CollapsiblePane({
  id,
  open,
  children,
  className,
  innerClassName,
}: CollapsiblePaneProps) {
  return (
    <div
      id={id}
      // `inert` is a boolean attribute; use empty string when set, otherwise omit.
      {...(open ? {} : { inert: "" })}
      aria-hidden={!open}
      className={cn(
        "grid transition-[grid-template-rows,opacity] duration-200 ease-out motion-reduce:transition-none",
        open
          ? "grid-rows-[1fr] opacity-100"
          : "grid-rows-[0fr] opacity-0",
        className,
      )}
    >
      <div className={cn("overflow-hidden min-h-0", innerClassName)}>
        {children}
      </div>
    </div>
  );
}
