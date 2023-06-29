eth的源码有下面几个包
- downloader 主要用于和网络同步，包含了传统同步方式和快速同步方式
- fetcher 主要用于基于块通知的同步，接收到当我们接收到NewBlockHashesMsg消息得时候，我们只收到了很多Block的hash值。 需要通过hash值来同步区块。
- filter 提供基于RPC的过滤功能，包括实时数据的同步(PendingTx)，和历史的日志查询(Log filter)
- gasprice 提供gas的价格建议， 根据过去几个区块的gasprice，来得到当前的gasprice的建议价格

eth 协议部分源码分析
- 以太坊的网络协议大概流程

fetcher部分的源码分析
- fetch部分源码分析
- 
downloader 部分源码分析
- 节点快速同步算法
- 用来提供下载任务的调度和结果组装 queue.go
- 用来代表对端，提供QoS等功能 peer.go
- 快速同步算法 用来提供Pivot point的 state-root的同步 statesync.go
- 同步的大致流程的分析

filter 部分源码分析
- 提供布隆过滤器的查询和RPC过滤功能