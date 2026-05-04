interface ConfirmationStyle {
  buttonClass: string;
  iconClass: string;
}

const styles: Record<string, ConfirmationStyle> = {
  "preview-only": {
    buttonClass: "border-l-4 border-l-sky-500/70 hover:bg-sky-500/5",
    iconClass: "text-sky-500 dark:text-sky-300",
  },
  "run-now": {
    buttonClass: "border-l-4 border-l-emerald-500/70 hover:bg-emerald-500/5",
    iconClass: "text-emerald-500 dark:text-emerald-300",
  },
  "already-erased-run-now": {
    buttonClass: "border-l-4 border-l-emerald-500/70 hover:bg-emerald-500/5",
    iconClass: "text-emerald-500 dark:text-emerald-300",
  },
  "run-without-erasing": {
    buttonClass: "border-l-4 border-l-amber-500/70 hover:bg-amber-500/5",
    iconClass: "text-amber-500 dark:text-amber-300",
  },
  "erase-first": {
    buttonClass: "border-l-4 border-l-rose-500/70 hover:bg-rose-500/5",
    iconClass: "text-rose-500 dark:text-rose-300",
  },
  back: {
    buttonClass: "border-l-4 border-l-slate-400/50 hover:bg-slate-400/5",
    iconClass: "text-slate-500 dark:text-slate-300",
  },
};

const fallback: ConfirmationStyle = {
  buttonClass: "border-l-4 border-l-muted hover:bg-muted/30",
  iconClass: "text-muted-foreground",
};

export function confirmationStyle(id: string): ConfirmationStyle {
  return styles[id] ?? fallback;
}
