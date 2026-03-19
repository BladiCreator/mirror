# mirror

Herramienta CLI `mirror` para leer archivos `.mrr` y generar código usando plugins.

## instalación

```sh
go install github.com/mirror/mirror/cmd/mirror@latest
```

## uso

```sh
mirror [--watch] [--verbose] [--plugins-dir dir] [--output-dir dir] archivo.mrr|archivo.yml
```

## formato .mrr

- secciones obligatorias: `plugin`, `paths`, `schemas`
- importación de otros `.mrr` con rutas en `schemas`
- `paths` define extensión, plugins y opciones (`f::`, `suffix`, `format`)

## plugin interno

- `go_mrr_parser`
- `dart_mrr_parser`

## protocolo de plugin externo

Entrada JSON en stdin:

```json
{ "schemas": [...], "output_config": {"path":"...","suffix":"...","format":"..."}}
```

Salida JSON en stdout:

```json
{ "files": [{"path":"...","content":"..."}] }
```
