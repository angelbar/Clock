# Reloj Digital Minimalista — PRD

## Problem Statement

Los usuarios necesitan una forma sencilla y discreta de ver la hora actual en su escritorio sin las distracciones de una ventana de aplicación completa del sistema operativo. Las aplicaciones de reloj estándar a menudo carecen de personalización o son demasiado intrusivas.

## Solution

Un reloj digital minimalista que muestra la hora actual, es personalizable en colores, movible, redimensionable, con persistencia de configuración y menú contextual al hover. Se integra de forma no intrusiva en el escritorio del usuario.

**Implementado en Go con API Win32 nativa** — sin dependencias externas, sin CGO, binario pequeño y eficiente.

## User Stories

1.  Como usuario, quiero ver la hora actual en formato HH:MM:SS, para mantenerme al tanto del tiempo de un vistazo.
2.  Como usuario, quiero que la ventana del reloj no tenga bordes ni barra de título y se mantenga siempre al frente, para que se vea minimalista y no se pierda entre otras ventanas.
3.  Como usuario, quiero poder mover el reloj a cualquier parte de mi pantalla, para colocarlo donde me sea más cómodo.
4.  Como usuario, quiero poder cambiar el tamaño del reloj, para ajustar su visibilidad según mis preferencias.
5.  Como usuario, quiero que el tamaño del texto del reloj se ajuste automáticamente al tamaño de la ventana con márgenes mínimos, para que siempre sea legible aprovechando todo el espacio.
6.  Como usuario, quiero poder personalizar el color de fondo y el color del texto del reloj mediante un selector visual, para que coincida con mi tema de escritorio.
7.  Como usuario, quiero un menú que aparezca al pasar el ratón por el borde superior con botones para cerrar, resetear configuración y cambiar colores, sin que ocupe espacio permanente ni tape el reloj.
8.  Como usuario, quiero que la posición, tamaño y colores del reloj se guarden automáticamente al cerrar, para que al abrirlo de nuevo esté exactamente como lo dejé.
9.  Como usuario, quiero un botón de reset que restaure los valores por defecto, para recuperarme si la ventana queda demasiado pequeña o con colores ilegibles.
10. Como usuario, quiero poder cerrar el reloj con la tecla Escape, para ocultarlo rápidamente cuando no lo necesite.
11. Como usuario, quiero poder redimensionar el reloj desde cualquier punto del borde inferior, no solo desde una esquina pequeña, para no perder el control cuando la ventana es muy chica.
12. Como usuario, quiero alternar entre formato 12h (AM/PM) y 24h directamente desde el menú, para adaptarme a diferentes preferencias regionales.

## Implementation Decisions

### Lenguaje y toolkit
- **Lenguaje:** Go 1.26+
- **GUI:** API Win32 nativa vía `syscall` y `golang.org/x/sys/windows`
- **Sin CGO:** Compilación cruzada nativa, sin dependencias de toolchain C
- **Binario:** Autocontenido (~2.2MB), sin DLLs externas

### Ventana
- Estilo `WS_POPUP` para eliminar adornos del sistema operativo
- Estilo extendido `WS_EX_TOPMOST | WS_EX_TOOLWINDOW | WS_EX_NOACTIVATE` para mantenerla siempre al frente sin robar el foco
- Arrastre implementado vía `WM_NCHITTEST` devolviendo `HTCAPTION` (arrastre nativo del sistema)
- Barra inferior completa (12px) para redimensionar, con indicador `◢` en la esquina

### Menú hover
- No hay frames cubrientes. Se usa `WM_MOUSEMOVE` para detectar la coordenada Y del cursor
- Si está dentro de los primeros 30px del borde superior, los botones aparecen dibujados directamente en el `HDC`
- Al salir de esa zona, se ocultan tras 300ms vía `WM_TIMER`
- Cada botón tiene efecto hover visual (rojo para cerrar, gris para reset/paleta)
- Botones: `✕` cerrar, `↺` reset, `🎨` selector colores, `24h`/`12h` toggle

