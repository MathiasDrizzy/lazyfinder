CONTEXTO
lazyfinder es un file manager TUI en Go (macOS/Linux) con: columnas, sidebar (--sidebar), preview con Space (Markdown, código, zip, imágenes), open-with configurable (openers/argv en config.json), copiar/cortar/pegar, papelera + undo, búsqueda (/ con filtros kind:), tabs, favoritos, recientes, pestaña Git, plugin de herdr, shell-init, 14 temas, 8 idiomas, mouse y keybindings reasignables. Docs en docs/ (SPEC.md, adr/, search.md, configuration.md, keybindings.md). Lee README.md, SECURITY.md, CONTRIBUTING.md, docs/SPEC.md y docs/adr antes de tocar nada.

REGLAS DE EJECUCIÓN
1. Trabaja en una rama nueva: audit/full-sweep. Haz commits pequeños y atómicos (uno por hallazgo), con mensaje claro.
2. Cada bug corregido lleva un test de regresión que falle antes del fix y pase después.
3. No des por cerrada ninguna fase sin ejecutar: go build ./... && go vet ./... && go test -race ./... && golangci-lint run (usa .golangci.yml del repo). Todo debe quedar en verde.
4. Si un fix revela otro bug, corrígelo también en el mismo ciclo. Itera hasta que una pasada completa no encuentre nada nuevo.
5. Mantén un archivo AUDIT.md (temporal, en la rama) con: hallazgo, severidad, archivo:línea, fix, commit. Al final debe tener 0 pendientes.

FASE 0 — Diagnóstico base
- /doctor: corrige todo lo que reporte sobre configuración, skills y costos.
- Corre la línea base: go build, go vet, go test -race -count=1 ./..., golangci-lint, y go test -cover ./... Anota la cobertura inicial por paquete.
- govulncheck ./... y go list -m -u all: corrige dependencias vulnerables o desactualizadas, y verifica que go.mod, go.sum y la versión de Go sean consistentes.

FASE 1 — Errores y bugs en la TUI
Revisa internal/ y cmd/lazyfinder buscando, como mínimo:
- Panics: índices fuera de rango con listas vacías, selección tras borrar/filtrar, nil derefs, type assertions sin comma-ok, división por cero en layout.
- Estado inconsistente del modelo: cursor/scroll tras resize, cambio de tab, búsqueda, undo, pegado, cambio de carpeta, historial [ ].
- Terminales pequeñas: menos de 60x16 y sidebar de menos de 24 columnas; resize durante preview, menús y diálogos.
- Unicode: nombres con CJK, emoji, combinados y ancho doble; truncado y alineación en columnas y breadcrumb; nombres con saltos de línea, tabs o secuencias ANSI.
- Concurrencia: goroutines filtradas, data races, cancelación de búsquedas (Tab para subcarpetas/disco), watchers de live follow, comandos Bubble Tea que llegan después de cerrar un modelo, y cierres de canales.
- Errores ignorados (_ =), errores tragados, mensajes de error poco claros y falta de feedback al usuario.
- Operaciones de archivos: copiar/pegar sobre sí mismo, carpeta dentro de sí misma, symlinks (ciclos y rotos), permisos denegados, archivos enormes, FIFOs, devices, sockets, nombres duplicados, disco lleno, rename con colisión, cortar entre volúmenes distintos, undo tras cambios externos.
- Terminal: restaurar siempre el modo normal (alt screen, cursor, mouse) al salir, por panic o por señal (SIGINT, SIGTERM, SIGHUP, SIGTSTP/SIGCONT).
- Plataforma: diferencias macOS vs Linux (Trash, open, Quick Look, portapapeles). Linux no se ha usado de forma interactiva, así que revisa ese camino con más cuidado.
- Corrige cada hallazgo. Después usa /code-review --fix sobre los cambios y revisa a mano lo que aplique.

FASE 2 — Seguridad
- /security-review sobre la rama y corrige todo.
- Revisión manual dirigida de:
  * Ejecución de comandos: openers/argv y custom commands. Nunca pasar por shell, usar siempre exec con argv separado, validar placeholders {file} contra inyección de argumentos (nombres que empiecen con "-", usar "--"), y nombres con espacios o metacaracteres.
  * Path traversal y symlinks: escapes del directorio esperado, TOCTOU entre check y uso, extracción o listado de zip (zip-slip, zip bombs, límites de tamaño y de entradas).
  * Preview: límites de lectura y de memoria, archivos binarios, Markdown o código con secuencias de escape de terminal (OSC, CSI, títulos de ventana, OSC 52 clipboard, protocolos gráficos kitty/Ghostty). Sanitiza todo contenido no confiable antes de renderizarlo.
  * Nombres de archivo hostiles mostrados en UI, breadcrumb, mensajes de error y la barra de estado.
  * Config: $LAZYFINDER_CONFIG, JSON malformado o gigante, permisos del archivo de config, y escritura atómica (tmp + rename) con permisos 0600/0644 correctos.
  * Papelera y undo: que no se pueda borrar o sobrescribir de forma permanente, que el undo no pueda restaurar fuera de su ruta original, y condiciones de carrera.
  * Git: invocación de git con argumentos seguros, repos con hooks o config maliciosos (core.fsmonitor, core.pager, etc.) al solo listar o hacer diff, y GIT_* del entorno.
  * Plugin herdr, scripts/ y --shell-init: que la salida de shell-init sea segura y esté bien citada, y que scripts/herdr-build.sh use set -euo pipefail y cite las variables.
  * Release y supply chain: .goreleaser.yaml, workflows de .github (permisos mínimos de GITHUB_TOKEN, acciones fijadas por SHA, sin secretos expuestos), checksums, SBOM y atestaciones.
  * Credenciales o rutas personales filtradas en el repo o en el historial (revisa testdata/tuiqa y assets).
