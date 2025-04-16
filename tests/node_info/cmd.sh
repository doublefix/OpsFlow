#!/bin/bash

kubectl get nri -o custom-columns="NAME:.metadata.name,AGE:.metadata.creationTimestamp,CPU_TOTAL:.spec.resources.cpu.total,CPU_ALLOC:.spec.resources.cpu.allocatable,CPU_USED:.spec.resources.cpu.used,MEM_TOTAL:.spec.resources.memory.total,MEM_ALLOC:.spec.resources.memory.allocatable,MEM_USED:.spec.resources.memory.used,NVIDIA_GPU_TOTAL:.spec.resources['nvidia\.com/gpu'].total,NVIDIA_GPU_ALLOC:.spec.resources['nvidia\.com/gpu'].allocatable,NVIDIA_GPU_USED:.spec.resources['nvidia\.com/gpu'].used"



SELECT 
    n.id AS node_id,
    n.node_name,
    n.cluster_id,
    n.node_status,
    n.is_unschedulable,
    n.last_heartbeat,
    n.heartbeat_interval,
    n.created_at AS node_created_at,
    n.updated_at AS node_updated_at,
    
    -- CPU资源
    MAX(CASE WHEN rt.resource_name = 'cpu' THEN nr.capacity ELSE NULL END) AS cpu_capacity,
    MAX(CASE WHEN rt.resource_name = 'cpu' THEN nr.allocatable ELSE NULL END) AS cpu_allocatable,
    MAX(CASE WHEN rt.resource_name = 'cpu' THEN nr.unit ELSE NULL END) AS cpu_unit,
    
    -- 内存资源
    MAX(CASE WHEN rt.resource_name = 'memory' THEN nr.capacity ELSE NULL END) AS memory_capacity,
    MAX(CASE WHEN rt.resource_name = 'memory' THEN nr.allocatable ELSE NULL END) AS memory_allocatable,
    MAX(CASE WHEN rt.resource_name = 'memory' THEN nr.unit ELSE NULL END) AS memory_unit,
    
    -- 其他自定义资源
    COALESCE(
        jsonb_agg(
            CASE WHEN rt.resource_name NOT IN ('cpu', 'memory') THEN
                jsonb_build_object(
                    'resource_name', rt.resource_name,
                    'description', rt.description,
                    'capacity', nr.capacity,
                    'allocatable', nr.allocatable,
                    'unit', nr.unit,
                    'properties', nr.properties
                )
            ELSE NULL END
        ) FILTER (WHERE rt.resource_name NOT IN ('cpu', 'memory') AND nr.is_deleted = FALSE),
        '[]'::jsonb
    ) AS custom_resources,
    
    -- 节点标签
    COALESCE(
        jsonb_object_agg(nl.key, nl.value) FILTER (WHERE nl.key IS NOT NULL),
        '{}'::jsonb
    ) AS labels,
    
    -- 节点污点
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'key', nt.key,
                'value', nt.value,
                'effect', nt.effect
            )
        ) FILTER (WHERE nt.key IS NOT NULL),
        '[]'::jsonb
    ) AS taints

FROM 
    node n
LEFT JOIN 
    node_resources nr ON n.id = nr.node_id AND nr.is_deleted = FALSE
LEFT JOIN 
    node_resource_types rt ON nr.resource_type_id = rt.id
LEFT JOIN 
    node_labels nl ON n.id = nl.node_id
LEFT JOIN 
    node_taints nt ON n.id = nt.node_id

WHERE 
    n.is_deleted = FALSE
    AND n.cluster_id = '73f91c59-9ac7-4c92-9a11-dcb3a89c92e6'

GROUP BY 
    n.id, n.node_name, n.cluster_id, n.node_status, n.is_unschedulable, 
    n.last_heartbeat, n.heartbeat_interval, n.created_at, n.updated_at

ORDER BY 
    n.node_name;