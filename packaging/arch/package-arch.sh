#!/usr/bin/env bash
# ==============================================================================
# packaging/arch/package-arch.sh - Arch Linux Package (.pkg.tar.zst & .tar.gz) Builder
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
VERSION="${1:-0.1.0}"
OUTPUT_DIR="${2:-$ROOT_DIR/dist/arch}"

echo "==> [arch] Building Arch Linux release package for ChessNova v${VERSION}..."

mkdir -p "$OUTPUT_DIR"
WORK_DIR=$(mktemp -d -t chessnova-arch-build-XXXXXX)
trap 'rm -rf "$WORK_DIR"' EXIT

# Locate compiled binary
BINARY_PATH=""
CANDIDATE_PATHS=(
    "$ROOT_DIR/apps/desktop/src-tauri/target/release/chessnova"
    "$ROOT_DIR/apps/desktop/src-tauri/target/release/ChessNova"
    "$ROOT_DIR/apps/desktop/src-tauri/target/debug/chessnova"
    "$ROOT_DIR/apps/desktop/src-tauri/target/debug/ChessNova"
    "$ROOT_DIR/apps/desktop/src-tauri/target/release/bundle/deb"
)

for p in "${CANDIDATE_PATHS[@]}"; do
    if [ -f "$p" ]; then
        BINARY_PATH="$p"
        break
    fi
done

if [ -z "$BINARY_PATH" ] && [ -d "${CANDIDATE_PATHS[4]}" ]; then
    DEB_FILE=$(find "${CANDIDATE_PATHS[4]}" -name "*.deb" | head -n 1)
    if [ -n "$DEB_FILE" ]; then
        echo "==> [arch] Extracting binary from built deb bundle: $DEB_FILE"
        dpkg-deb -x "$DEB_FILE" "$WORK_DIR/deb_extract" || ar x "$DEB_FILE" --output="$WORK_DIR/deb_extract"
        BINARY_PATH=$(find "$WORK_DIR/deb_extract/usr/bin" -type f | head -n 1)
    fi
fi

# If binary not found, attempt to build via desktop release if requested
if [ -z "$BINARY_PATH" ] || [ ! -f "$BINARY_PATH" ]; then
    echo "==> [arch] No existing release binary found. Building desktop via npm/cargo..."
    cd "$ROOT_DIR/apps/desktop"
    npm run build
    BINARY_PATH="$ROOT_DIR/apps/desktop/src-tauri/target/release/chessnova"
    if [ ! -f "$BINARY_PATH" ]; then
        BINARY_PATH="$ROOT_DIR/apps/desktop/src-tauri/target/release/ChessNova"
    fi
fi

if [ ! -f "$BINARY_PATH" ]; then
    echo "ERROR: Unable to locate compiled ChessNova desktop binary at $BINARY_PATH"
    exit 1
fi

echo "==> [arch] Using compiled binary: $BINARY_PATH"

PKG_DIR="$WORK_DIR/pkg"
mkdir -p "$PKG_DIR/usr/bin"
mkdir -p "$PKG_DIR/usr/share/applications"
mkdir -p "$PKG_DIR/usr/share/icons/hicolor/512x512/apps"
mkdir -p "$PKG_DIR/usr/share/icons/hicolor/128x128/apps"
mkdir -p "$PKG_DIR/usr/share/icons/hicolor/32x32/apps"

# 1. Install binary
cp "$BINARY_PATH" "$PKG_DIR/usr/bin/chessnova"
chmod 755 "$PKG_DIR/usr/bin/chessnova"

# 2. Install desktop entry
cp "$SCRIPT_DIR/chessnova.desktop" "$PKG_DIR/usr/share/applications/chessnova.desktop"
chmod 644 "$PKG_DIR/usr/share/applications/chessnova.desktop"

# 3. Install icons from desktop icons directory
ICONS_DIR="$ROOT_DIR/apps/desktop/src-tauri/icons"
if [ -f "$ICONS_DIR/icon.png" ]; then
    cp "$ICONS_DIR/icon.png" "$PKG_DIR/usr/share/icons/hicolor/512x512/apps/chessnova.png"
