```bash
curl -X POST http://localhost:8080/ray \
  -H "Content-Type: application/json" \
  -d '{
    "clusterType": "ray",
    "clusterName": "raycluster-kuberay",
    "namespace": "chess-kuberay",
    "machines": [
      {
        "name": "node-1",
        "cpu": "2",
        "memory": "4Gi",
        "ports": [
          {
            "name": "port-1",
            "containerPort": 8080
          }
        ],
        "isHeadNode": true
      },
      {
        "name": "node-2",
        "cpu": "2",
        "memory": "4Gi",
        "ports": [
          {
            "name": "port-2",
            "containerPort": 8081
          }
        ]
      }
    ]
  }'
```
