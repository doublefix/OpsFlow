```bash
docker buildx build --platform linux/amd64 \
  -f container/Dockerfile.gnu \
  -t modco/opsflow:2025.0313.1034 \
  --push \
  .

# Before run
deepseek_r1_pvc_model.yaml
deepseek_r1_cm_runcode.yaml

# After run
deepseek_r1_svc.yaml


# GRPC
protoc \
  --proto_path=/usr/include \
  --proto_path=./pkg/apis/proto \
  --go-grpc_out=./pkg/apis/proto \
  --go_out=./pkg/apis/proto \
  --go-grpc_opt=paths=source_relative \
  pkg/apis/proto/cluster_node.proto



```
