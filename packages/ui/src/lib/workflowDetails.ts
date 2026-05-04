export type WorkflowCategory = "this-mac" | "snapshots" | "health" | "settings";

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
  "set-up-this-mac": {
    action: "Installs",
    category: "this-mac",
    purpose: "Bootstrap a fresh or freshly erased Mac end-to-end so it matches the tracked policy.",
    details:
      "Runs the eleven-phase install pipeline: prerequisites (Xcode CLT, brew, git), Homebrew bundle, GitHub SSH/GPG, App Store apps, manual-app notes, 1Password-decrypted private secrets, dotfile preparation, oh-my-zsh, dotfile linking via stow, macOS defaults, and a final health check. Preview mode walks the same plan without writing or installing anything.",
    whenToRun: "First boot of a new or freshly erased Mac, or any time you intentionally want to recreate the full setup from scratch.",
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
  "update-this-mac": {
    action: "Updates",
    category: "this-mac",
    purpose: "Re-converge an already-set-up Mac to the tracked policy without overwriting tracked dotfiles from this machine.",
    details:
      "Runs the same install pipeline as Set Up This Mac, with two differences: it skips the dotfile-adoption step (so local edits do not get pushed back into the repo), and it restores supported app configs from the latest local snapshot under the archive root instead of leaving them as-is.",
    whenToRun: "After pulling repo changes, after editing the brew or apps lists, or periodically to keep this Mac aligned with the tracked policy.",
    sideEffects: [
      "Installs or updates Homebrew packages and apps to match the tracked policy",
      "Re-applies SSH/GPG and private secrets",
      "Restores app configs from the most recent snapshot",
      "Re-applies macOS defaults",
    ],
    prerequisites: [
      "Signed in to the configured 1Password CLI vault",
      "At least one prior snapshot under the archive root for the restore step",
    ],
  },
  "save-app-settings-snapshot": {
    action: "Snapshots",
    category: "snapshots",
    purpose: "Capture supported app configs into a reviewable archive so they can be restored later.",
    details:
      "Collects only the app settings and reference files that are explicitly supported by this setup, then writes them as a dated archive under the archive root. This is selective — not a full Mac backup. Preview mode lists what would be collected without writing anything.",
    whenToRun: "Before a risky change, before erasing the Mac, or any time you want a known-good restore point.",
    sideEffects: ["Writes a new dated snapshot archive under the archive root"],
    prerequisites: ["Write access to the configured archive root"],
  },
  "restore-app-settings": {
    action: "Restores",
    category: "snapshots",
    purpose: "Replay a previously saved snapshot's app configs onto this Mac.",
    details:
      "Reads a prior snapshot from the archive root and restores the supported app config files into their expected locations. Preview shows the restore plan without touching any files.",
    whenToRun: "After re-installing an app, after a setup change you want to undo, or as part of recovering a Mac to a known state.",
    sideEffects: ["Overwrites targeted app configuration files with snapshot contents"],
    prerequisites: ["At least one snapshot exists under the archive root"],
  },
  "update-installed-app-list": {
    action: "Generates",
    category: "this-mac",
    purpose: "Regenerate the candidate apps list by scanning what is actually installed on this Mac.",
    details:
      "Scans GUI applications, Homebrew casks, and Mac App Store apps, then writes the merged result to apps.generated.yaml so you can review the diff against the tracked apps.yaml. The tracked apps.yaml source of truth is never modified by this workflow.",
    whenToRun: "After installing or removing apps, or when preparing a PR that updates the tracked apps list.",
    sideEffects: ["Writes apps.generated.yaml at the configured generated path (never touches apps.yaml)"],
    prerequisites: [],
  },
  "apply-macos-settings": {
    action: "Applies",
    category: "settings",
    purpose: "Re-apply the tracked macOS preferences (Finder, Dock, keyboard, etc.) to this Mac.",
    details:
      "Runs the tracked defaults-style commands that customize macOS behavior. Preview prints every command without executing it, so you can review the exact changes before applying.",
    whenToRun: "After editing the macOS settings list, after a macOS upgrade reset some preferences, or when you want to restore your tweaks.",
    sideEffects: ["Runs defaults write and similar commands that change macOS preferences"],
    prerequisites: [],
  },
  "check-setup": {
    action: "Checks",
    category: "health",
    purpose: "Read-only diagnostic that reports whether the expected tools, paths, and configuration look correct.",
    details:
      "Runs the doctor-style checks: verifies Homebrew, Git, Xcode CLT, 1Password CLI, that the secrets file and dotfile links exist, and that runtime settings point at valid paths. Reports findings without changing anything.",
    whenToRun: "Anytime you suspect something is off, or before reporting an issue.",
    sideEffects: [],
    prerequisites: [],
  },
  "show-homebrew-packages": {
    action: "Prints",
    category: "this-mac",
    purpose: "Print the resolved Homebrew bundle that Set Up and Update would install.",
    details:
      "Outputs the generated Homebrew package list to the workflow output pane so you can review exactly which formulas and casks are tracked.",
    whenToRun: "Before running an install workflow, or to confirm the tracked package list matches expectations.",
    sideEffects: [],
    prerequisites: [],
  },
};

const empty: WorkflowDetail = {
  action: "",
  category: "this-mac",
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
  "set-up-this-mac":
    "border-indigo-500/30 bg-indigo-500/10 text-indigo-600 dark:text-indigo-300",
  "update-this-mac":
    "border-sky-500/30 bg-sky-500/10 text-sky-600 dark:text-sky-300",
  "save-app-settings-snapshot":
    "border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300",
  "restore-app-settings":
    "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-300",
  "update-installed-app-list":
    "border-cyan-500/30 bg-cyan-500/10 text-cyan-600 dark:text-cyan-300",
  "apply-macos-settings":
    "border-violet-500/30 bg-violet-500/10 text-violet-600 dark:text-violet-300",
  "check-setup":
    "border-slate-400/30 bg-slate-400/10 text-slate-600 dark:text-slate-300",
  "show-homebrew-packages":
    "border-zinc-400/30 bg-zinc-400/10 text-zinc-600 dark:text-zinc-300",
};

export function workflowActionPillClass(id: string): string {
  return actionPillClasses[id] ?? "border-muted bg-muted/30 text-muted-foreground";
}
