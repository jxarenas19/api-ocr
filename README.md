# API OCR - Proyecto de Prueba

Un API simple de prueba construida con **Chi Router** que simula servicios de OCR (reconocimiento óptico de caracteres).

## ⚠️ Proyecto de Prueba
Este es un proyecto de demostración que **NO realiza OCR real**. Todas las respuestas son datos simulados (mock) con fines de testing y desarrollo.

## Tecnologías
- **Go** - Lenguaje de programación
- **Chi Router** - Framework web minimalista
- **Goroutines** - Procesamiento concurrente
- **Context** - Manejo de timeouts y cancelaciones

## Endpoints

### `GET /health`
Endpoint de salud del servicio.
```
curl http://localhost:8080/health
```

### `POST /ocr`
Simula procesamiento OCR de imágenes.

**Request:**
```json
{
  "key": "unique-request-id",
  "url": "https://example.com/image.jpg"
}
```

**Response:**
```json
{
  "key": "unique-request-id",
  "status_code": 200,
  "full_text": "Documento de identificación validez 1234"
}
```

## Uso

```bash
# Ejecutar
go run main.go

# El servidor inicia en puerto 8080
# API listening on :8080
```

## Características
- ✅ Latencia simulada (1-4 segundos)
- ✅ Textos aleatorios de documentos
- ✅ Procesamiento concurrente con goroutines
- ✅ Manejo de timeouts y cancelaciones
- ✅ Códigos de error HTTP apropiados

## Variables de Entorno
- `PORT` - Puerto del servidor (default: 8080)