### Fuente y texto
- Fuente: Segoe UI Bold, renderizada con GDI vía `CreateFontIndirectW`
- Tamaño dinámico: **55%** de la altura de la ventana para la hora, **75%** de ese tamaño para AM/PM
- Renderizado con `TextOutW` + `SetTextColor` + modo `TRANSPARENT`
- Texto centrado horizontalmente, alineado al fondo con padding

### Selector de colores
- Diálogo nativo de Windows `CHOOSECOLORW` (comdlg32.dll)
- Selecciona color de fondo y color de texto por separado
- Colores guardados en formato hexadecimal

### Configuración persistente
- Archivo: `%APPDATA%/Clock/config.json`
- Formato JSON con: `bg`, `fg`, `w`, `h`, `x`, `y`, `ampm`
- Se guarda al cerrar la ventana o al cambiar colores
- Si el archivo no existe, usa valores por defecto (#222222/#F31A1A, 380×100)

### Toggle 12h/24h
- Botón en el menú hover que alterna entre formato 12h (`03:04:05 PM`) y 24h (`15:04:05`)
- Persiste el estado en la configuración

### Font scaling
```go
timeSize  = windowHeight * 55 / 100  // mínimo 12
ampmSize  = timeSize * 75 / 100       // mínimo 9
```

### Tamaños mínimos
- Ancho mínimo: 180px
- Alto mínimo: 60px

### Cierre
- Tecla `Escape` o botón `✕`: guarda configuración antes de destruir la ventana

## Controles de la Ventana

| Acción | Método |
|--------|--------|
| **Mover** | Arrastrar desde cualquier parte del área del reloj |
| **Redimensionar** | Arrastrar desde el borde inferior (barra completa) |
| **Cerrar** | Escape o botón ✕ en menú hover |
| **Reset** | Botón ↺ en menú hover |
| **Color** | Botón 🎨 en menú hover |
| **12h/24h** | Botón en menú hover |
| **Menú hover** | Pasar el ratón por el borde superior de la ventana |

## Arquitectura

```
main.go           → Punto de entrada, lazo de mensajes Win32
├── config.go     → Carga/guarda configuración JSON
├── wndProc()     → Proc de ventana (mensajes Win32)
├── drawClock()   → Renderizado GDI (fondo, texto, botones)
└── helpers       → Utilidades (color, fuentes, layout botones)
```

Todo en un solo archivo `main.go` (~470 líneas) para simplicidad.

## Out of Scope

- Funcionalidades de alarma
- Personalización de formato de fecha/hora más allá de 12h/24h
- Múltiples zonas horarias
- Animaciones o transiciones
- Soporte para temas complejos más allá de colores básicos
- Soporte para otros SO (Windows-only por naturaleza)

## Build

```bash
# 1. Generar versioninfo.syso (metadatos)
# Crea versioninfo.json, luego:
goversioninfo -o resource.syso versioninfo.json

# 2. Compilar
go build -ldflags="-H=windowsgui -s -w" -o dist/Clock.exe .
```

## Distribución

- Compañía: **eData101** (https://edata101.com)
- Código fuente: `~/Desa/Clock/main.go`
- Binario compilado: `~/Desa/Clock/dist/Clock.exe`
- Repo: https://github.com/angelbar/Clock
- No requiere instalación: descargar y ejecutar
- Config persistente en: `%APPDATA%/Clock/config.json`

## Further Notes

La configuración del usuario persiste en `%APPDATA%/Clock/config.json`. Las decisiones de interacción (hover threshold 30px, hide delay 300ms, fuente 55%) fueron refinadas iterativamente con el usuario para lograr un balance entre minimalismo y usabilidad.

La aplicación se ejecuta sin consola (`-H=windowsgui`) y sin robar el foco (`WS_EX_NOACTIVATE`), ideal para tenerla siempre visible en el escritorio sin interrumpir el flujo de trabajo.
