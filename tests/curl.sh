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


curl -X POST http://10.96.98.178:8080/api/v1/rayjob \
     -H "Content-Type: application/json" \
     -d '{
  "job": {
    "name": "deepseek-r1-671b",
    "cmd": "python /home/ray/runcode/vllm_deepseek_r1_671b.py"
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
        },
        {
          "name": "volume-runcode-deepseek-r1-671b",
          "path": "/home/ray/runcode",
          "configMap": {
            "name": "runcode-deepseek-r1-671b",
            "items": [
              {
                "key": "vllm_deepseek_r1_671b.py",
                "path": "vllm_deepseek_r1_671b.py"
              }
            ]
          }
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
      "replicas": 4,
      "minReplicas": 4,
      "maxReplicas": 4
    }
  ]
}'