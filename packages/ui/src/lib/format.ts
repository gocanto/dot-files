export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

export function formatDate(value?: string) {
  if (!value) {
    return "Not recorded";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

export function timeAgo(value?: string) {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  const seconds = Math.max(1, Math.round((Date.now() - date.getTime()) / 1000));
  const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ["year", 60 * 60 * 24 * 365],
    ["month", 60 * 60 * 24 * 30],
    ["week", 60 * 60 * 24 * 7],
    ["day", 60 * 60 * 24],
    ["hour", 60 * 60],
    ["minute", 60],
  ];
  const formatter = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  const match = units.find(([, unitSeconds]) => seconds >= unitSeconds);

  if (!match) {
    return "just now";
  }

  const [unit, unitSeconds] = match;

  return formatter.format(-Math.floor(seconds / unitSeconds), unit);
}

export function initials(value: string) {
  return value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 3)
    .map((chunk) => chunk[0]?.toUpperCase())
    .join("");
}
