#!/usr/bin/env python3

from __future__ import annotations

import hashlib
import json
import os
import sys
import urllib.request
from pathlib import Path


REPO = "jscaltreto/downstage"
HOMEPAGE = "https://github.com/jscaltreto/downstage"
DESCRIPTION = "Plaintext markup language and tools for stage plays"
LICENSE = "MIT"


def fail(message: str) -> None:
    print(message, file=sys.stderr)
    raise SystemExit(1)


def github_get(url: str, token: str) -> dict:
    req = urllib.request.Request(
        url,
        headers={
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {token}",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    with urllib.request.urlopen(req) as response:
        return json.load(response)


def github_download(url: str, token: str) -> bytes:
    req = urllib.request.Request(
        url,
        headers={
            "Accept": "application/octet-stream",
            "Authorization": f"Bearer {token}",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    with urllib.request.urlopen(req) as response:
        return response.read()


def parse_checksums(data: bytes) -> dict[str, str]:
    checksums: dict[str, str] = {}
    for raw_line in data.decode("utf-8").splitlines():
        line = raw_line.strip()
        if not line:
            continue
        checksum, filename = line.split(maxsplit=1)
        checksums[filename.strip()] = checksum
    return checksums


def asset_info(release: dict, checksums: dict[str, str], name: str) -> tuple[str, str]:
    for asset in release["assets"]:
        if asset["name"] == name:
            checksum = checksums.get(name)
            if not checksum:
                fail(f"missing checksum for asset {name}")
            return asset["browser_download_url"], checksum
    fail(f"missing release asset {name}")


def formula_content(version: str, release: dict, checksums: dict[str, str]) -> str:
    assets = {
        "darwin_amd64": asset_info(release, checksums, f"downstage_{version}_Darwin_x86_64.tar.gz"),
        "darwin_arm64": asset_info(release, checksums, f"downstage_{version}_Darwin_arm64.tar.gz"),
        "linux_amd64": asset_info(release, checksums, f"downstage_{version}_Linux_x86_64.tar.gz"),
        "linux_arm64": asset_info(release, checksums, f"downstage_{version}_Linux_arm64.tar.gz"),
    }

    return f"""class Downstage < Formula
  desc "{DESCRIPTION}"
  homepage "{HOMEPAGE}"
  version "{version.removeprefix("v")}"
  license "{LICENSE}"

  on_macos do
    on_intel do
      url "{assets["darwin_amd64"][0]}"
      sha256 "{assets["darwin_amd64"][1]}"
    end

    on_arm do
      url "{assets["darwin_arm64"][0]}"
      sha256 "{assets["darwin_arm64"][1]}"
    end
  end

  on_linux do
    on_intel do
      url "{assets["linux_amd64"][0]}"
      sha256 "{assets["linux_amd64"][1]}"
    end

    on_arm do
      url "{assets["linux_arm64"][0]}"
      sha256 "{assets["linux_arm64"][1]}"
    end
  end

  def install
    bin.install "downstage"
  end

  test do
    assert_match version.to_s, shell_output("#{{bin}}/downstage version")
  end
end
"""


def main() -> None:
    tag = os.environ.get("RELEASE_TAG")
    token = os.environ.get("GITHUB_TOKEN")
    output = os.environ.get("FORMULA_PATH")
    if not tag:
        fail("RELEASE_TAG is required")
    if not token:
        fail("GITHUB_TOKEN is required")
    if not output:
        fail("FORMULA_PATH is required")

    release = github_get(f"https://api.github.com/repos/{REPO}/releases/tags/{tag}", token)

    checksum_asset = next((asset for asset in release["assets"] if asset["name"] == "checksums.txt"), None)
    if checksum_asset is None:
        fail("missing checksums.txt release asset")

    checksum_data = github_download(checksum_asset["url"], token)
    checksums = parse_checksums(checksum_data)

    content = formula_content(tag, release, checksums)
    formula_path = Path(output)
    formula_path.parent.mkdir(parents=True, exist_ok=True)
    formula_path.write_text(content, encoding="utf-8")

    digest = hashlib.sha256(content.encode("utf-8")).hexdigest()
    print(f"wrote {formula_path} ({digest})")


if __name__ == "__main__":
    main()
