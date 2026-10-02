#!/usr/bin/env bash
# vendor-snapshot.sh — store and restore vendor/ in an OCI registry with oras.
#
# CI keeps a snapshot of vendor/ in GHCR, tagged by the sha256 of go.sum, so
# jobs restore dependencies instead of downloading them. This script is the
# one place that knows how a snapshot is stored.
#
# It needs only oras, jq and tar — no Docker daemon. The self-hosted runners
# are plain pods with no daemon, so `docker pull` / `docker build` cannot run
# there.
#
# Usage (run from the repo root):
#   hack/vendor-snapshot.sh exists  <repo> <tag>
#   hack/vendor-snapshot.sh restore <repo> <tag> [fallback-tag...]
#   hack/vendor-snapshot.sh push    <repo> <tag> [extra-tag...]
#
#   exists   exit 0 if <repo>:<tag> is in the registry, 1 if not.
#   restore  replace ./vendor with the first of the given tags that exists.
#            Exit 1 if none could be restored, leaving no ./vendor behind.
#   push     upload ./vendor as <repo>:<tag>, then point each extra tag at it.
#
# Registry credentials come from a prior `docker login` / `oras login`.
#
# Local cache: if VENDOR_SNAPSHOT_CACHE_DIR is set — or ZENITH_CACHE_DIR, which
# the self-hosted runners export for their persistent cache volume — a restored
# or pushed snapshot is also kept there as <repo>/<tag>.tar.gz, and `restore`
# uses that copy instead of downloading. The `latest` tag moves, so it is never
# served from the local cache.
#
# Snapshot format: a manifest whose layers are tarballs holding a top-level
# vendor/ directory. `push` writes a single gzip layer. `restore` unpacks
# vendor/ from every layer, which also reads the snapshots that were built
# with `docker build` from a `FROM scratch` + `COPY vendor/ /vendor/` image.
set -euo pipefail

ARTIFACT_TYPE="application/vnd.blanketops.vendor-snapshot.v1"
LAYER_MEDIA_TYPE="application/vnd.oci.image.layer.v1.tar+gzip"

CACHE_DIR="${VENDOR_SNAPSHOT_CACHE_DIR:-${ZENITH_CACHE_DIR:+${ZENITH_CACHE_DIR}/vendor-snapshots}}"

# Print the local cache file for <repo>:<tag>, or nothing if caching is off or
# the tag is one that moves.
cache_file() {
	local repo=$1 tag=$2
	[ -n "${CACHE_DIR}" ] && [ "${tag}" != "latest" ] || return 0
	printf '%s/%s/%s.tar.gz' "${CACHE_DIR}" "${repo//\//_}" "${tag}"
}

# Store ./vendor (or an already-built tarball, if given) as the local copy of
# <repo>:<tag>. Best-effort: a cache that cannot be written is not an error.
cache_store() {
	local repo=$1 tag=$2 tarball=${3:-} file tmp
	file=$(cache_file "${repo}" "${tag}")
	[ -n "${file}" ] || return 0
	tmp="${file}.tmp.$$"
	{
		mkdir -p "$(dirname "${file}")" &&
			if [ -n "${tarball}" ]; then cp "${tarball}" "${tmp}"; else tar -czf "${tmp}" vendor; fi &&
			mv -f "${tmp}" "${file}"
	} 2>/dev/null || rm -f "${tmp}"
}

die() {
	echo "vendor-snapshot: $*" >&2
	exit 1
}

usage() {
	sed -n '/^# Usage/,/^# Registry credentials/p' "$0" | sed 's/^# \{0,1\}//' >&2
	exit 2
}

