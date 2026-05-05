import { ArchiveRestore, Camera, Download, Eye, FileCode2, FileText, RefreshCw, ShieldCheck, Trash2 } from "lucide-vue-next";

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
