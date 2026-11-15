#!/data/data/com.termux/files/usr/bin/bash
# install.sh — instalador de termux-forjanet para Termux.
#
# Qué hace:
#   1. Detecta la arquitectura real del dispositivo con `uname -m`.
#   2. Elige un PERFIL (alto/medio/bajo/patata) y unas flags de compilación
#      adecuadas a esa arquitectura y, de forma aproximada, a la RAM del equipo.
#   3. Compila termux-forjanet NATIVAMENTE, con el propio toolchain de Go instalado
#      en Termux para esa arquitectura (nada se cross-compila desde un PC: así
#      evitamos necesitar el NDK de Android, que Go exige para armv7/32 bits).
#   4. Deja el binario en $PREFIX/bin/termux-forjanet.
#
# Uso:
#   pkg install -y golang git
#   bash scripts/install.sh
set -euo pipefail

echo "== termux-forjanet: instalador consciente de arquitectura =="

ARCH_RAW="$(uname -m)"
echo "Arquitectura detectada (uname -m): ${ARCH_RAW}"

# --- 1. Mapeo de arquitectura -> GOARCH/GOARM y perfil ---------------------
case "${ARCH_RAW}" in
  aarch64 | arm64)
    GOARCH="arm64"
    GOARM=""
    PROFILE="alto"
    GOGC_DEFAULT="150"
    ;;
  armv7l | armv8l | arm)
    GOARCH="arm"
    GOARM="7"
    PROFILE="medio"
    GOGC_DEFAULT="75"
    ;;
  i686 | x86)
    GOARCH="386"
    GOARM=""
    PROFILE="bajo"
    GOGC_DEFAULT="75"
    ;;
  x86_64)
    GOARCH="amd64"
    GOARM=""
    PROFILE="alto"
    GOGC_DEFAULT="150"
    ;;
  *)
    echo "Arquitectura no reconocida (${ARCH_RAW}); usando perfil mínimo 'patata'."
    GOARCH="arm"
    GOARM="5"
    PROFILE="patata"
    GOGC_DEFAULT="50"
    ;;
esac

# --- 2. RAM disponible, para afinar aún más el perfil -----------------------
MEM_KB="$(awk '/MemTotal/{print $2}' /proc/meminfo 2>/dev/null || echo 0)"
MEM_MB=$(( MEM_KB / 1024 ))
echo "RAM total aproximada: ${MEM_MB} MB"

if [ "${MEM_MB}" -gt 0 ] && [ "${MEM_MB}" -lt 2048 ] && [ "${PROFILE}" != "patata" ]; then
  echo "RAM por debajo de 2 GB: forzando perfil 'bajo' aunque la CPU sea potente."
  PROFILE="bajo"
  GOGC_DEFAULT="60"
fi

echo "Perfil elegido: ${PROFILE}  (GOARCH=${GOARCH} GOARM=${GOARM:-n/a})"

# --- 3. Dependencias de compilación -----------------------------------------
if ! command -v go >/dev/null 2>&1; then
  echo "Instalando toolchain de Go nativo para esta arquitectura..."
  pkg install -y golang
fi

# --- 4. Compilación nativa (sin cross-compiling, sin NDK) -------------------
cd "$(dirname "$0")/.."
export CGO_ENABLED=0
export GOARCH GOARM

LDFLAGS="-s -w -X termux-forjanet/internal/config.BuildArch=${GOARCH}${GOARM}"

echo "Compilando (go build, nativo para ${ARCH_RAW})..."
go build -ldflags "${LDFLAGS}" -o "${PREFIX}/bin/termux-forjanet" ./cmd/termux-forjanet

# --- 5. Config por defecto, acorde al perfil --------------------------------
CONF_DIR="${PREFIX}/etc/termux-forjanet"
mkdir -p "${CONF_DIR}"
cat > "${CONF_DIR}/profile.env" <<EOF
# Generado automáticamente por install.sh — perfil detectado: ${PROFILE}
export TERMUXFORJANET_PROFILE="${PROFILE}"
export GOGC="${GOGC_DEFAULT}"
EOF

echo
echo "Listo. Binario instalado en: ${PREFIX}/bin/termux-forjanet"
echo "Perfil activo guardado en:   ${CONF_DIR}/profile.env"
echo
echo "Prueba rápida:"
echo "  source ${CONF_DIR}/profile.env && termux-forjanet doctor"
