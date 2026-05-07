import {
  AlertTriangle,
  CheckCircle2,
  Circle,
  CircleDashed,
  CircleSlash,
  Loader2,
  MinusCircle,
} from "lucide-vue-next";

const statusPillClasses: Record<string, string> = {
  idle: "border-[var(--status-idle-border)] bg-[var(--status-idle-bg)] text-[var(--status-idle-fg)]",
  pending:
    "border-[var(--status-neutral-border)] bg-[var(--status-neutral-bg)] text-[var(--status-neutral-fg)]",
  running:
    "border-[var(--status-running-border)] bg-[var(--status-running-bg)] text-[var(--status-running-fg)]",
  completed:
    "border-[var(--status-success-border)] bg-[var(--status-success-bg)] text-[var(--status-success-fg)]",
  ok: "border-[var(--status-success-border)] bg-[var(--status-success-bg)] text-[var(--status-success-fg)]",
  failed:
    "border-[var(--status-danger-border)] bg-[var(--status-danger-bg)] text-[var(--status-danger-fg)]",
  stopped:
    "border-[var(--status-attention-border)] bg-[var(--status-attention-bg)] text-[var(--status-attention-fg)]",
  skipped:
    "border-[var(--status-neutral-border)] bg-[var(--status-neutral-bg)] text-[var(--status-neutral-fg)]",
};

export function phaseStatusPillClass(status: string): string {
  return statusPillClasses[status] ?? "border-muted bg-muted/30 text-muted-foreground";
}

export const statusPillClass = phaseStatusPillClass;

const statusIcons: Record<string, typeof Circle> = {
  idle: CircleDashed,
  pending: Circle,
  running: Loader2,
  completed: CheckCircle2,
  ok: CheckCircle2,
  failed: AlertTriangle,
  stopped: CircleSlash,
  skipped: MinusCircle,
};

export function statusIconFor(status: string): typeof Circle {
  return statusIcons[status] ?? Circle;
}