# Print the image manifest for <repo>:<tag>, following an image index to its
# linux/amd64 entry (or its first entry if it has no such platform).
fetch_manifest() {
	local repo=$1 tag=$2 manifest digest
	manifest=$(oras manifest fetch "${repo}:${tag}" 2>/dev/null) || return 1
	if jq -e '.manifests' >/dev/null <<<"${manifest}"; then
		digest=$(jq -r '
			([.manifests[] | select(.platform.os == "linux" and .platform.architecture == "amd64")][0]
			 // .manifests[0]).digest' <<<"${manifest}")
		manifest=$(oras manifest fetch "${repo}@${digest}" 2>/dev/null) || return 1
	fi
	printf '%s' "${manifest}"
}

# Unpack vendor/ from every layer of <repo>:<tag> into ./vendor.
restore_tag() {
	local repo=$1 tag=$2 manifest layers digest media_type status cached
	local -a tar_flags

	cached=$(cache_file "${repo}" "${tag}")
	if [ -n "${cached}" ] && [ -f "${cached}" ]; then
		if tar -xzf "${cached}" vendor/ 2>/dev/null && [ -f vendor/modules.txt ]; then
			echo "Using local copy ${cached}"
			return 0
		fi
		echo "vendor-snapshot: discarding unreadable local copy ${cached}" >&2
		rm -rf ./vendor "${cached}"
	fi

	manifest=$(fetch_manifest "${repo}" "${tag}") || return 1
	layers=$(jq -r '.layers[] | "\(.digest) \(.mediaType)"' <<<"${manifest}")
	[ -n "${layers}" ] || return 1

	while read -r digest media_type; do
		case "${media_type}" in
		*gzip) tar_flags=(-xzf -) ;;
		*zstd) tar_flags=(--zstd -xf -) ;;
		*) tar_flags=(-xf -) ;;
		esac

		# A layer with no vendor/ in it makes tar exit non-zero; that is fine.
		# A failed or corrupt download is not — oras verifies the digest and
		# reports it through its own exit status, checked separately here.
		set +e
		oras blob fetch --output - "${repo}@${digest}" | tar "${tar_flags[@]}" vendor/ 2>/dev/null
		status=("${PIPESTATUS[@]}")
		set -e
		if [ "${status[0]}" -ne 0 ]; then
			echo "vendor-snapshot: failed to fetch layer ${digest} of ${repo}:${tag}" >&2
			rm -rf ./vendor
			return 1
		fi
	done <<<"${layers}"

	if [ ! -f vendor/modules.txt ]; then
		echo "vendor-snapshot: ${repo}:${tag} has no vendor/modules.txt" >&2
		rm -rf ./vendor
		return 1
	fi
	cache_store "${repo}" "${tag}"
}

cmd_exists() {
	[ $# -eq 2 ] || usage
	oras manifest fetch "$1:$2" >/dev/null 2>&1
}

cmd_restore() {
	[ $# -ge 2 ] || usage
	local repo=$1 tag
	shift
	rm -rf ./vendor
	for tag in "$@"; do
		if restore_tag "${repo}" "${tag}"; then
			echo "vendor/ restored from ${repo}:${tag} — $(find vendor -mindepth 1 -maxdepth 1 -type d | wc -l) packages"
			return 0
		fi
		echo "No usable snapshot at ${repo}:${tag}"
	done
	return 1
}

cmd_push() {
	[ $# -ge 2 ] || usage
	local repo=$1 tag=$2 workdir root
	shift 2

	[ -f vendor/modules.txt ] || die "no vendor/modules.txt here — run 'go mod vendor' first"

	root=$(pwd)
	workdir=$(mktemp -d)
	# shellcheck disable=SC2064 # expand workdir now, not when the trap fires
	trap "rm -rf '${workdir}'" EXIT

	tar -czf "${workdir}/vendor.tar.gz" -C "${root}" vendor
	# oras records the file's path as given, so push from inside workdir.
	(cd "${workdir}" && oras push "${repo}:${tag}" \
		--artifact-type "${ARTIFACT_TYPE}" \
		"vendor.tar.gz:${LAYER_MEDIA_TYPE}")

	# A push to an existing tag replaces a stale snapshot, so replace the local
	# copy too rather than keep serving the old one.
	cache_store "${repo}" "${tag}" "${workdir}/vendor.tar.gz"

	if [ $# -gt 0 ]; then
		oras tag "${repo}:${tag}" "$@"
	fi
}

[ $# -ge 1 ] || usage
command=$1
shift

for tool in oras jq tar; do
	command -v "${tool}" >/dev/null || die "${tool} is required but not installed"
done

case "${command}" in
exists) cmd_exists "$@" ;;
restore) cmd_restore "$@" ;;
push) cmd_push "$@" ;;
*) usage ;;
esac
