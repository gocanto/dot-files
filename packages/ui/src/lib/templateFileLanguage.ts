export function templateFileLanguage(path: string): string {
  const normalized = path.trim().toLowerCase();
  const basename = normalized.split(/[\\/]/).at(-1) ?? normalized;

  if (basename.endsWith(".yaml") || basename.endsWith(".yml")) {
    return "yaml";
  }

  if (
    basename === ".zshrc" ||
    basename === ".zprofile" ||
    basename === ".bash_profile" ||
    basename.endsWith(".sh")
  ) {
    return "shell";
  }

  if (basename.endsWith(".json")) {
    return "json";
  }

  return "plaintext";
}
