import {
  Activity,
  Apple,
  AppWindow,
  ArchiveRestore,
  ArrowLeft,
  Beer,
  Camera,
  Circle,
  Download,
  Eye,
  FileCheck2,
  FileCode2,
  Files,
  FileText,
  FolderOpen,
  Github,
  Link2,
  ListChecks,
  Lock,
  Play,
  Printer,
  RefreshCw,
  Search,
  ShieldCheck,
  Sliders,
  TerminalSquare,
  Trash2,
  Wand2,
} from "lucide-vue-next";

const workflowActionIcons: Record<string, typeof Download> = {
  "preview-template": Eye,
  "validate-template": ShieldCheck,
  "inspect-current": ShieldCheck,
  "regenerate-installed-list": FileCode2,
  "save-snapshot": Camera,
  "converge-to-template": RefreshCw,
  "restore-snapshot": ArchiveRestore,
  "remove-untracked-apps": Trash2,
};

export function workflowActionIcon(id: string) {
  return workflowActionIcons[id] ?? FileText;
}

const phaseIcons: Record<string, typeof Download> = {
  "check-install-prerequisites": ListChecks,
  "install-homebrew-packages": Beer,
  "set-up-github-access-and-signing": Github,
  "install-app-store-apps": Apple,
  "show-manual-app-install-notes": FileText,
  "restore-private-secrets-from-1password": Lock,
  "prepare-existing-dotfiles": FolderOpen,
  "install-oh-my-zsh": TerminalSquare,
  "link-dotfiles": Link2,
  "apply-macos-settings": Wand2,
  "apply-tracked-macos-settings": Wand2,
  "run-health-checks": Activity,
  "restore-supported-app-configs-from-latest-snapshot": ArchiveRestore,
  "restore-supported-app-settings": ArchiveRestore,
  "save-supported-app-settings-snapshot": Camera,
  "generate-installed-app-list-candidate": FileCode2,
  "print-generated-homebrew-package-list": Printer,
  "print-tracked-homebrew-bundle": Printer,
  "list-tracked-apps": AppWindow,
  "list-tracked-macos-settings": Sliders,
  "list-tracked-dotfile-bundles": Files,
  "validate-template-files": FileCheck2,
  "list-installed-homebrew-formulae-and-casks": Beer,
  "show-current-macos-defaults-values": Sliders,
  "scan-untracked-items": Search,
  "snapshot-before-remove": Camera,
  "uninstall-untracked-homebrew-formulae-and-casks": Trash2,
  "uninstall-untracked-app-store-apps-best-effort": Trash2,
};

export function phaseIcon(id: string) {
  return phaseIcons[id] ?? Circle;
}

const confirmationIcons: Record<string, typeof Play> = {
  "preview-only": Eye,
  "run-now": Play,
  "already-erased-run-now": Play,
  "run-without-erasing": Play,
  "erase-first": Trash2,
  back: ArrowLeft,
};

export function confirmationIcon(id: string) {
  return confirmationIcons[id] ?? Play;
}
