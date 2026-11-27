#!/usr/bin/env bash
# Runs tfplugindocs once per resource prefix (scm, ztna).  Each run uses a
# temporary copy of templates/ with empty sentinel files for the other prefix's
# resources so tfplugindocs skips them (empty template → Render returns nil).
set -euo pipefail

PREFIXES=("scm" "ztna")

collect_types() {
  local prefix="$1"
  grep -rh 'resp\.TypeName = "' internal/provider/ 2>/dev/null \
    | sed 's/.*resp\.TypeName = "//;s/".*//' \
    | grep "^${prefix}_" \
    | sort -u || true
}

mkdir -p docs
for PREFIX in "${PREFIXES[@]}"; do
  echo "Generating docs for prefix: $PREFIX"

  TMPL_DIR=$(mktemp -d)
  cp -r templates/. "$TMPL_DIR/"

  for OTHER in "${PREFIXES[@]}"; do
    [ "$OTHER" = "$PREFIX" ] && continue
    while IFS= read -r TYPE; do
      [ -z "$TYPE" ] && continue
      for SUBDIR in resources data-sources; do
        mkdir -p "$TMPL_DIR/$SUBDIR"
        SENTINEL="$TMPL_DIR/$SUBDIR/${TYPE}.md.tmpl"
        [ -f "$SENTINEL" ] || touch "$SENTINEL"
      done
    done < <(collect_types "$OTHER")
  done

  go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate \
    --provider-name "$PREFIX" \
    --rendered-website-dir "docs-${PREFIX}" \
    --website-source-dir "$TMPL_DIR"

  rm -rf "$TMPL_DIR"

  find "docs-${PREFIX}" -type f | while read -r SRC; do
    [ -s "$SRC" ] || continue
    REL="${SRC#docs-${PREFIX}/}"
    mkdir -p "docs/$(dirname "$REL")"
    cp "$SRC" "docs/$REL"
  done
  rm -rf "docs-${PREFIX}"
done

# Enterprise DLP (scm_dlp_*) is Beta / Experimental. Insert a callout under the
# H1 of every DLP doc page so the notice survives regeneration. Idempotent.
DLP_BETA_CALLOUT='> **Beta / Experimental:** Enterprise DLP support (`scm_dlp_*`) in `terraform-provider-scm` is currently in **Beta / Experimental** and subject to schema or behavior changes.'
for DOC in docs/resources/dlp_*.md docs/data-sources/dlp_*.md docs/actions/dlp_*.md; do
  [ -f "$DOC" ] || continue
  grep -qF -- "$DLP_BETA_CALLOUT" "$DOC" && continue
  CALLOUT="$DLP_BETA_CALLOUT" awk '
    { print }
    !done && /^# scm_dlp_/ { print ""; print ENVIRON["CALLOUT"]; done = 1 }
  ' "$DOC" > "$DOC.tmp" && mv "$DOC.tmp" "$DOC"
done

echo "Documentation generation complete."
