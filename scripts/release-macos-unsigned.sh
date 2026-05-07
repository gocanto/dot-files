#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
UI_DIR="$ROOT_DIR/packages/ui"
MACBOOK_DIR="$ROOT_DIR/packages/macbook"

NOTES_FILE=""
RELEASE_REPO="${RELEASE_REPO:-}"

while [ $# -gt 0 ]; do
  case "$1" in
    --notes-file)
      NOTES_FILE="$2"
      shift 2
      ;;
    --notes-file=*)
      NOTES_FILE="${1#*=}"
      shift
      ;;
    --repo)
      RELEASE_REPO="$2"
      shift 2
      ;;
    --repo=*)
      RELEASE_REPO="${1#*=}"
      shift
      ;;
    *)
      echo "Error: unknown argument: $1"
      echo "Usage: $0 --notes-file <path> [--repo owner/name]"
      exit 1
      ;;
  esac
done

if [ -z "$NOTES_FILE" ]; then
  echo "Error: --notes-file <path> is required"
  exit 1
fi

if [ ! -s "$NOTES_FILE" ]; then
  echo "Error: notes file is missing or empty: $NOTES_FILE"
  exit 1
fi

for command in git gh pnpm node shasum; do
  if ! command -v "$command" >/dev/null 2>&1; then
    echo "Error: required command not found: $command"
    exit 1
  fi
done

if [ -z "$RELEASE_REPO" ]; then
  RELEASE_REPO="$(gh repo view --json nameWithOwner --jq '.nameWithOwner')"
fi

VERSION="$(node -e "console.log(require('$UI_DIR/package.json').version)")"
TAG="v$VERSION"
HEAD="$(git -C "$ROOT_DIR" rev-parse HEAD)"
CURRENT_BRANCH="$(git -C "$ROOT_DIR" rev-parse --abbrev-ref HEAD)"
DEFAULT_BRANCH="$(git -C "$ROOT_DIR" symbolic-ref --short refs/remotes/origin/HEAD 2>/dev/null | sed 's#^origin/##' || true)"
DEFAULT_BRANCH="${DEFAULT_BRANCH:-main}"

if [ "$CURRENT_BRANCH" != "$DEFAULT_BRANCH" ]; then
  echo "Error: releases must be cut from $DEFAULT_BRANCH, currently on $CURRENT_BRANCH"
  exit 1
fi

if ! git -C "$ROOT_DIR" diff-index --quiet HEAD --; then
  echo "Error: working tree has uncommitted changes"
  git -C "$ROOT_DIR" status --short
  exit 1
fi

echo "Fetching origin and tags..."
git -C "$ROOT_DIR" fetch origin "$DEFAULT_BRANCH" --tags

LOCAL_REV="$(git -C "$ROOT_DIR" rev-parse HEAD)"
REMOTE_REV="$(git -C "$ROOT_DIR" rev-parse "origin/$DEFAULT_BRANCH")"
BASE_REV="$(git -C "$ROOT_DIR" merge-base HEAD "origin/$DEFAULT_BRANCH")"

if [ "$LOCAL_REV" != "$REMOTE_REV" ] && [ "$BASE_REV" != "$REMOTE_REV" ]; then
  echo "Error: local $DEFAULT_BRANCH is not a fast-forward of origin/$DEFAULT_BRANCH"
  echo "  local:  $LOCAL_REV"
  echo "  origin: $REMOTE_REV"
  exit 1
fi

if git -C "$ROOT_DIR" rev-parse --verify --quiet "refs/tags/$TAG" >/dev/null; then
  echo "Error: tag $TAG already exists locally"
  exit 1
fi

if git -C "$ROOT_DIR" ls-remote --tags --exit-code origin "refs/tags/$TAG" >/dev/null 2>&1; then
  echo "Error: tag $TAG already exists on origin"
  exit 1
fi

echo "Running tests and builds..."
pnpm -C "$ROOT_DIR" test
pnpm -C "$ROOT_DIR" build
pnpm --dir "$MACBOOK_DIR" run build
pnpm --dir "$UI_DIR" run dist:mac:unsigned

RELEASE_DIR="$UI_DIR/release"
DMG_FILE="$(find "$RELEASE_DIR" -maxdepth 1 -type f -name '*.dmg' | head -1)"
ZIP_FILE="$(find "$RELEASE_DIR" -maxdepth 1 -type f -name '*.zip' | head -1)"

if [ -z "$DMG_FILE" ] || [ -z "$ZIP_FILE" ]; then
  echo "Error: expected DMG and ZIP artifacts in $RELEASE_DIR"
  exit 1
fi

CHECKSUMS_FILE="$RELEASE_DIR/SHASUMS256.txt"
(
  cd "$RELEASE_DIR"
  shasum -a 256 "$(basename "$DMG_FILE")" "$(basename "$ZIP_FILE")" > "$CHECKSUMS_FILE"
)

RELEASE_NOTES="$(mktemp)"
cat > "$RELEASE_NOTES" <<EOF
> Private testing build: these macOS artifacts are unsigned while Developer ID approval is pending.
> On first launch, use right-click > Open, or remove quarantine manually:
> \`xattr -dr com.apple.quarantine "/Applications/Mac OS Manager.app"\`

EOF
cat "$NOTES_FILE" >> "$RELEASE_NOTES"

echo "Creating draft release $TAG on $RELEASE_REPO..."
gh release create "$TAG" "$DMG_FILE" "$ZIP_FILE" "$CHECKSUMS_FILE" \
  --repo "$RELEASE_REPO" \
  --target "$HEAD" \
  --title "Mac OS Manager $TAG (unsigned private testing)" \
  --notes-file "$RELEASE_NOTES" \
  --draft

if ! git -C "$ROOT_DIR" rev-parse --verify --quiet "refs/tags/$TAG" >/dev/null; then
  git -C "$ROOT_DIR" tag "$TAG" "$HEAD"
fi

git -C "$ROOT_DIR" push origin "$TAG"

DRAFT_URL="$(gh release view "$TAG" --repo "$RELEASE_REPO" --json url --jq '.url')"
echo "Draft release created: $DRAFT_URL"
