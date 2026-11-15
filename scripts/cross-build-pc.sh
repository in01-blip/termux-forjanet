#!/usr/bin/env bash
# cross-build-pc.sh — compilar desde un PC (Linux/Mac/WSL), NO desde Termux.
#
# Solo puede generar el binario ARM64 (android/arm64) sin dependencias extra,
# porque el propio compilador de Go para "android/arm" (32 bits) exige CGO,
# y CGO para Android exige el NDK de Google instalado localmente.
#
# Si necesitas el binario de 32 bits y no quieres compilarlo en el propio
# teléfono con scripts/install.sh, instala el NDK y exporta:
#   CC=$NDK/toolchains/llvm/prebuilt/<host>/bin/armv7a-linux-androideabi21-clang
# y vuelve a ejecutar este script con CGO_ENABLED=1 GOARCH=arm GOARM=7.
set -euo pipefail
cd "$(dirname "$0")/.."

mkdir -p dist
echo "Compilando android/arm64 (puro Go, sin NDK)..."
CGO_ENABLED=0 GOOS=android GOARCH=arm64 \
  go build -ldflags "-s -w -X termux-forjanet/internal/config.BuildArch=arm64" \
  -o dist/termux-forjanet-arm64 ./cmd/termux-forjanet
echo "OK -> dist/termux-forjanet-arm64"

if [ "${1:-}" = "--with-ndk" ]; then
  : "${ANDROID_NDK_HOME:?Define ANDROID_NDK_HOME apuntando a tu NDK instalado}"
  CC="${ANDROID_NDK_HOME}/toolchains/llvm/prebuilt/linux-x86_64/bin/armv7a-linux-androideabi21-clang"
  echo "Compilando android/arm (32 bits) con NDK en ${CC}..."
  CGO_ENABLED=1 GOOS=android GOARCH=arm GOARM=7 CC="${CC}" \
    go build -ldflags "-s -w -X termux-forjanet/internal/config.BuildArch=arm32" \
    -o dist/termux-forjanet-arm32 ./cmd/termux-forjanet
  echo "OK -> dist/termux-forjanet-arm32"
else
  echo "Binario de 32 bits NO generado (necesita --with-ndk y el NDK instalado)."
  echo "Alternativa recomendada: ejecuta scripts/install.sh dentro del propio Termux del dispositivo de 32 bits."
fi