fi
if [ -f "$ICONS_DIR/128x128.png" ]; then
    cp "$ICONS_DIR/128x128.png" "$PKG_DIR/usr/share/icons/hicolor/128x128/apps/chessnova.png"
fi
if [ -f "$ICONS_DIR/32x32.png" ]; then
    cp "$ICONS_DIR/32x32.png" "$PKG_DIR/usr/share/icons/hicolor/32x32/apps/chessnova.png"
fi

# Compute installed size in bytes
INSTALLED_SIZE=$(du -sb "$PKG_DIR" | cut -f1)
BUILD_DATE=$(date +%s)

# 4. Generate .PKGINFO metadata conforming to Arch pacman standard
cat <<EOF > "$PKG_DIR/.PKGINFO"
pkgname = chessnova
pkgbase = chessnova
pkgver = ${VERSION}-1
pkgdesc = Modern, distraction-free chess platform and Stockfish 19 Game Review Studio
url = https://github.com/Yogesh-Kumar-Mallik-dev/ChessNova
builddate = ${BUILD_DATE}
packager = Yogesh Kumar Mallik <yogeshkumarmallikdev@gmail.com>
size = ${INSTALLED_SIZE}
arch = x86_64
license = MIT
depend = webkit2gtk-4.1
depend = gtk3
depend = glib2
depend = openssl
depend = libayatana-appindicator
optdepend = stockfish: external UCI engine analysis
provides = chessnova
conflict = chessnova
EOF

# 5. Pack Arch Linux .pkg.tar.zst package
PKG_FILE="$OUTPUT_DIR/chessnova-${VERSION}-1-x86_64.pkg.tar.zst"
echo "==> [arch] Creating Pacman package: $PKG_FILE"
cd "$PKG_DIR"
# Pacman requires .PKGINFO to be the first file entry
tar --zstd -cf "$PKG_FILE" .PKGINFO usr/

# 6. Also create a portable standalone tarball for manual installs or AUR
TAR_DIR="$WORK_DIR/tarball/chessnova-${VERSION}"
mkdir -p "$TAR_DIR/icons"
cp "$BINARY_PATH" "$TAR_DIR/chessnova"
cp "$SCRIPT_DIR/chessnova.desktop" "$TAR_DIR/"
cp "$PKG_DIR/usr/share/icons/hicolor/512x512/apps/chessnova.png" "$TAR_DIR/icons/512x512.png" || true
cp "$PKG_DIR/usr/share/icons/hicolor/128x128/apps/chessnova.png" "$TAR_DIR/icons/128x128.png" || true
cp "$PKG_DIR/usr/share/icons/hicolor/32x32/apps/chessnova.png" "$TAR_DIR/icons/32x32.png" || true

cat << 'EOF' > "$TAR_DIR/install.sh"
#!/usr/bin/env bash
set -e
PREFIX="${PREFIX:-/usr/local}"
echo "Installing ChessNova to $PREFIX..."
install -Dm755 chessnova "$PREFIX/bin/chessnova"
install -Dm644 chessnova.desktop "$PREFIX/share/applications/chessnova.desktop"
if [ -d "icons" ]; then
    install -Dm644 icons/512x512.png "$PREFIX/share/icons/hicolor/512x512/apps/chessnova.png" 2>/dev/null || true
    install -Dm644 icons/128x128.png "$PREFIX/share/icons/hicolor/128x128/apps/chessnova.png" 2>/dev/null || true
    install -Dm644 icons/32x32.png "$PREFIX/share/icons/hicolor/32x32/apps/chessnova.png" 2>/dev/null || true
fi
echo "ChessNova installed successfully! Run 'chessnova' to launch."
EOF
chmod +x "$TAR_DIR/install.sh"

TAR_FILE="$OUTPUT_DIR/ChessNova_${VERSION}_x86_64.tar.gz"
echo "==> [arch] Creating standalone archive: $TAR_FILE"
cd "$WORK_DIR/tarball"
tar -czf "$TAR_FILE" "chessnova-${VERSION}"

# Copy PKGBUILD into output
cp "$SCRIPT_DIR/PKGBUILD" "$OUTPUT_DIR/PKGBUILD"

echo "✅ [arch] Arch Linux packages generated successfully:"
ls -lh "$OUTPUT_DIR"
