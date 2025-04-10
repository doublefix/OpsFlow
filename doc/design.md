## vllmOnRayAutoJob

通过参数自动构建运行代码中的张量、管道、交换内存与模型路径。

## Realtime node resource info

1. 基本情况：

- go 程序开启多个副本运行
- k8s 集群，多节点
- crd 资源，名称叫 nri。能够存储集群多个节点的基本信息资源信息
- redis-cluster 高可用集群

两个主要逻辑，把节点的资源信息统计添加更新到 crd, 删除已经下线的节点资源 crd。当 crd 资源发生变动的时候调用 rpc 接口去通知 manager, 两个主要逻辑一是根据 watch 实时更新，另外是定时去统计资源信息。

2. 功能实现

- 基于 redis-cluster 高可用集群和多副本程序实现高可用分布式定时任务，部分任务直接在分配到的程序上执行，部分任务添加到队列分布式执行
- 基于 redis-cluster 高可用集群和多副本程序实现监控队列分布式执行任务

分布式定时任务详解：每个定时任务开一个线程，然后放到 redis-cluster 中做个锁，定时任务根据锁执行。为什么使用 crd 存储节点信息，直接使用 node 信息不是更好吗，其实是未来方便对比上一个版本的资源信息和变动后发生了变化需要把变动通知给 manager,另外资源单位的统一比如 cpu 使用 m 内存使用 Mi,还有就是支持各种的自定义资源信息比如 gpu 等。程序会根据 CRD 资源的变化调用多种 rpc 接口，有更新有添加有删除，而且还有定时的心跳。是当 crd 资源发生变动的时候就去触发调用 rpc 接口的逻辑，crd 资源发生变动要么是根据定时要么根据 k8s 的 watch

3. crd 锁

使用乐观锁+重试机制，实现群多个节点的基本信息资源信息 crd 的更新。

```mermaid
sequenceDiagram
    participant GoApp1 as Go 副本1
    participant GoApp2 as Go 副本2
    participant Redis as Redis-Cluster
    participant K8s as K8s Node
    participant CRD as CRD(NRI)
    participant Manager as NodeManager RPC

    %% 定时任务或 Watch 触发
    par 定时任务触发（副本1）
        GoApp1->>Redis: 请求分布式锁（定时任务）
        Redis-->>GoApp1: 获得锁
        GoApp1->>K8s: 拉取节点资源
    and Watch 触发（副本2）
        K8s-->>GoApp2: 节点资源变动事件（watch）
    end

    %% 副本对比并处理资源变更
    GoApp1->>GoApp1: 对比上次采集数据
    GoApp2->>GoApp2: 对比上次采集数据

    alt 副本1发现更新
        GoApp1->>CRD: 添加/更新/删除节点资源
        GoApp1->>Manager: RPC 通知 Manager（添加/更新/删除）
    end
    alt 副本2发现更新
        GoApp2->>CRD: 添加/更新/删除节点资源
        GoApp2->>Manager: RPC 通知 Manager（添加/更新/删除）
    end

    %% 心跳逻辑
    loop 每30秒
        GoApp1->>Manager: RPC 心跳上报
        GoApp2->>Manager: RPC 心跳上报
    end

    %% 分布式任务处理
    alt 副本2分配任务
        GoApp2->>Redis: 写入任务队列（异步任务）
    end
    GoApp1->>Redis: 拉取任务
    Redis-->>GoApp1: 返回任务
    GoApp1->>CRD: 更新任务处理结果

    Note over GoApp1,GoApp2: 多副本协同 + Redis 保证高可用性，<br>所有资源单位标准化（m/Mi）+ 自定义扩展字段支持
```
