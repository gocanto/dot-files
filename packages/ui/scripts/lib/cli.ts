export function failWithMatches(matches: string[]): void {
  if (matches.length === 0) {
    return;
  }

  console.error(matches.join("\n"));
  process.exit(1);
}
