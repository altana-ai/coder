# Building a Coder Release Binary

This runbook documents how to build a full production Coder server binary
(~320MB) from source, suitable for replacing the official release binary
on a deployment.

## Prerequisites

- Go (version matching `go.mod`)
- Node.js <23.0.0 (use `mise install node@22` if needed)
- pnpm 10.x (`corepack enable && corepack prepare pnpm@10.14.0 --activate`)
- zstd

## Step 1: Create a branch from the target release

```bash
git checkout -b fix/my-fix-v2.31.3 v2.31.3
# Cherry-pick or apply your fixes
git cherry-pick <commit-sha>
```

## Step 2: Build the frontend

```bash
pnpm install --dir site
pnpm run --dir site build
```

This produces `site/out/index.html` and the rest of the React app
in `site/out/`.

## Step 3: Build all 7 slim binaries

The fat binary embeds slim CLI binaries for all platforms in a
tar.zst archive. All 7 must be built.

```bash
VERSION="v2.31.3-fix"  # adjust as needed

for arch in linux_amd64 linux_arm64 linux_armv7 \
            darwin_amd64 darwin_arm64 \
            windows_amd64 windows_arm64; do
  os=$(echo $arch | cut -d_ -f1)
  a=$(echo $arch | cut -d_ -f2)
  ext=""
  if [ "$os" = "windows" ]; then ext=".exe"; fi
  ./scripts/build_go.sh \
    --os "$os" --arch "$a" \
    --version "$VERSION" \
    --slim \
    --output "build/coder-slim_${VERSION}_${arch}${ext}"
done
```

## Step 4: Package slim binaries for embedding

This follows the exact Makefile pipeline:

```bash
VERSION="v2.31.3-fix"  # must match step 3

# Copy slim binaries to site/out/bin/ with expected naming
mkdir -p site/out/bin
cp build/coder-slim_${VERSION}_linux_amd64       site/out/bin/coder-linux-amd64
cp build/coder-slim_${VERSION}_linux_arm64        site/out/bin/coder-linux-arm64
cp build/coder-slim_${VERSION}_linux_armv7         site/out/bin/coder-linux-armv7
cp build/coder-slim_${VERSION}_darwin_amd64       site/out/bin/coder-darwin-amd64
cp build/coder-slim_${VERSION}_darwin_arm64        site/out/bin/coder-darwin-arm64
cp build/coder-slim_${VERSION}_windows_amd64.exe  site/out/bin/coder-windows-amd64.exe
cp build/coder-slim_${VERSION}_windows_arm64.exe  site/out/bin/coder-windows-arm64.exe

# Generate sha1 checksums
cd site/out/bin
openssl dgst -r -sha1 coder-* | tee coder.sha1
cd -

# Create tar
cd site/out/bin
tar cf ../../../build/coder-slim_${VERSION}.tar coder-*
cd -

# Remove uncompressed binaries from embedded dir
rm -f site/out/bin/coder-*

# Compress with zstd
zstd --force --long --no-progress \
  -o build/coder-slim_${VERSION}.tar.zst \
  build/coder-slim_${VERSION}.tar

# Copy to site/out/bin/ for embedding
cp build/coder-slim_${VERSION}.tar.zst site/out/bin/coder.tar.zst
```

After this step, `site/out/bin/` should contain only:

- `coder.sha1`
- `coder.tar.zst`

## Step 5: Build the fat (production) binary

```bash
./scripts/build_go.sh \
  --os linux --arch arm64 \
  --version "$VERSION" \
  --output build/coder_${VERSION}_linux_arm64
```

Change `--os` and `--arch` for your target platform.

## Step 6: Verify

```bash
file build/coder_${VERSION}_linux_arm64
# Should show: ELF 64-bit, statically linked, stripped

ls -lh build/coder_${VERSION}_linux_arm64
# Should be ~320MB

./build/coder_${VERSION}_linux_arm64 version
# Should show: Coder <version>+<commit>
# Should show: Full build of Coder, supports the server subcommand.
```

The "Full build" line confirms the frontend and slim binaries are
embedded. If it says "slim build" instead, the embed tag was not
used or `site/out/` was missing assets.

## Key details

- The build script (`scripts/build_go.sh`) uses the `embed` build
  tag for fat binaries and the `slim` tag for slim binaries.
- The `embed` tag triggers `site/site_embed.go` which embeds
  `site/out/` (frontend) and `site/out/bin/*` (slim binary archive).
- The `slim` tag triggers `site/site_slim.go` which provides a
  no-op site filesystem.
- Version is baked in via ldflags:
  `-X github.com/coder/coder/v2/buildinfo.tag=$version`
- The fat binary is built from `./enterprise/cmd/coder` (includes
  enterprise features). Use `--agpl` flag for AGPL-only builds.
- Build flags `-s -w` strip debug info (default unless `--debug`).

## Deploying via SSM

To deploy to an EC2 instance without direct SSH:

```bash
# Upload to S3
aws s3 cp build/coder_${VERSION}_linux_arm64 \
  s3://your-bucket/tmp/coder_${VERSION}

# Generate presigned URL (valid 1 hour)
PRESIGNED=$(aws s3 presign s3://your-bucket/tmp/coder_${VERSION} \
  --expires-in 3600)

# Download to instance via SSM
aws ssm send-command \
  --instance-ids i-xxxxxxxxxxxx \
  --document-name "AWS-RunShellScript" \
  --parameters "commands=[\"curl -sS -o /tmp/coder_${VERSION} '${PRESIGNED}' && chmod +x /tmp/coder_${VERSION} && /tmp/coder_${VERSION} version\"]"
```

Then swap the binary and restart the coder service on the instance.
