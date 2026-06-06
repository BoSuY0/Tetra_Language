# Installing Tetra

Tetra's current installable release channel is scoped to the checked `v0.4.0`
local compiler/tooling profile. The binary release baseline is `linux-x64`.
Other targets remain governed by the target evidence in the release docs.

## Linux x64 Release Installer

Use the GitHub Release installer after a matching release exists:

```sh
curl -fsSL https://raw.githubusercontent.com/BoSuY0/Tetra_Language/main/install.sh | bash
tetra version
```

The installer downloads `tetra-v0.4.0-linux-x64.tar.gz`, verifies
`checksums.txt`, and installs `tetra` plus the `t` alias into
`${HOME}/.local/bin` by default.

Override the install directory or version with:

```sh
TETRA_INSTALL_DIR="$HOME/bin" TETRA_VERSION=v0.4.0 \
  bash -c "$(curl -fsSL https://raw.githubusercontent.com/BoSuY0/Tetra_Language/main/install.sh)"
```

## Container Image

The release workflow publishes a GHCR image:

```sh
docker run --rm ghcr.io/bosuy0/tetra-language:0.4.0 tetra version
docker run --rm -v "$PWD:/work" ghcr.io/bosuy0/tetra-language:0.4.0 \
  tetra check /work/examples/flow_hello.tetra
```

If the repository or package is private, authenticate to GHCR before pulling.

## Homebrew

The release workflow renders a Homebrew formula for the tap repository
`BoSuY0/homebrew-tetra`.

```sh
brew tap BoSuY0/tetra
brew install tetra
tetra version
```

The formula builds Tetra from the release source archive with Go. The workflow
can also push `Formula/tetra.rb` to the tap when `HOMEBREW_TAP_TOKEN` is
configured and `update_homebrew_tap` is enabled.
