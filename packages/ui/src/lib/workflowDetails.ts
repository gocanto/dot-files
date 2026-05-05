export type WorkflowCategory = "template" | "current" | "update" | "settings";

export interface WorkflowDetail {
  action: string;
  category: WorkflowCategory;
  purpose: string;
  details: string;
  whenToRun: string;
  sideEffects: string[];
  prerequisites: string[];
}

export const workflowDetails: Record<string, WorkflowDetail> = {
  "preview-template": {
    action: "Previews",
    category: "template",
    purpose: "Print the tracked source of truth so you can see exactly what the template defines.",
    details:
      "Outputs the tracked Homebrew bundle (formulae and casks), the apps from apps.yaml grouped by install method, the tracked macOS defaults, and the dotfile bundles under stow/. Read-only.",
    whenToRun: "Before any converge or remove workflow, or to confirm what the template currently declares.",
    sideEffects: [],
    prerequisites: [],
  },
  "validate-template": {
    action: "Validates",
    category: "template",
    purpose: "Check that apps.yaml, secrets.yaml, the stow directory, the tracked Brewfile, and the tracked macOS settings are well-formed.",
    details:
      "Loads and validates each tracked manifest. Reports parse errors, missing files, or invalid install methods. Read-only.",
    whenToRun: "After editing any template file, or before opening a PR that changes apps.yaml/secrets.yaml.",
    sideEffects: [],
    prerequisites: [],
  },
  "inspect-current": {
    action: "Inspects",
    category: "current",
    purpose: "Read-only snapshot of what is actually installed and configured on this Mac right now.",
    details:
      "Runs the doctor checks (git, gh, node, go, op, etc.), lists installed Homebrew formulae and casks via `brew list`, and prints the current values of tracked macOS defaults domains.",
    whenToRun: "When something feels off, or before deciding whether to converge / remove.",
    sideEffects: [],
    prerequisites: [],
  },
  "regenerate-installed-list": {
    action: "Generates",
    category: "current",
    purpose: "Regenerate the candidate apps list by scanning what is actually installed on this Mac.",
    details:
      "Scans GUI applications, Homebrew casks, and Mac App Store apps, then writes the merged result to apps.generated.yaml so you can review the diff against the tracked apps.yaml. The tracked apps.yaml source of truth is never modified.",
    whenToRun: "After installing or removing apps, or when preparing a PR that updates the tracked apps list.",
    sideEffects: ["Writes apps.generated.yaml at the configured generated path (never touches apps.yaml)"],
    prerequisites: [],
  },
  "save-snapshot": {
    action: "Snapshots",
    category: "current",
    purpose: "Capture supported app configs, dotfiles, and macOS defaults exports into a reviewable archive.",
    details:
      "Collects only the app settings and reference files that are explicitly supported by the template, then writes them as a dated archive under the archive root. Selective — not a full Mac backup. Preview lists what would be collected without writing.",
    whenToRun: "Before a risky change, before erasing the Mac, or any time you want a known-good restore point.",
    sideEffects: ["Writes a new dated snapshot archive under the archive root"],
    prerequisites: ["Write access to the configured archive root"],
  },
  "converge-to-template": {
    action: "Converges",
    category: "update",
    purpose: "Apply the tracked template (Homebrew, apps, secrets, dotfiles, macOS settings) to this Mac.",
    details:
      "Runs the shared converge pipeline. Fresh setup adopts existing dotfiles into the repo (for clean or freshly-erased Macs). Re-converge skips the adopt step and restores app configs from the latest snapshot. Erase first opens Apple's reset assistant. Preview shows the plan without changing anything.",
    whenToRun: "First boot of a new Mac (Fresh setup), or after pulling repo changes / editing the tracked policy (Re-converge).",
    sideEffects: [
      "Installs Homebrew, formulas, casks, and Mac App Store apps",
      "Writes SSH keys and GPG signing config for GitHub",
      "Decrypts secrets from 1Password and writes them to disk",
      "Links dotfiles into $HOME via stow",
      "Applies macOS defaults (Finder, Dock, keyboard, etc.)",
    ],
    prerequisites: [
      "Signed in to the configured 1Password CLI vault",
      "Internet access for Homebrew and the App Store",
      "Repository cloned at the configured root",
    ],
  },
  "restore-snapshot": {
    action: "Restores",
    category: "update",
    purpose: "Replay a previously saved snapshot's app configs onto this Mac.",
    details:
      "Reads a prior snapshot from the archive root and restores the supported app config files into their expected locations. Preview shows the restore plan without touching any files.",
    whenToRun: "After re-installing an app, after a setup change you want to undo, or as part of recovering a Mac to a known state.",
    sideEffects: ["Overwrites targeted app configuration files with snapshot contents"],
    prerequisites: ["At least one snapshot exists under the archive root"],
  },
  "remove-untracked-apps": {
    action: "Removes",
    category: "update",
    purpose: "Uninstall Homebrew formulae, casks, and Mac App Store apps that are not in the tracked template.",
    details:
      "Scans installed Homebrew formulae and casks against the tracked Brewfile, plus the Mac App Store install list against apps.yaml. Preview lists candidates without changing anything. Run now writes a snapshot first, then runs `brew uninstall --zap` for each untracked Homebrew item; App Store removal is best-effort via `mas uninstall` and is gated off by default — untracked App Store apps are reported for manual cleanup.",
    whenToRun: "After auditing what is installed (Regenerate Installed App List) and confirming the tracked Brewfile / apps.yaml is the desired state.",
    sideEffects: [
      "Writes a pre-remove snapshot under the archive root",
      "Uninstalls Homebrew formulae and casks not present in the tracked Brewfile",
      "Reports (does not uninstall by default) untracked Mac App Store apps for manual cleanup",
    ],
    prerequisites: [
      "Tracked Brewfile in internal/brewfile/brewfile.go reflects the desired state",
      "apps.yaml `appStore` entries reflect the desired Mac App Store apps",
    ],
  },
};

