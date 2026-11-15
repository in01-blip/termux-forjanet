# termux-forjanet

Servidor MCP para Termux/Android, en español, con detección automática de
arquitectura (ARM32/ARM64) y perfiles de rendimiento por dispositivo.

Este es un **proyecto independiente**: revisa y se inspira en tres
repositorios existentes, pero su código está escrito desde cero.

## 0. Antes de nada: qué es realista y qué no

- **ARM64 (64 bits): compilación cruzada verificada en este mismo proceso.**
  Go compila `android/arm64` en Go puro (`CGO_ENABLED=0`), sin necesitar el
  NDK de Android. Lo he compilado y comprobado con `file`: es un ELF ARM64
  real, enlazado contra `/system/bin/linker64` (el linker de Android).
- **ARM32 (armeabi-v7a, 32 bits): Go exige CGO para este objetivo**, y CGO
  para Android exige el NDK de Google instalado localmente — algo que este
  entorno de desarrollo no puede descargar (su red solo permite dominios
  como GitHub, PyPI, npm, crates.io, repositorios de Ubuntu; no
  `dl.google.com`). Por eso el binario de 32 bits **se compila dentro del
  propio Termux del dispositivo**, usando el compilador de Go nativo que
  Termux ya instala para esa arquitectura — exactamente como hacen
  `termux-mcp` y `DroidMCP`. `scripts/install.sh` hace esto automáticamente.
- No tengo un dispositivo Android ni Termux reales para probar en caliente
  las llamadas a `termux-api` (batería, portapapeles, etc.); el código está
  escrito contra su interfaz documentada y con manejo explícito de "falta
  termux-api", pero la primera prueba real tiene que hacerla el usuario en
  su teléfono.

## 1. Evaluación de los tres repositorios de referencia

| Repo | Lenguaje | Qué aporta | Qué evitar/adaptar |
|---|---|---|---|
| **TecnicalBot/termux-mcp** | Go | Buen modelo de seguridad: *default-deny*, tiers safe/sensitive/dangerous, exige token para binds no-loopback, auditoría JSONL, instalador de un comando. | Es la referencia de arquitectura que más se parece a lo que pides; `termux-forjanet` sigue su mismo esquema de tiers. |
| **hyhAsma/termux-api-tools-mcp-server** | Python | Cubre SMS/llamadas/contactos vía SSH a Termux. Proyecto declarado como "de aprendizaje" por su propio autor. | Usa SSH con usuario/contraseña en texto plano en el JSON de config y no tiene sandboxing de rutas ni límites de tasa para SMS. No lo replicaría tal cual; si se necesita SMS, mejor el modelo con clave obligatoria y límite de envíos de DroidMCP. |
| **kahz12/DroidMCP** | Go | El más completo y el mejor documentado: 14 servidores modulares, binarios ARM64 sin dependencias, SSRF/path-traversal bloqueados, `mcp-sms` nunca arranca sin clave y limita a 10 SMS/hora. | Es multi-binario (un ejecutable por servidor); `termux-forjanet` lo simplifica a un único binario con módulos activables por variables de entorno, más fácil de gestionar en un solo dispositivo. |

**Decisión de diseño:** `termux-forjanet` toma la estructura de módulos y el
modelo de seguridad de `termux-mcp`/`DroidMCP` (ambos en Go, ambos
default-deny) como inspiración, pero el código es propio y deja fuera el
patrón SSH-con-contraseña-en-config de la tercera opción.

## 2. Por qué Go y no otro "C para Android"

Pediste un lenguaje tan rápido y potente como C pero orientado a Android y
compatible con Termux. Las alternativas reales son:

- **Rust**: rendimiento equivalente a C, pero cross-compilar a Android
  (`aarch64-linux-android`, `armv7-linux-androideabi`) exige instalar y
  configurar el NDK de Google igualmente, con una curva de configuración
  mayor que Go para este caso de uso.
- **Kotlin/Native o Swift**: Kotlin/Native tiene soporte Android limitado y
  más pensado para apps con UI, no para binarios de servidor headless;
  Swift no tiene soporte serio en Android.
- **Go** (lo que usan los dos repos mejor construidos de los tres que
  revisaste): compila a binario estático sin runtime externo, cross-compila
  a ARM64 sin NDK, y cuando Termux lo instala nativamente (`pkg install
  golang`) compila también ARM32 sin ningún paso extra. Es la opción que de
  verdad resuelve "rápido + sin dependencias + compatible con Termux" sin
  añadir un NDK a la ecuación.

Por eso `termux-forjanet` está en Go, no en Python (como sospechabas) ni en Rust.

## 3. Estructura del proyecto

```
cmd/termux-forjanet/   punto de entrada (main.go, subcomando "doctor")
internal/config/       variables de entorno, perfil activo, arquitectura de build
internal/server/       protocolo MCP mínimo sobre stdio (JSON-RPC), sin dependencias
internal/audit/        registro JSONL con redacción de datos sensibles
internal/tools/system/     batería, wifi, info de perfil         [safe]
internal/tools/clipboard/  portapapeles Android vía termux-api    [safe]
internal/tools/files/      lectura/escritura con sandbox de ruta  [sensible]
internal/tools/exec/       shell con lista blanca obligatoria     [peligroso]
scripts/install.sh         instalador consciente de arquitectura (ejecutar EN Termux)
scripts/cross-build-pc.sh  compilación desde PC (solo ARM64 sin NDK)
```

## 4. Modelo de seguridad (igual que termux-mcp/DroidMCP)

- **Default-deny**: los módulos `files` y `exec` están apagados hasta que se
  active `TERMUXFORJANET_ENABLE_SENSIBLE=true` / `TERMUXFORJANET_ENABLE_PELIGROSO=true`.
- **`exec` sin lista blanca = inexistente**: si no defines
  `TERMUXFORJANET_SHELL_ALLOWLIST`, la herramienta `run_command` ni siquiera se
  registra en el servidor.
- **Sandbox de rutas**: `files` resuelve symlinks y rechaza cualquier ruta
  que quede fuera de `TERMUXFORJANET_ROOT`.
- **Auditoría**: cada llamada se registra en JSONL con los campos sensibles
  (token, password, body de SMS...) redactados, si defines
  `TERMUXFORJANET_AUDIT_LOG`.

## 5. Instalación (dentro de Termux, en el dispositivo)

```bash
pkg install -y golang git
git clone <tu-fork-o-carpeta> termux-forjanet && cd termux-forjanet
bash scripts/install.sh
source "$PREFIX/etc/termux-forjanet/profile.env"
termux-forjanet doctor
```

## 6. Conectar un cliente MCP (stdio)

```json
{
  "mcpServers": {
    "termux-forjanet": {
      "command": "termux-forjanet",
      "args": []
    }
  }
}
```

## 7. Próximos pasos sugeridos

- Añadir los módulos de SMS/contactos **solo si de verdad los necesitas**,
  replicando el patrón de DroidMCP (clave obligatoria + límite de envíos),
  nunca el patrón SSH-con-contraseña del tercer repo.
- Cablear `audit.Logger` dentro de cada `Handler` (ahora mismo se abre pero
  no se usa en todas las llamadas — queda como TODO marcado en `main.go`).
- Probar `scripts/install.sh` en un dispositivo ARM32 real y confirmar que
  el perfil detectado y el `GOGC` resultante no generan lag perceptible.
