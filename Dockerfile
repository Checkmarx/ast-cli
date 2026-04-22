# TODO: Update base image to a version with glibc >= 2.43-r6 to fix OS-layer vulnerabilities
# (CVE-2026-4046, CVE-2026-4437, CVE-2026-4438 in glibc/glibc-locale-posix/ld-linux/libcrypt1,
#  CVE-2026-2673 in libcrypto3/libssl3). Requires Checkmarx to publish an updated base image.
FROM checkmarx/bash:5.3-r5-98621acba7807a@sha256:98621acba7807a4e128f3e00aba3987e4f659ff352191f79cdbaa7f8a32cfb58
USER nonroot

COPY cx /app/bin/cx

ENTRYPOINT ["/app/bin/cx"]

HEALTHCHECK NONE
