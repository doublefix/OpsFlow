#!/bin/bash

kubectl get nri -o custom-columns="NAME:.metadata.name,AGE:.metadata.creationTimestamp,CPU_TOTAL:.spec.resources.cpu.total,CPU_ALLOC:.spec.resources.cpu.allocatable,CPU_USED:.spec.resources.cpu.used,MEM_TOTAL:.spec.resources.memory.total,MEM_ALLOC:.spec.resources.memory.allocatable,MEM_USED:.spec.resources.memory.used,NVIDIA_GPU_TOTAL:.spec.resources['nvidia\.com/gpu'].total,NVIDIA_GPU_ALLOC:.spec.resources['nvidia\.com/gpu'].allocatable,NVIDIA_GPU_USED:.spec.resources['nvidia\.com/gpu'].used"

