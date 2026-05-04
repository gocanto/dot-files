interface ConfirmationStyle {
  buttonClass: string;
  iconClass: string;
}

const styles: Record<string, ConfirmationStyle> = {
  "preview-only": {
    buttonClass: "border-l-4 border-l-primary/70 hover:bg-primary/5",
    iconClass: "text-primary",
  },
  "run-now": {
    buttonClass: "border-l-4 border-l-[var(--diff-added)]/70 hover:bg-[var(--diff-added)]/5",
    iconClass: "text-[var(--diff-added)]",
  },
  "already-erased-run-now": {
    buttonClass: "border-l-4 border-l-[var(--diff-added)]/70 hover:bg-[var(--diff-added)]/5",
    iconClass: "text-[var(--diff-added)]",
  },
  "run-without-erasing": {
    buttonClass: "border-l-4 border-l-chart-3/70 hover:bg-chart-3/5",
    iconClass: "text-chart-3",
  },
  "erase-first": {
    buttonClass: "border-l-4 border-l-destructive/70 hover:bg-destructive/5",
    iconClass: "text-destructive",
  },
  back: {
    buttonClass: "border-l-4 border-l-muted-foreground/40 hover:bg-muted/30",
    iconClass: "text-muted-foreground",
  },
};

const fallback: ConfirmationStyle = {
  buttonClass: "border-l-4 border-l-muted hover:bg-muted/30",
  iconClass: "text-muted-foreground",
};

export function confirmationStyle(id: string): ConfirmationStyle {
  return styles[id] ?? fallback;
}