const empty: WorkflowDetail = {
  action: "",
  category: "template",
  purpose: "",
  details: "",
  whenToRun: "",
  sideEffects: [],
  prerequisites: [],
};

export function getWorkflowDetail(id: string): WorkflowDetail {
  return workflowDetails[id] ?? empty;
}

export function workflowDetailHaystack(id: string): string {
  const detail = workflowDetails[id];
  if (!detail) return "";
  return [detail.action, detail.purpose, detail.details, detail.whenToRun, ...detail.sideEffects, ...detail.prerequisites].join(" ");
}

export function workflowsInCategory<T extends { id: string }>(workflows: T[], category: WorkflowCategory): T[] {
  return workflows.filter((workflow) => workflowDetails[workflow.id]?.category === category);
}

const actionPillClasses: Record<string, string> = {
  "preview-template":
    "border-zinc-400/30 bg-zinc-400/10 text-zinc-600 dark:text-zinc-300",
  "validate-template":
    "border-cyan-500/30 bg-cyan-500/10 text-cyan-600 dark:text-cyan-300",
  "inspect-current":
    "border-slate-400/30 bg-slate-400/10 text-slate-600 dark:text-slate-300",
  "regenerate-installed-list":
    "border-cyan-500/30 bg-cyan-500/10 text-cyan-600 dark:text-cyan-300",
  "save-snapshot":
    "border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300",
  "converge-to-template":
    "border-indigo-500/30 bg-indigo-500/10 text-indigo-600 dark:text-indigo-300",
  "restore-snapshot":
    "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-300",
  "remove-untracked-apps":
    "border-rose-500/30 bg-rose-500/10 text-rose-600 dark:text-rose-300",
};

export function workflowActionPillClass(id: string): string {
  return actionPillClasses[id] ?? "border-muted bg-muted/30 text-muted-foreground";
}