- Verifica que SECURITY.md sea correcto y que las instrucciones de verificación de descargas funcionen.

FASE 3 — Auditoría de calidad del proyecto
- Compara el código con docs/SPEC.md y docs/adr: cada requisito debe estar implementado y probado, o el spec debe corregirse si quedó obsoleto.
- Verifica que README, docs/keybindings.md, docs/configuration.md, docs/search.md, la página man (docs/lazyfinder.1) y --help coincidan con el comportamiento real: teclas, flags, defaults y --print-default-config.
- Revisa que las completions (zsh, bash, fish) funcionen.
- Mide la cobertura. Sube a un mínimo razonable los paquetes críticos (operaciones de archivos, undo, config, búsqueda, sanitización) con tests table-driven, y agrega fuzz tests (go test -fuzz) para parsers y entradas externas: config, filtros de búsqueda, nombres, zip.
- Revisa rendimiento: directorios con 100k+ archivos, búsqueda en disco completo, rendering, asignaciones innecesarias, y fugas de memoria o goroutines. Usa pprof o benchmarks cuando sea útil.
- Accesibilidad y UX: contraste de los 14 temas, NO_COLOR, TERM=dumb, terminales sin Nerd Font (icons emoji/ascii/off), todas las claves de i18n presentes en los 8 idiomas, sin textos sin traducir.
- Limpia código muerto, duplicación, comentarios obsoletos, y lint warnings. Nada de nolint sin justificación.
- Revisa los tests de TUI en testdata/tuiqa: que sean deterministas, sin flakiness (corre go test -race -count=20 en los paquetes TUI).

FASE 4 — Mejoras y nuevas funciones para la TUI
Evalúa qué le falta frente a yazi, ranger, nnn, lf y Finder, y propón con justificación. Implementa las que sean de bajo riesgo y alto valor, con tests y docs. Candidatas a evaluar:
- Selección múltiple y operaciones en lote, con barra de progreso y cancelación para copias grandes.
- Paleta de comandos (fuzzy) y búsqueda fuzzy rápida.
- Bookmarks por tecla, y marcas tipo vim.
- Vista previa de más formatos (PDF, CSV/JSON con formato, hex dump para binarios) con límites de seguridad.
- Confirmaciones claras para acciones destructivas y mensajes de error accionables.
- Mejor undo/redo con historial visible.
- Información de archivo (tamaño de carpetas calculado en async, permisos, propietario) y chmod desde la UI.
- Copiar ruta/nombre al portapapeles de forma segura.
- Mejoras de accesibilidad y modo alto contraste.
- Soporte Linux completo (Trash XDG) con tests.
Lo que no implementes ahora, descártalo con motivo en docs/adr (decisión explícita), nunca como "pendiente".

FASE 5 — Verificación final y cierre
- Repite las fases 1 a 3 en una segunda pasada completa. Continúa iterando hasta que una pasada entera no produzca ningún hallazgo nuevo.
- Corre /code-review --fix y /security-review por última vez sobre el diff completo de la rama y resuelve todo.
- Usa /debug para revisar los logs de la sesión y diagnosticar cualquier problema en tiempo de ejecución que haya aparecido durante las pruebas; corrígelo.
- Ejecuta la TUI de verdad (con pty o los escenarios de testdata/tuiqa) en tamaños 24x10, 60x16, 120x40 y 300x80, probando cada tecla documentada.
- Actualiza CHANGELOG.md (sección Unreleased) con todo lo corregido y agregado, separando Security, Fixed y Added.
- Usa /insights al final para generar el informe de la sesión y resume lo aprendido.

CRITERIO DE TERMINADO (todos obligatorios)
[ ] go build, go vet, golangci-lint y govulncheck sin errores ni warnings.
[ ] go test -race -count=1 ./... en verde, y estable con -count=20 en la TUI.
[ ] /security-review y /code-review sin hallazgos abiertos.
[ ] AUDIT.md con 0 pendientes, y cada fix con test de regresión.
[ ] Docs, man page, completions y --help consistentes con el código.
[ ] CHANGELOG actualizado.
[ ] Entrega un reporte final: hallazgos por severidad, qué se corrigió, qué se agregó, qué se descartó con ADR y la cobertura antes y después. Luego deja la rama lista para PR, pero NO hagas merge ni push a main sin mi confirmación.