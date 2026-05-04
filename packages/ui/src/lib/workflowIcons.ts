import { ArchiveRestore, Camera, Download, FileCode2, FileText, Printer, RefreshCw, ShieldCheck, Wand2 } from "lucide-vue-next";

const workflowActionIcons: Record<string, typeof Download> = {
  "set-up-this-mac": Download,
  "update-this-mac": RefreshCw,
  "save-app-settings-snapshot": Camera,
  "restore-app-settings": ArchiveRestore,
  "update-installed-app-list": FileCode2,
  "apply-macos-settings": Wand2,
  "check-setup": ShieldCheck,
  "show-homebrew-packages": Printer,
};

export function workflowActionIcon(id: string) {
  return workflowActionIcons[id] ?? FileText;
}
