# Contributing to Clock

¡Gracias por tu interés en contribuir a Clock! Este proyecto es un reloj digital minimalista para Windows implementado en Go con API Win32 nativa.

## Código de Conducta

Al participar en este proyecto, aceptas mantener un ambiente respetuoso y colaborativo. No se tolera acoso ni conductas inapropiadas.

## ¿Cómo contribuir?

### Reportar Bugs

1. Verifica que el bug no haya sido reportado antes en [Issues](https://github.com/Angelbar/Clock/issues)
2. Usa la plantilla de [Bug Report](.github/ISSUE_TEMPLATE/bug_report.md)
3. Incluye:
   - Versión de Windows (10/11)
   - Comportamiento esperado vs real
   - Pasos para reproducir
   - Captura de pantalla si aplica

### Solicitar Features

1. Revisa las [Issues](https://github.com/Angelbar/Clock/issues) existentes
2. Usa la plantilla de [Feature Request](.github/ISSUE_TEMPLATE/feature_request.md)
3. Describe el problema que resuelve, no solo la solución propuesta

### Enviar Pull Requests

1. **Fork** el repositorio
2. Crea una rama: `git checkout -b feature/mi-mejora`
3. **Sigue el estilo del código existente**
4. **Asegura que compila**: `go build -o Clock.exe`
5. **Prueba manualmente** la aplicación
6. Commit con mensajes claros (inglés o español)
7. Push a tu fork y crea un PR

## Estilo de código

- El proyecto es **un solo archivo** (`main.go`) para simplicidad
- Usa nombres de funciones en camelCase
- Las constantes Win32 van en `SCREAMING_SNAKE_CASE`
- Comentarios en español (idioma del proyecto)
- Las funciones de callback Win32 llevan prefijo `wnd` o sufijo `Proc`
- Mantén las funciones bajo 80 líneas cuando sea posible

## Compilación

```bash
# Desarrollo (con consola para ver errores)
go build -o Clock.exe

# Release (sin consola, optimizado)
go build -ldflags="-H=windowsgui -s -w" -o Clock.exe
```

## Arquitectura

```
main.go (~470 líneas)
├── Win32 API bindings (syscall.NewLazyDLL)
├── Config (JSON en %APPDATA%/Clock/)
├── wndProc() → mensajes Win32
├── drawClock() → GDI rendering
└── helpers (fuentes, colores, layout)
```

### Principios de diseño

- **Sin CGO**: Todo el código debe compilar con `CGO_ENABLED=0`
- **Sin dependencias externas**: Solo stdlib + syscall
- **Binario pequeño**: Apunta a < 3MB
- **Windows nativo**: Usa API Win32 directamente, no wrappers
- **Un solo archivo**: Mantén `main.go` como única fuente

## Testing

Actualmente el proyecto no tiene tests automatizados. Las pruebas son manuales:

1. Ejecutar `Clock.exe`
2. Verificar que la ventana aparece centrada
3. Arrastrar, redimensionar, cambiar colores
4. Verificar persistencia (cerrar y abrir de nuevo)
5. Probar toggle 12h/24h
6. Verificar botón de reset

Si agregas tests, usa solo la stdlib (`testing`).

## Estructura del repositorio

```
Clock/
├── main.go           # Aplicación completa
├── go.mod            # Módulo Go
├── PRD.md            # Documento de requisitos
├── README.md         # Documentación principal
├── CONTRIBUTING.md   # Esta guía
├── LICENSE           # MIT
├── .github/
│   └── ISSUE_TEMPLATE/
│       ├── bug_report.md
│       └── feature_request.md
└── docs/
    └── screenshot.png
```

## Preguntas?

Abre un [Issue](https://github.com/Angelbar/Clock/issues) o contacta al mantenedor.

¡Gracias por contribuir! 🕐
