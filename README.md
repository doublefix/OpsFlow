```bash
docker buildx build --platform linux/amd64 \
  -f container/Dockerfile.gnu \
  -t opsflow:20250227 \
  .

```
