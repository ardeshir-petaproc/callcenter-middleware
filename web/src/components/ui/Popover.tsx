import { useEffect, useLayoutEffect, useRef, useState, type ReactNode, type RefObject } from 'react';
import { createPortal } from 'react-dom';

interface Props {
  anchor: RefObject<HTMLElement>;
  open: boolean;
  onClose: () => void;
  children: ReactNode;
  width?: number;
}

/** Menu rendered in a portal with fixed positioning so it is never clipped by scroll containers. */
export function Popover({ anchor, open, onClose, children, width = 230 }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);

  useLayoutEffect(() => {
    if (!open || !anchor.current) return;
    const r = anchor.current.getBoundingClientRect();
    const left = Math.max(8, Math.min(window.innerWidth - width - 8, r.right - width));
    const below = r.bottom + 6;
    const top = below + 220 > window.innerHeight ? Math.max(8, r.top - 6 - 220) : below;
    setPos({ top, left });
  }, [open, anchor, width]);

  useEffect(() => {
    if (!open) return;
    const close = (e: MouseEvent) => {
      const t = e.target as Node;
      if (!ref.current?.contains(t) && !anchor.current?.contains(t)) onClose();
    };
    const esc = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    const scroll = () => onClose();
    document.addEventListener('mousedown', close);
    document.addEventListener('keydown', esc);
    window.addEventListener('resize', scroll);
    return () => {
      document.removeEventListener('mousedown', close);
      document.removeEventListener('keydown', esc);
      window.removeEventListener('resize', scroll);
    };
  }, [open, onClose, anchor]);

  if (!open || !pos) return null;
  return createPortal(
    <div ref={ref} className="menu" role="menu" style={{ position: 'fixed', top: pos.top, left: pos.left, right: 'auto', width, zIndex: 120 }}>
      {children}
    </div>,
    document.body,
  );
}
