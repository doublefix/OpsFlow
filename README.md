```bash
docker buildx build --platform linux/amd64 \
  -f container/Dockerfile.gnu \
  -t modco/opsflow:2025.0303.1742 \
  --push \
  .

```
