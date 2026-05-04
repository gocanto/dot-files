const statusPillClasses: Record<string, string> = {
  pending: "border-slate-400/30 bg-slate-400/10 text-slate-600 dark:text-slate-300",
  running: "border-sky-500/30 bg-sky-500/10 text-sky-600 dark:text-sky-300",
  completed: "border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300",
  ok: "border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300",
  failed: "border-rose-500/30 bg-rose-500/10 text-rose-600 dark:text-rose-300",
  stopped: "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-300",
  skipped: "border-zinc-400/30 bg-zinc-400/10 text-zinc-600 dark:text-zinc-300",
};

export function phaseStatusPillClass(status: string): string {
  return statusPillClasses[status] ?? "border-muted bg-muted/30 text-muted-foreground";
}

export const statusPillClass = phaseStatusPillClass;
