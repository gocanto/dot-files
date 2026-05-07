import { execFileSync } from "node:child_process";
import { existsSync } from "node:fs";
import os from "node:os";
import { pathToFileURL } from "node:url";

export function architectureLabel(architecture: string) {
  if (architecture === "arm64") return "Apple silicon";
  if (architecture === "x64") return "Intel";

  return architecture;
}

export function osLabel() {
  const type = os.type();
  const release =
    type === "Darwin"
      ? execFileSync("/usr/bin/sw_vers", ["-productVersion"], {
          encoding: "utf8",
          stdio: ["ignore", "pipe", "ignore"],
          timeout: 1000,
        }).trim()
      : os.release();

  return type === "Darwin" ? `macOS ${release}` : `${type} ${release}`;
}

function dsclRead(user: string, key: "JPEGPhoto" | "Picture") {
  try {
    return execFileSync("dscl", [".", "-read", `/Users/${user}`, key], {
      encoding: "utf8",
      maxBuffer: 1024 * 1024 * 4,
      stdio: ["ignore", "pipe", "ignore"],
      timeout: 1000,
    });
  } catch {
    return "";
  }
}

export function accountAvatarUrl() {
  if (process.platform !== "darwin") {
    return undefined;
  }

  const username = os.userInfo().username;
  const jpegPhoto = dsclRead(username, "JPEGPhoto");
  const jpegHex = jpegPhoto.replace(/^JPEGPhoto:\s*/u, "").replace(/[^a-fA-F0-9]/gu, "");

  if (jpegHex.length >= 2) {
    try {
      const image = Buffer.from(jpegHex, "hex");

      if (image.length > 0) {
        return `data:image/jpeg;base64,${image.toString("base64")}`;
      }
    } catch {
      // Fall back to the Picture path below.
    }
  }

  const picture = dsclRead(username, "Picture")
    .replace(/^Picture:\s*/u, "")
    .trim();

  if (!picture || !existsSync(picture)) {
    return undefined;
  }

  return pathToFileURL(picture).toString();
}
