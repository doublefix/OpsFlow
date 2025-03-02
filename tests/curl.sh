curl -X POST http://10.96.40.83:8080/api/v1/raycluster \
  -H "Content-Type: application/json" \
  -d '{
    "clusterType": "ray",
    "clusterName": "raycluster-kuberay",
    "rayVersion":"2.41.0",
    "rayImage":"rayproject/ray:2.41.0-vllm",
    "namespace": "idp-kuberay",
    "machines": [
      {
        "machineType": "single",
        "name": "ray-head",
        "cpu": "12",
        "memory": "142Gi",
        "customResources": {
          "nvidia.com/gpu": "10"
        },
        "volumes": [
          {
            "name": "volume-deepseek-r1",
            "source": "nfs-pvc-model-deepseek-r1-671b",
            "path": "/mnt/data/models/DeepSeek-R1"
          }
        ],
        "ports": [
          {
            "name": "gcs-server",
            "containerPort": 6379
          },
          {
            "name": "dashboard",
            "containerPort": 8265
          },
          {
            "name": "client",
            "containerPort": 10001
          }
        ],
        "isHeadNode": true
      },
      {
        "machineType": "group",
        "name": "ray-worker",
        "cpu": "12",
        "memory": "142Gi",
        "customResources": {
          "nvidia.com/gpu": "10"
        },
        "volumes": [
          {
            "name": "volume-deepseek-r1",
            "source": "nfs-pvc-model-deepseek-r1-671b",
            "path": "/mnt/data/models/DeepSeek-R1"
          }
        ],
        "groupName": "workergroup",
        "replicas": 4,
        "minReplicas": 4,
        "maxReplicas": 4
      }
    ]
  }'


curl -X POST http://10.96.85.6:8080/api/v1/rayjob \
     -H "Content-Type: application/json" \
     -d '{
  "job": {
    "name": "deepseek-r1-671b",
    "cmd": "vllm serve /mnt/data/models/DeepSeek-R1   --tensor-parallel-size 8    --pipeline-parallel-size 6   --swap-space 32   --trust-remote-code"
  },
  "namespace": "idp-kuberay",
  "rayVersion":"2.43.0",
  "rayImage":"rayproject/ray:2.43.0-vllm_0.7.3",
  "machines": [
    {
      "machineType": "single",
      "name": "ray-head",
      "cpu": "32",
      "memory": "512Gi",
      "customResources": {
        "nvidia.com/gpu": "10"
      },
      "volumes": [
        {
          "name": "volume-deepseek-r1",
          "source": "nfs-pvc-model-deepseek-r1-671b",
          "path": "/mnt/data/models/DeepSeek-R1"
        }
      ],
      "ports": [
        {
          "name": "gcs-server",
          "containerPort": 6379
        },
        {
          "name": "dashboard",
          "containerPort": 8265
        },
        {
          "name": "client",
          "containerPort": 10001
        }
      ],
      "isHeadNode": true
    },
    {
      "machineType": "group",
      "name": "ray-worker",
      "cpu": "32",
      "memory": "512Gi",
      "customResources": {
        "nvidia.com/gpu": "10"
      },
      "volumes": [
        {
          "name": "volume-deepseek-r1",
          "source": "nfs-pvc-model-deepseek-r1-671b",
          "path": "/mnt/data/models/DeepSeek-R1"
        }
      ],
      "groupName": "workergroup",
      "replicas": 3,
      "minReplicas": 3,
      "maxReplicas": 3
    }
  ]
}'