## ~~agent.go~~
~~agent 是具体执行挖矿的对象。 它执行的流程就是，接受计算好了的区块头， 计算mixhash和nonce， 把挖矿好的区块头返回。~~

~~构造CpuAgent, 一般情况下不会使用CPU来进行挖矿，一般来说挖矿都是使用的专门的GPU进行挖矿， GPU挖矿的代码不会在这里体现~~

> agent.go在当前版本已被删除，需要重新研究一下代码

## ~~remote_agent~~
~~remote_agent 提供了一套RPC接口，可以实现远程矿工进行采矿的功能。 比如我有一个矿机，矿机内部没有运行以太坊节点，矿机首先从remote_agent获取当前的任务，然后进行挖矿计算，当挖矿完成后，提交计算结果，完成挖矿。~~

> agent.go在当前版本已被删除，需要重新研究一下代码

# 1. 挖矿
下图是以太坊挖矿的主要环节，一环扣一环，缺一不可。

<img src="../img/挖矿主要环节.webp">

以太坊 geth 项目中，关于挖矿逻辑，全部在 miner 包中，3个文件便清晰滴定义挖矿逻辑。所以，后面讲解挖矿业主要集中在这几个文件中。

## 开启挖矿
在以太坊控制台，只需要输入命令：miner.start()即可进开启实时挖矿。 能否能挖新新区，取决于交易池是否有交易和服务器性能。

## 构建新区块
挖矿过程实际就是创建一个符合共识的新区块过程。所以在开启挖矿后，矿工的第一件事情则是集中完成一个新区块的构建，为后续挖矿过程做准备。

## POW寻找Nonce
以太坊本是基于POW工作量证明的共识算法。在这里必须找出一个通过哈希计算，在约定时间内找出一个符合难度值的Nonce。找到相符的Nonce，表示进行了一定的工作量计算。 这是所有POW共识的区块链所必须经历的一个挖矿流程。

## 挖矿成功
并不是一定能成功找出Nonce。特别是在规定的时间内和别人已挖出该高度区块时。只有成功找出Nonce后，则可以大胆的告诉所有人，我已经挖出这个高度的区块了。可以理直气壮地将此区块保留到本地，并广播到整个网络。

## 本次存储新区块
一旦成功挖出新区块，则可以直接将其存储到本地。存储后，可以等待别人对他的认可度。一旦超过50%的节点认可后，大概率上你这个新区块将作为最长链的一部分。

## 网络广播新区块
如何让别人能快速认可你的区块，所以需要在第一时间将区块广播到网络中。抢先于别人一秒，将增大新区块的被认可度。

# 2. 以太坊挖矿架构设计
以太坊 geth 项目中，关于挖矿逻辑，全部在 miner 包中，仅用3个文件便清晰滴定义挖矿逻辑。上一篇文章中，可以看到挖矿的主要环节也就几个。但是如何协商每个环节的处理，却在于架构设计。本文先同大家讲解以太坊关于挖矿的架构设计。

<img src="../img/挖矿架构设计.webp">

> 当前版本有所变化

上图是以太坊 Miner 包中关于挖矿的核心结构。对外部仅仅开放了 Miner 对象，注意：
1. 任何对挖矿的操作均通过 Miner 对象操作
2. 实际挖矿细节全部在 worker 实例中实现
3. 将关键核心数据存储在 environment 中。

外部只需通过调用 Start、Stop 即可完成启动和停止挖矿。当然也可修改一些挖矿参数，如区块头中的可自定义数据 Extra，以及随时调用 SetEtherebase 修改矿工地址 coinbase。

这个的核心全部在 worker 中，worker 是一个工作管理器。订阅了 blockchain 的三个事件，分别监听
1. 新区块事件 chainHeadeCh
2. 新分叉链事件 chainSideCh
3. 新交易事件txCh
4. 以及定义了内部一系列的信号，如 newWork 信号、task信号等。根据信号，执行不同的工作，在各种信号综合作用下协同作业。

在创建worker 时，将在 worker 内开启四个 goroutine 来分别监听不同信号。

<img src="../img/worker信号.webp">

首先是 mainLoop ，将监听 newWork 、tx、chainSide 信号。

- newWork:表示将开始挖采下一个新区块。这个信号在需要重新挖矿时发出，而此信号来自于 newWorkLoop 。当收到newWork信号，矿工将立刻将当前区块作为父区块，来挖下一个区块。

- 当收到来自交易池的tx信号时，如果已经是挖矿中，则可以忽略这些交易。因为交易一样会被矿工从交易池主动拿取。如果尚未开始挖矿，则有机会将交易暂时提交，并更新到state中。

同样，当 blockchain 发送变化（新区块）时，而自己当下的挖掘块中仅有极少的叔块，此时允许不再处理交易，而是直接将此叔块加入，立刻提交当前挖掘块。

newWorkLoop 负责根据不同情况来抉择是否需要终止当前工作，或者开始新一个区块挖掘。有以下几种情况将会终止当前工作，开始新区块挖掘。
1. 接收到 start 信号，表示需要开始挖矿。
2. 接收到 chainHeadCh 新区块信号，表示已经有新区块出现。你所处理的交易或者区块高度都极有可能重复，需要终止当下工作，立即开始新一轮挖矿。
3. timer计时器，默认每三秒检查一次是否有新交易需要处理。如果有则需要重新开始挖矿。以便将加高的交易优先打包到区块中。

在 newWorkLoop 中还有一个辅助信号，resubmitAdjustCh 和 resubmitIntervalCh。运行外部修改timer计时器的时钟。resubmitAdjustCh是根据历史情况重新计算一个合理的间隔时间。而resubmitIntervalCh则允许外部，实时通过 Miner 实例方法 SetRecommitInterval 修改间隔时间。

上面是在控制何时挖矿，而 `taskLoop` 和 `resultLoop` 则不同。`taskLoop` 是在监听任务。任务是指包含了新区块内容的任务，表示可以将此新区块进行PoW计算。一旦接受到新任务，则立即将此任务进行PoW工作量计算，寻找符合要求的Nonce。一旦计算完成，把任务和计算结果作为一项结果数据告知 `resultLoop` 。由resultLoop 完成区块的最后工作，即将计算结构和区块基本数据组合成一个符合共识算法的区块。完成区块最后的数据存储和网络广播。

同时，在挖矿时将当下的挖矿工作的过程信息记录在 environment 中，打包区块时只需要从当前的environment中获取实时数据，或者快照数据。

采用goroutine下使用 channel 作为信号，以太坊 miner 完成一个激活挖矿到最终挖矿成功的整个逻辑。上面的讲解不涉及细节，只是让大家对挖矿有一个整体了解。为后续的各环节详解做准备。                   

# 3. 启动挖矿
挖矿模块只通过 Miner 实例对外提供数据访问。可以通过多种途径开启挖矿服务。程序运行时已经将 Miner 实例化，并进入等待挖矿状态，随时可以启动挖矿。

## 挖矿参数
矿工可以根据矿机的服务器性能，来定制化挖矿参数。下面是一份 geth 关于挖矿的运行时参数清单，全部定义在 `cmd/utils/flags.go` 文件中。

| 参数              | 默认值         | 用途  |
| ----------------- | -------------- | -------------------- |
| --mine            | false          | 是否自动开启挖矿        |
| --miner.threads   | 0              | 挖矿时可用并行PoW计算的协程（轻量级线程）数。<br>兼容过时参数 —minerthreads。 |
| --miner.notify    | 空             | 挖出新块时用于通知远程服务的任意数量的远程服务地址。<br>是用 `,`分割的多个远程服务器地址。<br>如：”http://api.miner.com,http://api2.miner.com“ |
| --miner.noverify  | false          | 是否禁用区块的PoW工作量校验。   |
| --miner.gasprice  | 1000000000 wei | 矿工可接受的交易Gas价格，<br>低于此GasPrice的交易将被拒绝写入交易池和不会被矿工打包到区块。 |
| --miner.gastarget | 8000000 gas    | 动态计算新区块燃料上限（gaslimit）的下限值。<br>兼容过时参数 —targetgaslimit。 |
| --miner.gaslimit  | 8000000 gas    | 动态技术新区块燃料上限的上限值。    |
| --miner.etherbase | 第一个账户     | 用于接收挖矿奖励的账户地址，<br>默认是本地钱包中的第一个账户地址。 |
| --miner.extradata | geth版本号     | 允许矿工自定义写入区块头的额外数据。  |
| --miner.recommit  | 3s             | 重新开始挖掘新区块的时间间隔。<br>将自动放弃进行中的挖矿后，重新开始一次新区块挖矿。 |
| --minerthreads    |                | *已过时*   |
| —targetgaslimit   |                | *已过时*    |
| --gasprice        |                | *已过时*    |

可以通过执行程序 `geth` 来查看参数。

    geth -h |grep "mine"

## 实例化Miner
geth 程序运行时已经将 Miner 实例化，只需等待命令开启挖矿。
```go
// eth/backend.go:233
func New(stack *node.Node, config *ethconfig.Config) (*Ethereum, error) {
    ...
    
    eth.miner = miner.New(eth, &config.Miner, eth.blockchain.Config(), eth.EventMux(), eth.engine, eth.isLocalBlock)
    eth.miner.SetExtra(makeExtraData(config.Miner.ExtraData))
	
    ...
}
```

从上可看出，在实例化 miner 时所用到的配置项只有4项。实例化后，便可通过 API 操作 Miner。

Miner API 分 public 和 private。挖矿属于隐私，不得让其他人任意修改。因此挖矿API全部定义在 Private 中，公共部分只有 Mining()。

## 启动挖矿
geth 运行时默认不开启挖矿。如果用户需要启动挖矿，则可以通过以下几种方式启动挖矿。

### 参数方式自动开启挖矿
使用参数 —mine，可以在启动程序时默认开启挖矿。下面我们用 geth 在开发者模式启动挖矿为例：

    geth --dev --mine

启动后，可以看到默认情况下已开启挖矿。开发者模式下已经挖出了一个高度为1的空块。

当参数加入了 `--mine` 参数表示启用挖矿，此时将根据输入个各项挖矿相关的参数启动挖矿服务。
```go
// cmd/geth/main.go:410
// startNode函数启动系统节点和所有注册的协议，之后解锁任何请求的账户，并启动RPC/IPC接口和矿工。
func startNode(ctx *cli.Context, stack *node.Node, backend ethapi.Backend, isConsole bool) {
	...

	// 如果启用了辅助服务，则开始运行 
    if ctx.Bool(utils.MiningEnabledFlag.Name) || ctx.Bool(utils.DeveloperFlag.Name) {
        // 只有在运行完整的以太坊节点时，挖矿才有意义
        if ctx.String(utils.SyncModeFlag.Name) == "light" {
            utils.Fatalf("轻客户端不支持挖矿")
        }
        ethBackend, ok := backend.(*eth.EthAPIBackend)
        if !ok {
            utils.Fatalf("以太坊服务未运行")
        }
        // 将燃气价格设置为命令行界面上的限制，并开始挖矿
        gasprice := flags.GlobalBig(ctx, utils.MinerGasPriceFlag.Name)
        ethBackend.TxPool().SetGasTip(gasprice)
        if err := ethBackend.StartMining(); err != nil {
            utils.Fatalf("无法启动挖矿：%v", err)
        }
    }
	
	...
}
```

启动 geth 过程是，如果启用挖矿`--mine`或者属于开发者模式`—dev`，则将启动挖矿。

在启动挖矿之前，还需要获取 `—miner.gasprice` 实时应用到交易池中。同时也需要指定将允许使用多少协程来并行参与PoW计算。然后开启挖矿，如果开启挖矿失败则终止程序运行并打印错误信息。

### 控制台命令启动挖矿
在实例化Miner后，已经将 miner 的操作API化。因此我们可以在 geth 的控制台中输入Start命令启动挖矿。

调用API `miner_start` 将使用给定的挖矿计算线程数来开启挖矿。下面表格是调用 API 的几种方式。

| 客户端  | 调用方式                                            |
| ------- | --------------------------------------------------- |
| Go      | `miner.Start(threads *rpc.HexNumber) (bool, error)` |
| Console | `miner.start(number)`                               |
| RPC     | `{"method": "miner_start", "params": [number]}`     |

首先，我们进入 geth 的 JavaScript 控制台，后输入命令miner.start(1)来启动挖矿。

    geth --maxpeers=0 console

启动挖矿后，将开始出新区块。

### RPC API 启动挖矿
因为 API 已支持开启挖矿，如上文所述，可以直接调用 RPC  `{"method": "miner_start", "params": [number]}` 来启动挖矿。实际上在控制台所执行的 `miner.start(1)`，则相对于 `{"method": "miner_start", "params": [1]}`。

如，启动 geth 时开启RPC。
```sh
geth --maxpeer 0 --rpc --rpcapi --rpcport 8080 "miner,admin,eth" console
```

开启后，则可以直接调用API，开启挖矿服务。
```shell
curl -d '{"id":1,"method": "miner_start", "params": [1]}' http://127.0.0.1:8080
```

## 挖矿启动细节
不管何种方式启动挖矿，最终都是通过调用 miner 对象的 Start 方法来启动挖矿。不过在开启挖矿前，geth 还处理了额外内容。

当你通过控制台或者 RPC API 调用启动挖矿命令后，在程序都将引导到方法`func (s *Ethereum) StartMining(threads int) error `。
```go
// StartMining 启动矿工并指定CPU线程数量。
// 如果矿工已经在运行，此方法会调整允许使用的线程数量，并更新交易池所需的最低价格。
func (s *Ethereum) StartMining() error {
	// 如果矿工没有在运行，则进行初始化
	if !s.IsMining() { //❸
		// 将初始价格传播到交易池
		s.lock.RLock()
		price := s.gasPrice
		s.lock.RUnlock()
		s.txPool.SetGasTip(price) //❹

		// 配置本地挖矿地址
		eb, err := s.Etherbase() //❺
		if err != nil {
			log.Error("无法在没有以太坊基地址的情况下开始挖矿", "err", err)
			return fmt.Errorf("缺少以太坊基地址：%v", err)
		}
		var cli *clique.Clique
		if c, ok := s.engine.(*clique.Clique); ok {
			cli = c
		} else if cl, ok := s.engine.(*beacon.Beacon); ok {
			if c, ok := cl.InnerEngine().(*clique.Clique); ok {
				cli = c
			}
		}
		if cli != nil {
			wallet, err := s.accountManager.Find(accounts.Account{Address: eb}) //❻
			if wallet == nil || err != nil {
				log.Error("本地不存在以太坊基地址账户", "err", err)
				return fmt.Errorf("缺少签名器：%v", err)
			}
			cli.Authorize(eb, wallet.SignData) //❼
		}
		// 如果挖矿已经开始，我们可以禁用交易拒绝机制，以加快同步时间。
		s.handler.acceptTxs.Store(true) //❽

		go s.miner.Start() //❾
	}
	return nil
}

// StopMining终止挖矿操作，包括共识引擎级别和区块创建级别。
func (s *Ethereum) StopMining() {
    // 更新共识引擎中的线程数
    type threaded interface {
        SetThreads(threads int)
    }
    if th, ok := s.engine.(threaded); ok { //❶
        th.SetThreads(-1) //❷
    }
    // 停止区块创建本身
    s.miner.Stop()
}
```

> 代码发生了一些变化，`StartMining` 少了❶❷中的代码

在此方法中，首先看挖矿的共识引擎是否支持设置协程数❶，如果支持，将更新此共识引擎参数 ❷。接着，如果已经是在挖矿中，则忽略启动，否则将开启挖矿 ❸。在启动前，需要确定两项配置：交易GasPrice下限❹，和挖矿奖励接收账户（矿工账户地址）❺。

这里对于 clique.Clique 共识引擎（PoA 权限共识），进行了特殊处理，需要从钱包中查找对于挖矿账户❻。在进行挖矿时不再是进行PoW计算，而是使用认可的账户进行区块签名❼即可。

可能由于一些原因，不允许接收网络交易。因此，在挖矿前将允许接收网络交易❽。随即，开始在挖矿账户下开启挖矿❾。此时，已经进入了miner实例的 Start 方法。

```go
// miner/miner.go:168
func (miner *Miner) Start() {
	miner.startCh <- struct{}{}
}

func New(eth Backend, config *Config, chainConfig *params.ChainConfig, mux *event.TypeMux, engine consensus.Engine, isLocalBlock func(header *types.Header) bool) *Miner {
    miner := &Miner{
        mux:     mux,
        eth:     eth,
        engine:  engine,
        exitCh:  make(chan struct{}),
        startCh: make(chan struct{}),
        stopCh:  make(chan struct{}),
        worker:  newWorker(config, chainConfig, engine, eth, mux, isLocalBlock, true),
    }
    miner.wg.Add(1)
    go miner.update()
    return miner
}

// update函数用于跟踪下载器事件。请注意，这是一种一次性的更新循环。
// 它只会进入一次，一旦`Done`或`Failed`被广播，事件将被取消注册并退出循环。
// 这是为了防止外部方通过块进行DOS攻击，并在DOS攻击持续期间停止您的挖矿操作，从而造成重大安全漏洞。
func (miner *Miner) update() {
	defer miner.wg.Done()

	// 订阅下载器的StartEvent、DoneEvent和FailedEvent事件
	events := miner.mux.Subscribe(downloader.StartEvent{}, downloader.DoneEvent{}, downloader.FailedEvent{})
	defer func() {
		if !events.Closed() {
			events.Unsubscribe()
		}
	}()

	shouldStart := false
	canStart := true
	dlEventCh := events.Chan()
	for {
		select {
		case ev := <-dlEventCh:
			if ev == nil {
				// 取消订阅完成，停止监听
				dlEventCh = nil
				continue
			}
			switch ev.Data.(type) {
			case downloader.StartEvent:
				wasMining := miner.Mining()
				miner.worker.stop()
				canStart = false
				if wasMining {
					// 同步完成后恢复挖矿
					shouldStart = true
					log.Info("由于同步而中止挖矿")
				}
				miner.worker.syncing.Store(true)

			case downloader.FailedEvent:
				canStart = true
				if shouldStart {
					miner.worker.start()
				}
				miner.worker.syncing.Store(false)

			case downloader.DoneEvent:
				canStart = true
				if shouldStart {
					miner.worker.start()
				}
				miner.worker.syncing.Store(false)

				// 停止对下载器事件的响应
				events.Unsubscribe()
			}
		case <-miner.startCh:
			if canStart {
				miner.worker.start()
			}
			shouldStart = true
		case <-miner.stopCh:
			shouldStart = false
			miner.worker.stop()
		case <-miner.exitCh:
			miner.worker.close()
			return
		}
	}
}

// miner/worker.go:356
// start将运行状态设置为1，并触发新的工作提交。
func (w *worker) start() {
    w.running.Store(true)
    w.startCh <- struct{}{}
}
```

> miner的构造方法中会调用update方法，`update`中会一直监听各种`miner`信号

存储coinbase 账户后⑩，有可能因为正在同步数据，此时将不允许启动挖矿⑪。如果能够启动挖矿，则立即开启worker 让其开始干活。只需要发送一个开启挖矿信号，worker 将会被自动触发挖矿工作。

## Worker Start 信号
对 worker 发送 start 信号后，该信号将进入 startCh chain中。一旦获得信号，则立即重新开始commit新区块，重新开始干活。
```go
// miner/worker.go:441
// newWorkLoop是一个独立的goroutine，在接收到事件后提交新的密封工作。
func (w *worker) newWorkLoop(recommit time.Duration) {
	...

	for {
        select {
        case <-w.startCh:
            clearPending(w.chain.CurrentBlock().Number.Uint64())
            timestamp = time.Now().Unix()
            commit(commitInterruptNewHead)
        ...
        }
	}
	
	...
}
```

# 4. 以太坊挖矿信号监控
挖矿的核心集中在 worker 中。worker 采用Go语言内置的 chain 跨进程通信方式。在不同工作中，根据信号处理不同工作。

下图是实例化 worker 时启动的四个循环，分别监听不同信号来处理不同任务。
- mainLoop
- taskLoop
- resultLoop
- newWorkLoop

<img src="../img/以太坊挖矿信号监控1.webp">

## 4.1 挖矿工作信号
首先是在 `newWorkLoop` 中监控新挖矿任务。分别监控了三种信号，不管接收到三种中的哪种信号都会触发新一轮挖矿。

但根据信号类型，会告知内部需要重新开启挖矿的原因。如果已经在挖矿中，那么在开启新一轮挖矿前，会将旧工作终止。

如上图，当前的信号类型有：
1. start 信号：
   
start信号属于开启挖矿的信号。在上一篇启动挖矿中，已经有简单介绍。每次在 miner.Start() 时将会触发新挖矿任务。
```go
// newWorkLoop
case <-w.startCh:
    clearPending(w.chain.CurrentBlock().Number.Uint64())
    timestamp = time.Now().Unix()
    commit(commitInterruptNewHead)
```

2. chainHead信号：

节点接收到了新的区块。比如，你原本是是在下一个新区块上挖矿，区块高度是 1000。此时你从网络上收到了一个合法的区块，高度也一样。这样，你就不需要再花力气和别人竞争了，赶快投入到下一个区块的挖矿竞争，才是有意义的。
```go
// newWorkLoop
case head := <-w.chainHeadCh:
    clearPending(head.Block.NumberU64())
    timestamp = time.Now().Unix()
    commit(commitInterruptNewHead)
```

3. timer 信号：

一个时间timer，默认每三秒检查执行一次检查。如果当下正在挖矿中，那么需要检查是否有新交易。如果有新交易，则需要放弃当前交易处理，重新开始一轮挖矿。这样可以使得愿意支付更多手续费的交易能被优先处理。
```go
// newWorkLoop
case <-timer.C:
	// 如果封存正在运行，请定期重新提交新的工作周期以吸引更高价值的交易。
	// 对于待处理的区块，禁用此开销。
	if w.isRunning() && (w.chainConfig.Clique == nil || w.chainConfig.Clique.Period > 0) {
		// 如果没有新的交易到达，则进行短路处理。
		if w.newTxs.Load() == 0 {
			timer.Reset(recommit)
			continue
		}
		commit(commitInterruptResubmit)
	}
```

这三类信号最终都聚集在新一轮挖矿上。那么是如何处理的呢？上图中，挖矿工作在 `mainLoop` 监控中一直等待 newWork信号。此处的三个工作信息，都通过 commit 方法，发送 newWork 信号。
```go
// newWorkLoop
// commit函数中断正在执行的事务并提交一个新的事务。
commit := func(s int32) {
	if interrupt != nil {
		interrupt.Store(s)
	}
	interrupt = new(atomic.Int32)
	select {
	case w.newWorkCh <- &newWorkReq{interrupt: interrupt, timestamp: timestamp}:
	case <-w.exitCh:
		return
	}
	timer.Reset(recommit)
	w.newTxs.Store(0)
}
```

newWork 信号数据中有两个字段（少了noempty）：
1. interrupt：这是一个数字指针，也就不管新work信号还是旧work信号，都能一直跟踪相同的一个全局唯一的任务终止信号值interrupt。 如果是需要终止旧任务，只需要更新信号值atomic.StoreInt32(interrupt, s)后，work 内部便会感知到，从而终止挖矿工作。
2. ~~noempty：是否不能为空块。默认情况下是允许挖空块的，但是明知有交易需要处理，则不允许挖空块（见 timer信号）。~~
3. timestamp：记录的是当前操作系统时间，最终会被用作区块的区块时间戳。

## 动态估算交易处理时长
再回到 timer 信号上。geth 程序启动时，timmer 计时器默认是三秒。但这个时间间隔不是一成不变的，会根据挖矿时长来动态调整。

为什么是默认值是三秒呢？也就是说，系统默认有三秒时间来处理交易，一笔转账交易执行时间是毫秒级的。如果三秒后，仍有新交易未处理完毕，则需要重来，将根据新的交易排序，将愿意支付更多手续费的交易优先处理。

在挖矿timer计时器中，不能固定为三秒钟，这样时间可能太短。采用动态估算的方式也许更加有效。 动态估算的计算公式分两部分：先是计算出一个比例ratio=燃料剩余率，再加工计算出一个新的计时器时间。
```
新时间间隔 = 当前时间间隔 * (1-基准增长率) + 基准增长率 * ( 当前时间间隔/燃料剩余率 )
	        = 当前时间间隔 * (1-0.1) + 0.1 * ( 当前时间间隔/燃料剩余率 )
```

这里的基准增长率是一个常量 0.1 ，通过公式可以看出，是否能有10%的时间延长，取决于燃料剩余率。剩余燃料越多，增长越小，最低是接近90%的负值长。剩余燃料越少，增长越快，最大有近60%的增长。当然也不能一直增长下去，这里有一个15秒的上限值。

动态估算是发生在本次处理到期后，根据一定策略估算出一个新计时器。当正在处理一笔交易时，将检查终止信息值interrupt，如果刚好遇上时间到期，则需要调整计时器❶。以太坊是根据燃料实际执行情况来参与动态估算。首先计算直接等于剩余燃料在区块总燃料中的占比❷。这种计算方式完全是根据单个gas的基础用时，来推导剩余gas可以处理多长时间的交易。

> 这里逻辑发生了变化

```go
// commitWork函数基于父块生成了多个新的封装任务，并将它们提交给封装器。
func (w *worker) commitWork(interrupt *atomic.Int32, timestamp int64) {
	// 如果节点仍在同步中，则终止提交
	if w.syncing.Load() {
		return
	}
	start := time.Now()

	// 如果工作线程正在运行或需要coinbase，则设置coinbase
	var coinbase common.Address
	if w.isRunning() {
		coinbase = w.etherbase()
		if coinbase == (common.Address{}) {
			log.Error("没有etherbase，拒绝挖矿")
			return
		}
	}
	work, err := w.prepareWork(&generateParams{
		timestamp: uint64(timestamp),
		coinbase:  coinbase,
	})
	if err != nil {
		return
	}
	// 从交易池中将待处理交易填充到块中
	err = w.fillTransactions(interrupt, work)
	switch {
	case err == nil:
		// 整个块已填满，在当前间隔大于用户指定间隔的情况下，减少重新提交间隔。
		w.resubmitAdjustCh <- &intervalAdjust{inc: false}

	case errors.Is(err, errBlockInterruptedByRecommit):
		// 如果中断是由于频繁提交而导致的，则通知重新提交循环增加重新提交间隔。
		gaslimit := work.header.GasLimit
		ratio := float64(gaslimit-work.gasPool.Gas()) / float64(gaslimit) //❷
		if ratio < 0.1 {
			ratio = 0.1
		}
		w.resubmitAdjustCh <- &intervalAdjust{ //❸
			ratio: ratio,
			inc:   true,
		}

	case errors.Is(err, errBlockInterruptedByNewHead):
		// 如果块构建被newhead事件中断，则完全丢弃它。提交中断的块会引入不必要的延迟，
		// 并可能导致矿工在前一个块上进行挖矿，从而导致更高的叔块率。
		work.discard()
		return
	}
	// 提交生成的块进行共识封装。
	w.commit(work.copy(), w.fullTaskHook, true, start)

	// 用新的工作替换旧的工作，同时终止任何剩余的预取进程并启动一个新的进程。
	if w.current != nil {
		w.current.discard()
	}
	w.current = work
}
```

在计算出时间增长率后，发送一个自动更新计时器时间的信号 resubmitAdjust。要求按剩余率调整计时器❸。在接收到信号后❹，根据剩余率重新计算计时器时间❺。
```go
// miner/worker.go:476 (newWorkLoop)
case adjust := <-w.resubmitAdjustCh:
	// 根据反馈调整重新提交间隔。
	if adjust.inc {
		before := recommit
		target := float64(recommit.Nanoseconds()) / adjust.ratio
		recommit = recalcRecommit(minRecommit, recommit, target, true)
		log.Trace("增加矿工重新提交间隔", "从", before, "到", recommit)
	} else {
		before := recommit
		recommit = recalcRecommit(minRecommit, recommit, float64(minRecommit.Nanoseconds()), false)
		log.Trace("减少矿工重新提交间隔", "从", before, "到", recommit)
	}
	if w.resubmitHook != nil {
		w.resubmitHook(minRecommit, recommit)
	}
```

重新计算计时器时间间隔后，将会下一个计时器上生效。

同时，还支持矿工通过调用RPC API `{"method": "miner_setRecommitInterval", "params": [interval]}` 来直接修改计时器间隔。调用API后，将会在 worker 中产生信号。
```go
// miner/worker.go:244
// setRecommitInterval函数用于更新矿工封存工作重新提交的间隔。
func (w *worker) setRecommitInterval(interval time.Duration) {
	select {
	case w.resubmitIntervalCh <- interval:
	case <-w.exitCh:
	}
}
```

而在 newWorkLoop 监控中，将监控该信号。发现信号后，立即重置计时器的时间间隔。
```go
case interval := <-w.resubmitIntervalCh:
	// 显式地根据用户调整重新提交间隔。
	if interval < minRecommitInterval {
		log.Warn("清理矿工重新提交间隔", "提供的间隔", interval, "更新后的间隔", minRecommitInterval)
		interval = minRecommitInterval
	}
	log.Info("矿工重新提交间隔更新", "从", minRecommit, "到", interval)
	minRecommit, recommit = interval, interval
	
	if w.resubmitHook != nil {
		w.resubmitHook(minRecommit, recommit)
	}
```

# 以太坊挖矿逻辑流程
上一篇文章中，有介绍是如何发出挖矿工作信号的。当有了挖矿信号后，就可以开始挖矿了。

先回头看看，在讲解挖矿的第一篇文章中，有讲到挖矿流程。这篇文章将讲解挖矿中的各个环节。

<img src="../img/以太坊挖矿逻辑流程1.webp">

## 挖矿代码方法介绍
在继续了解挖矿过程之前，先了解几个miner方法的作用。

- commitTransactions：提交交易到当前挖矿的上下文环境(environment)中。上下文环境中记录了当前挖矿工作信息，如当前挖矿高度、已提交的交易、当前State等信息。
- updateSnapshot：更新 environment 快照。快照中记录了区块内容和区块StateDB信息。相对于把当前 environment 备份到内存中。这个备份对挖矿没什么用途，只是方便外部查看 PendingBlock。
- commitNewWork：重新开始下一个区块的挖矿的第一个环节“构建新区块”。这个是整个挖矿业务处理的一个核心，值得关注。
- commit： 提交新区块工作，发送 PoW 计算信号。这将触发竞争激烈的 PoW 寻找Nonce过程。

## 挖矿工作管理
什么时候可以进行挖矿？如下图所述，挖矿启动工作时由 mainLoop 中根据三个信号来管理。首先是新工作启动信号(newWorkCh)、再是根据新交易信号(txsCh)和最长链链切换信号(chainSideCh)来管理挖矿。

<img src="../img/以太坊挖矿逻辑流程2.webp">

三种信号，三种管理方式。

### 新工作启动信号
这个信号，意思非常明确。一旦收到信号，立即开始挖矿。
```go
// miner/worker.go:514 (mainLoop)
case req := <-w.newWorkCh:
	w.commitWork(req.interrupt, req.timestamp)
```
这个信号的来源，已经在上一篇文章 挖矿工作信号监控中讲解。信号中的各项信息也来源与外部，这里仅仅是忠实地传递意图。

### 新交易信号
在交易池文章中有讲到，交易池在将交易推入交易池后，将向事件订阅者发送 NewTxsEvent。在 miner 中也订阅了此事件。
```go
// miner/worker.go/newWorker()
// Subscribe NewTxsEvent for tx pool
worker.txsSub = eth.TxPool().SubscribeNewTxsEvent(worker.txsCh)
```

当接收到新交易信号时，将根据挖矿状态区别对待。当尚未挖矿(`!w.isRunning()`)，但可以挖矿`w.current != nil`时❶，将会把交易提交到待处理中。
```go
// miner/worker.go/mainLoop
case ev := <-w.txsCh:
	// 如果我们没有在封装区块，将交易应用于待处理状态
	//
	// 注意，接收到的所有交易可能与当前封装区块中已包含的交易不连续。这些交易将被自动消除。
	if !w.isRunning() && w.current != nil { //❶
		// 如果区块已满，则中止
		if gp := w.current.gasPool; gp != nil && gp.Gas() < params.TxGas {
			continue
		}
		txs := make(map[common.Address][]*types.Transaction, len(ev.Txs))
		for _, tx := range ev.Txs { //❷
			acc, _ := types.Sender(w.current.signer, tx)
			txs[acc] = append(txs[acc], tx)
		}
		txset := types.NewTransactionsByPriceAndNonce(w.current.signer, txs, w.current.header.BaseFee) //❸
		tcount := w.current.tcount
		w.commitTransactions(w.current, txset, nil) //❹
		// 只有在添加了任何新交易到待处理区块时才更新快照
		if tcount != w.current.tcount {
			w.updateSnapshot(w.current) //❺
		}
	} else {
		// 特殊情况，如果共识引擎是0周期的clique（开发模式），
		// 在这里提交封装工作，因为所有空提交将被clique拒绝。
		// 当然，禁用了提前封装（空提交）。
		if w.chainConfig.Clique != nil && w.chainConfig.Clique.Period == 0 { //❻
			w.commitWork(nil, time.Now().Unix())
		}
	}
	w.newTxs.Add(int32(len(ev.Txs))) //❼
```

首先，将新交易按发送者分组❷后，根据交易价格和Nonce值排序❸。形成一个有序的交易集后，依次提交每笔交易❹。最新完毕后将最新的执行结果进行快照备份❺。当正处于 PoA挖矿，右允许无间隔出块时❻，则将放弃当前工作，重新开始挖矿。

最后，不管何种情况都对新交易数计加❼。但实际并未使用到数据量，仅仅是充当是否有进行中交易的一个标记。

总得来说，新交易信息并不会干扰挖矿。而仅仅是继续使用当前的挖矿上下文，提交交易。也不用考虑交易是否已处理， 因为当交易重复时，第二次提交将会失败。

### ~~最长链链切换信号~~
~~当一个区块落地成功后，有可能是在另一个分支上。当此分支的挖矿难度大于当前分支时，将发生最长链切换。此时 miner 将需要订阅从信号，以便更新叔块信息。~~

> 当前版本代码中没找到对应部分

## 挖矿流程环节
当开始新区块挖矿时，第一步就是构建区块，打包出包含交易的区块。在打包区块中，是按逻辑顺序依次组装各项信息。如果你对区块内容不清楚，请先查阅文章区块结构。

### 设置新区块基本信息
挖矿是在竞争挖下一个区块，需要把最新高度的区块作为父块来确定新区块的基本信息❶。
```go
// prepareWork 根据给定的参数构建密封任务，可以基于上一个区块头或指定的父区块头。
// 在这个函数中，尚未填充待处理的交易，只返回空的任务。
func (w *worker) prepareWork(genParams *generateParams) (*environment, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// 查找密封任务的父区块
	parent := w.chain.CurrentBlock() //❶
	if genParams.parentHash != (common.Hash{}) {
		block := w.chain.GetBlockByHash(genParams.parentHash)
		if block == nil {
			return nil, fmt.Errorf("缺少父区块")
		}
		parent = block.Header()
	}
	// 检查时间戳的正确性，并根据需要调整时间戳
	timestamp := genParams.timestamp
	if parent.Time >= timestamp { //❷
		if genParams.forceTime {
			return nil, fmt.Errorf("无效的时间戳，父区块时间：%d，给定时间：%d", parent.Time, timestamp)
		}
		timestamp = parent.Time + 1
	}
	// 构建密封区块的区块头
	header := &types.Header{ //❹
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number, common.Big1),
		GasLimit:   core.CalcGasLimit(parent.GasLimit, w.config.GasCeil),
		Time:       timestamp,
		Coinbase:   genParams.coinbase,
	}
	// 设置额外字段
	if len(w.extra) != 0 {
		header.Extra = w.extra
	}
	// 如果可用，从 Beacon 链中设置随机字段
	if genParams.random != (common.Hash{}) {
		header.MixDigest = genParams.random
	}
	// 如果在 EIP-1559 链上，设置 baseFee 和 GasLimit
	if w.chainConfig.IsLondon(header.Number) {
		header.BaseFee = misc.CalcBaseFee(w.chainConfig, parent)
		if !w.chainConfig.IsLondon(parent.Number) {
			parentGasLimit := parent.GasLimit * w.chainConfig.ElasticityMultiplier()
			header.GasLimit = core.CalcGasLimit(parentGasLimit, w.config.GasCeil)
		}
	}
	// 使用默认或自定义的共识引擎运行共识准备
	if err := w.engine.Prepare(w.chain, header); err != nil { //❻
		log.Error("准备密封区块头失败", "错误", err)
		return nil, err
	}
	// 如果在奇怪的状态下开始挖矿，可能会发生
	// 注意 genParams.coinbase 可能与 header.Coinbase 不同，
	// 因为 clique 算法可以修改 header 中的 coinbase 字段。
	env, err := w.makeEnv(parent, header, genParams.coinbase)
	if err != nil {
		log.Error("创建密封上下文失败", "错误", err)
		return nil, err
	}
	return env, nil
}
```

先根据父块时间戳调整新区块的时间戳。如果新区块时间戳还小于父块时间戳，则直接在父块时间戳上加一秒。一种情是，新区块链时间戳比当前节点时间还快时，则需要稍做休眠❸，避免新出块属于未来。这也是区块时间戳可以作为区块链时间服务的一种保证。

有了父块，新块的基本信息是确认的。分别是父块哈希、新块高度、燃料上限、挖矿自定义数据、区块时间戳❹。

为了接受区块奖励，还需要设置一个不为空的矿工账户 Coinbase ❺。一个区块的挖矿难度是根据父块动态调整的，因此在正式处理交易前，需要根据共识算法设置新区块的挖矿难度❻。

至此，区块头信息准备就绪。

> ❸❺处对应的代码没找到，已经改写逻辑？

### 准备上下文环境
为了方便的共享当前新区块的信息，是专门定义了一个 environment ，专用于记录和当前挖矿工作相关内容。为即将开始的挖矿，先创建一份新的上下文环境信息。
```go
// Could potentially happen if starting to mine in an odd state.
// Note genParams.coinbase can be different with header.Coinbase
// since clique algorithm can modify the coinbase field in header.
env, err := w.makeEnv(parent, header, genParams.coinbase)
if err != nil {
	log.Error("Failed to create sealing context", "err", err)
	return nil, err
}
```

上下文环境信息中，记录着此新区块信息，分别有：
1. state： 状态DB，这个状态DB继承自父块。每笔交易的处理，实际上是在改变这个状态DB。
2. ~~ancestors： 祖先区块集，用于检测叔块是否合法。~~
3. ~~family: 近亲区块集，用于检测叔块是否合法。~~
4. ~~uncles：已合法加入的叔块集。~~
5. ~~tcount： 当请挖矿周期内已提交的交易数。~~
6. gasPool： 新区块可用燃料池。
7. header： 新区块区块头。
8. txs: 已提交的交易集合。
9. receipts： 已提交交易产生的交易回执集合。

makeEnv方法就是在初始化好上述信息。
```go
// makeEnv 创建一个用于封装区块的新环境。
func (w *worker) makeEnv(parent *types.Header, header *types.Header, coinbase common.Address) (*environment, error) {
	// 获取父状态以便在其上执行，并为矿工启动预取器以加快区块封装速度。
	state, err := w.chain.StateAt(parent.Root)
	if err != nil {
		return nil, err
	}
	state.StartPrefetcher("miner")

	// 注意传入的coinbase可能与header.Coinbase不同。
	env := &environment{
		signer:   types.MakeSigner(w.chainConfig, header.Number, header.Time),
		state:    state,
		coinbase: coinbase,
		header:   header,
	}
	// 跟踪返回错误的交易，以便将其删除
	env.tcount = 0
	return env, nil
}
```

### ~~选择叔块~~
~~前面不断将非分支上的区块存放在叔块集中。在打包新块选择叔块时，将从叔块集中选择适合的叔块。~~

> `worker` 中已经删除了 `localUncles` 、 `remoteUncles` 两个字段

> ### 以太坊中的分支和叔块是什么关系?
> 
> 以太坊中的分支（Fork）和叔块（Uncle Block）之间有一定的关系。  
> 
> 当多个矿工在同一时间内解决了区块的哈希难题时，会出现分支。在分支中，只有一个区块能被确认为主区块，其他区块则成为叔块。这些叔块与主区块具有相同的父区块，但由于网络延迟等原因未能及时传播到整个网络中被确认为主区块。  
> 
> 叔块的存在有助于减少分支的发生。当一个矿工解决了哈希难题并广播了区块时，其他矿工可能已经在同一时间内解决了相同的难题。这样就会出现分支，其中一个区块成为主区块，而其他区块成为叔块。通过允许叔块的存在，矿工们意识到即使他们的区块未能成为主区块，它们仍然可以获得一定的奖励。这样可以减少矿工为了争夺主区块而分散算力的情况，提高了整个网络的效率和稳定性。
> 
> 叔块的存在也有助于增加区块链网络的安全性和去中心化程度。通过允许叔块存在，矿工可以获得一定的奖励，即使他们的区块最终未能成为主区块。这鼓励了更多的矿工参与到网络中，增加了网络的算力和安全性。

### 提交交易
区块头已准备就绪，此刻开始从交易池拉取待处理的交易。将交易根据交易发送者分为两类，本地账户交易 localTxs 和外部账户交易 remoteTxs。本地交易优先不仅在交易池交易排队如此，在交易打包到区块中也是如此。本地交易优先，先将本地交易提交❸，再将外部交易提交❹。
```go
// fillTransactions 函数从txpool中检索待处理的交易并填充到给定的封装块中。
// 交易选择和排序策略可以在将来通过插件进行自定义。
func (w *worker) fillTransactions(interrupt *atomic.Int32, env *environment) error {
	// 将待处理的交易分为本地交易和远程交易
	// 填充块中的所有可用待处理交易。
	pending := w.eth.TxPool().Pending(true) //❶
	/*blobtxs := w.eth.BlobPool().Pending(
		uint256.MustFromBig(env.header.BaseFee),
		uint256.MustFromBig(misc.CalcBlobFee(*env.header.ExcessDataGas)),
	)
	log.Trace("副作用日志，非常棒", "blobs", len(blobtxs))*/

	localTxs, remoteTxs := make(map[common.Address][]*types.Transaction), pending //❷
	for _, account := range w.eth.TxPool().Locals() {
		if txs := remoteTxs[account]; len(txs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = txs
		}
	}
	if len(localTxs) > 0 { //❸
		txs := types.NewTransactionsByPriceAndNonce(env.signer, localTxs, env.header.BaseFee)
		if err := w.commitTransactions(env, txs, interrupt); err != nil {
			return err
		}
	}
	if len(remoteTxs) > 0 { //❹
		txs := types.NewTransactionsByPriceAndNonce(env.signer, remoteTxs, env.header.BaseFee)
		if err := w.commitTransactions(env, txs, interrupt); err != nil {
			return err
		}
	}
	return nil
}
```

交易处理完毕后，便可进入下一个环节。

### 提交区块
在交易处理完毕时，会获得交易回执和变更了区块状态。这些信息已经实时记录在上下文环境 environment 中。

将 environment 中的数据整理，便可根据共识规则构建一个区块。
```go
// commit 运行任何事务后的状态修改，组装最终的区块，并在共识引擎运行时提交新的工作。
// 注意假设允许对传递的环境进行突变，因此先进行深拷贝。
func (w *worker) commit(env *environment, interval func(), update bool, start time.Time) error {
	if w.isRunning() {
		if interval != nil {
			interval()
		}
		// 创建一个本地环境副本，避免与快照状态发生数据竞争。
		// https://github.com/ethereum/go-ethereum/issues/24299
		env := env.copy()
		// 在这里将提款设置为nil，因为这仅在PoW中调用。
		block, err := w.engine.FinalizeAndAssemble(w.chain, env.header, env.state, env.txs, nil, env.receipts, nil)
		if err != nil {
			return err
		}
		// 如果我们已经达到合并状态，则忽略
		if !w.isTTDReached(block.Header()) {
			select {
			case w.taskCh <- &task{receipts: env.receipts, state: env.state, block: block, createdAt: time.Now()}:
				fees := totalFees(block, env.receipts)
				feesInEther := new(big.Float).Quo(new(big.Float).SetInt(fees), big.NewFloat(params.Ether))
                log.Info("Commit new sealing work", "number", block.Number(), "sealhash", w.engine.SealHash(block.Header()),
                "txs", env.tcount, "gas", block.GasUsed(), "fees", feesInEther,
                "elapsed", common.PrettyDuration(time.Since(start)))

			case <-w.exitCh:
				log.Info("Worker has exited")
			}
		}
	}
	if update {
		w.updateSnapshot(env)
	}
	return nil
}
```

有了区块，就剩下最重要也是最核心的一步，执行 PoW 运算寻找 Nonce。这里并不是立刻开始寻找，而是发送一个PoW计算任务信号。
```go
select {
case w.taskCh <- &task{receipts: env.receipts, state: env.state, block: block, createdAt: time.Now()}:
...
}
```

### PoW计算寻找Nonce
之所以称之为挖矿，也是因为寻找Nonce的精髓所在。这是一道数学题，只能暴力破解，不断尝试不同的数字。直到找出一个符合要求的数字，这个数字称之为Nonce。寻找Nonce的过程，称之为挖矿。

寻找Nonce是需要时间的，耗时主要由区块难度决定。在代码设计上，以太坊是在 taskLoop 方法中，一直等待 task ❶。
```go
// taskLoop是一个独立的goroutine，用于从生成器获取密封任务并将其推送给共识引擎。
func (w *worker) taskLoop() {
	defer w.wg.Done()
	var (
		stopCh chan struct{}
		prev   common.Hash
	)

	// interrupt用于中断正在进行的密封任务。
	interrupt := func() {
		if stopCh != nil {
			close(stopCh)
			stopCh = nil
		}
	}
	for {
		select {
		case task := <-w.taskCh: //❶
			if w.newTaskHook != nil {
				w.newTaskHook(task)
			}
			// 拒绝由于重新提交而产生的重复密封任务。
			sealHash := w.engine.SealHash(task.block.Header()) //❷
			if sealHash == prev {
				continue
			}
			// 中断先前的密封操作
			interrupt() //❹
			stopCh, prev = make(chan struct{}), sealHash

			if w.skipSealHook != nil && w.skipSealHook(task) {
				continue
			}
			w.pendingMu.Lock()
			w.pendingTasks[sealHash] = task //❸
			w.pendingMu.Unlock()

			if err := w.engine.Seal(w.chain, task.block, w.resultCh, stopCh); err != nil {
				log.Warn("区块密封失败", "err", err)
				w.pendingMu.Lock()
				delete(w.pendingTasks, sealHash)
				w.pendingMu.Unlock()
			}
		case <-w.exitCh:
			interrupt()
			return
		}
	}
}
```

> 由 newWork 调用

当接收到挖矿任务后，先计算出这个区块所对应的一个哈希摘要❷，并登记此哈希对应的挖矿任务❸。登记的用途是方便查找该区块对应的挖矿任务信息，同时在开始新一轮挖矿时，会取消旧的挖矿工作，并从pendingTasks 中删除标记。以便快速作废挖矿任务。

随后，在共识规则下开始寻找Nonce，一旦找到Nonce，则发送给 resutlCh。同时，如果想取消挖矿任务，只需要关闭 stopCh。而在每次开始挖矿寻找Nonce前，便会关闭 stopCh 将当前进行中的挖矿任务终止❹。

### 等待挖矿结果 Nonce
上一步已经开始挖矿，寻找Nonce。下一步便是等待挖矿结束，在 resultLoop 中，一直在等待执行结果❶。
```go
// resultLoop 是一个独立的 goroutine，用于处理封装结果的提交并将相关数据刷新到数据库中。
func (w *worker) resultLoop() {
	defer w.wg.Done()
	for {
		select {
		case block := <-w.resultCh: //❶
			// 当接收到空结果时，进行短路处理。
			if block == nil {
				continue
			}
			// 当接收到由于重新提交导致的重复结果时，进行短路处理。
			if w.chain.HasBlock(block.Hash(), block.NumberU64()) { //❷
				continue
			}
			var (
				sealhash = w.engine.SealHash(block.Header())
				hash     = block.Hash()
			)
			w.pendingMu.RLock()
			task, exist := w.pendingTasks[sealhash]
			w.pendingMu.RUnlock()
			if !exist { //❸
				log.Error("找到区块但没有相关待处理任务", "number", block.Number(), "sealhash", sealhash, "hash", hash)
				continue
			}
			// 不同的区块可能共享相同的 sealhash，在此进行深拷贝以防止写-写冲突。
			var (
				receipts = make([]*types.Receipt, len(task.receipts))
				logs     []*types.Log
			)
			for i, taskReceipt := range task.receipts { //❹
				receipt := new(types.Receipt)
				receipts[i] = receipt
				*receipt = *taskReceipt

				// 添加区块位置字段
				receipt.BlockHash = hash
				receipt.BlockNumber = block.Number()
				receipt.TransactionIndex = uint(i)

				// 更新所有日志中的区块哈希，因为现在可用，而不是在创建各个交易的收据/日志时。
				receipt.Logs = make([]*types.Log, len(taskReceipt.Logs))
				for i, taskLog := range taskReceipt.Logs {
					log := new(types.Log)
					receipt.Logs[i] = log
					*log = *taskLog
					log.BlockHash = hash
				}
				logs = append(logs, receipt.Logs...)
			}
			// 将区块和状态提交到数据库。
			_, err := w.chain.WriteBlockAndSetHead(block, receipts, logs, task.state, true)
			if err != nil {
				log.Error("将区块写入链失败", "err", err)
				continue
			}
			log.Info("成功封装新区块", "number", block.Number(), "sealhash", sealhash, "hash", hash,
				"elapsed", common.PrettyDuration(time.Since(task.createdAt)))

			// 广播区块并宣布链插入事件
			w.mux.Post(core.NewMinedBlockEvent{Block: block}) //❻

		case <-w.exitCh:
			return
		}
	}
}
```

一旦找到Nonce，则说明挖出了新区块。

### 存储与广播挖出的新块
挖矿结果已经是一个包含正确Nonce 的新区块。在正式存储新区块前，需要检查区块是否已经存在，存在则不继续处理❷。

也许挖矿任务已被取消，如果Pending Tasks 中不存在区块对应的挖矿任务信息，则说明任务已被取消，就不需要继续处理❸。从挖矿任务中，整理交易回执，补充缺失信息，并收集所有区块事件日志信息❹。

随后，将区块所有信息写入本地数据库❺，对外发送挖出新块事件❻。在 eth 包中会监听并订阅此事件。

> 上方对应代码见上一小节

```go
// eth/handler.go
// minedBroadcastLoop函数将挖掘的区块发送给连接的节点。
func (h *handler) minedBroadcastLoop() {
	defer h.wg.Done()

	for obj := range h.minedBlockSub.Chan() {
		if ev, ok := obj.Data.(core.NewMinedBlockEvent); ok {
			h.BroadcastBlock(ev.Block, true)  // 首先将区块传播给节点 ❼
			h.BroadcastBlock(ev.Block, false) // 然后再向其他节点宣布 ❽
		}
	}
}
```

一旦接受到事件，则立即将广播。首随机广播给部分节点❼，再重新广播给不存在此区块的其他节点❽。

```go
// Commit block and state to database.
_, err := w.chain.WriteBlockAndSetHead(block, receipts, logs, task.state, true)
if err != nil {
	log.Error("Failed writing block to chain", "err", err)
	continue
}
log.Info("Successfully sealed new block", "number", block.Number(), "sealhash", sealhash, "hash", hash,
	"elapsed", common.PrettyDuration(time.Since(task.createdAt)))

// Broadcast the block and announce chain insertion event
w.mux.Post(core.NewMinedBlockEvent{Block: block}) //❾
```

同时，也需要通知程序内部的其他子系统，发送事件。新存储的区块，有可能导致切换链分支。如果变化，则队伍是发送 ChainSideEvent 事件。如果没有切换，则说明新区块仍然在当前的最长链上。对外发送 ChainEvent 和 ChainHeadEvent事件❾。新区块并非立即稳定，暂时存入到未确认区块集中。~~可这个 unconfirmed 仅仅是记录，但尚未具体使用。~~

## 总结
至此，已经讲解完以太坊挖出一个新区块所经历的各个环节。下面是一张流程图是对挖矿环节的细化，可以边看图便对比阅读此文。同时在讲解时，并没有涉及共识内部逻辑、以及提交交易到虚拟机执行内容。这些内容不是挖矿流程的重点，共识部分将在一下次讲解共识时细说。

<img src="../img/挖矿流程图.webp">

# 叔块
## 什么是叔块
是指**没能**成为区块链最长链的一部分的区块（陈旧的区块），但被后续区块收录时，这些区块称之为“叔块”。

它是针对区块而言的，是指被区块收录的陈旧的**祖先**孤块，它**没能**成为区块链最长链的一部分而被收录。

是针对当前区块所的而已的叔辈区块，一个区块最多可以记录 7 个叔块。叔块也是数据合法的区块，只是它所在的区块链分支没有成功成为主链的一部分。

如上图所示，新区块 E，它可以收录两个绿色的孤块B和C，但是灰色的区块不能被收录，因为他们的父区块并不在新区块所在的区块链上。而黄色区块和红色新区块是同辈区块，不能被新区块作为叔块收录。

## 为什么要设计叔块
在比特币中，因临时分叉（软分叉），没能成为最长合法链上的区块的区块，称之为孤块，孤块是没有区块奖励的。研究发现， 比特币的全网需要12.6秒一个新的区块才能传播到全网95%的节点。比特币系统，是平均10分钟才出一个区块，有足够的时间将新区块广播到全网其他节点，这种临时性的分叉几率就相当小。根据历史数据，大概平均3000多个区块，才会出现一次临时性分叉，相当于20多天出现一次这种临时性分叉，属于比较“罕见”的情况。

但是以太坊的出块时间已经缩短到12 到14 秒一个区块。更短的时间意味着，临时分叉的几率大幅提升。这是因为当矿工A挖出一个新区块后，需要向全网广播，广播的过程需要时间的。 由于以太坊出块时间短，其他节点可能还没有收到矿工A发布的区块，就已经挖出了同一高度的区块，这就造成了临时分叉。**在以太坊网络中，临时性分叉发生的几率在 6.6% 左右。**

![image](https://img.learnblockchain.cn/book_geth/20200905150333.png)

上图数据来源于 https://etherchain.org/ （2020年06月03日），当前以太坊叔块率为 6.6%。意味着在以太坊网络中，每 100 个区块，大约有 7 个叔块产生。如果按照平均 13.5 秒的出块时间计算，一个小时内有约 17.6 次临时分叉。

以太坊系统出现临时性分叉是一种普遍现象，如果采取和比特币一样处理方式，只有最长链上的区块才有出块奖励，**对于那些挖到区块而最终不在最长链上的矿工来说，就很不公平，而且这种“不公平”将是一个普遍情况。**这会影响矿工们挖矿的积极性，甚至可能削弱以太坊网络的系统安全，也是对算力的一种浪费。因此，以太坊系统对不在最长链上的叔块，设置了叔块奖励。

## 区块如何收录叔块
当节点不断接受到区块时，特别是同一高度的多个区块，会让以太坊陷入短期的软分叉中，或者在多个软分叉分支中来回切换。一旦出现软分叉，那么意味着有一个区块没能成为最长链的一部分。

![image](https://img.learnblockchain.cn/book_geth/20200905150334.png)

比如上图中，挖矿依次接收到 A、B 、C ，会在本地校验和存储这些区块，但根据最长链规则最终会切换到分支B上。那么在此刻，A和C 暂时成为了孤块。矿工会基于 B 挖取下一个新区块D。



![image](https://img.learnblockchain.cn/book_geth/20200905150335.png)

此时，D 就可以将本地还暂存的孤块 A 和 C 作为它的叔块而收录到区块 D 中。当然 D 不仅可以收录第一代祖先，还可以收录七代内的孤块。但有一些限制条件，以下图新区块 N 为例：

![image](https://img.learnblockchain.cn/book_geth/20200905150336.png)

+ N 不能收录 A ：因 A 不在七代祖先内（区间要求）；
+ N 不能收录 M：因 M 不是 N 的祖先，是兄弟而已；
+ N 不能收录 E、G、K、L ：因它们的父区块并不在 N 所在的区块分支上，但 B 可以；
+ N 不能同时收录 D、C 和 B：因为一个区块最多能收录两个叔块，做多是三选二；
+ 当 D 被 F 收录后，N 是不能重复收录 D 的；
+ N 不收录 F 或 H： 因为 F 和 H 不是孤块；

矿工在开挖新区块，准备区块头信息时，矿工将从本地节点存储中获取七代内的所有家族区块，根据上述规则选择最多两个叔块。另外，在选择时本地叔块优先选择。

## 叔块奖励分配
叔块的奖励分为两部分：奖励收录叔块的矿工和奖励叔块创建者。具体的奖励分配如下：

**奖励叔块的创建者**

叔块创建者的奖励根据“近远”关系而不同，和当前区块隔得越远，奖励越少。

$$ \text{叔块奖励}=\frac{8-(当前区块高度-叔块高度)}{8} * \text{当前区块挖矿奖励} $$

| 叔块   | 奖励 | 按挖矿奖励 2 ETH计算 |
| ------ | ---- | -------------------- |
| 第一代 | 7/8  | 1.75 ETH             |
| 第二代 | 6/8  | 1.5 ETH              |
| 第三代 | 5/8  | 1.25 ETH             |
| 第四代 | 4/8  | 1 ETH                |
| 第五代 | 3/8  | 0.75 ETH             |
| 第六代 | 2/8  | 0.5 ETH              |
| 第七代 | 1/8  | 0.25 ETH             |

注意叔块中所产生的交易费是不返给创建者的，毕竟叔块中的交易是不能作数的。

**收录叔块的矿工**

该矿工即为当前新区块的矿工，他处理获得原本的区块挖矿奖励（2 ETH）和交易手续费外，还能获得收录叔块奖励，每收录一个区块将得到多得 1/32 的区块挖矿奖励。

以区块 [10192970](https://etherscan.io/block/10192970) 为例：

![image](https://img.learnblockchain.cn/book_geth/20200905150337.png)

该区块矿工 2Miners:SOLO 总共获得了 2**.**385338652682918613 ETH奖励，其中：
1. 2 ETH 是挖矿奖励；
2. 0.322838652682918613 ETH 是交易手续费；
3. 0**.**0625 ETH 是收录了一个叔块的奖励，是 2 ETH的挖矿奖励的 1/32。

而收录的一个[叔块](https://etherscan.io/uncle/0xb3f6c988ba064ac1cad2058c52ab280f05dcc558687c8734e07aabf4cd00e855)是第 1代叔块，奖励 2 ETH 的 7 /8。

## 叔块是如何收录在区块中的
新区块收录的叔块是只记录叔块区块头的，叔块区块头信息记录在新区块体中，新区块头中记录了叔块集合的默克尔哈希值。

![image](https://img.learnblockchain.cn/book_geth/20200905150338.png)


# 挖矿奖励
矿工的收益来自于挖矿奖励。这样才能激励矿工积极参与挖矿，维护网络安全。那么，以太坊是如何奖励矿工的呢？我通过两个问题：“奖励是如何计算”和“何时何地奖励”，来协助你理解机制。

## 一、奖励是如何计算的
奖励分成三部分：新块奖励、叔块奖励和矿工费。

$\text{总奖励} =  新块奖励 + 叔块奖励 + 矿工费$

### 第一部分：**新块奖励**
它是奖励矿工消耗电能，完成工作量证明所给予的奖励。该奖励已进行两次调整，起初每个区块有 5 个以太币的奖励，在2017年10月16日（区块高度 4370000）  执行拜占庭硬分叉，将奖励下降到 3 个以太币；在2019年2月28日（区块高度 7280000）执行君士坦丁堡硬分叉，将奖励再次下降到 2 个以太币。

| 时间 |  事件    |  新块奖励  |
| ---- | ---- | ---- |
|      | 创世     |    5 ETH  |
|  2017年10月16日（4370000）    |  拜占庭硬分叉    |    3 ETH  |
| 在2019年2月28日（7280000）     |   君士坦丁堡硬分叉   |   2 ETH   |

以太坊在2015年7月正式发布以太坊主网后，其团队便规划发展阶段，分为前沿、家园、大都会和宁静四个阶段。拜占庭（Byzantium）和君士坦丁堡（Constantinople）是大都会的两个阶段。

新块奖励是矿工的主要收入来源，下降到 2 个 以太币的新块奖励。对矿机厂商和矿工，甚至以太坊挖矿生态都会产生比较大的影响和调整。因为挖矿收益减少，机会成本增加，在以太坊上挖矿将会变得性价比低于其他币种，因此可能会降低矿工的积极性。这也是迫使以太坊向以太坊2.0升级的一种助燃剂，倒逼以太坊更新换代。

### 第二部分：叔块奖励
以太坊出块间隔平均为12秒，区块链软分叉是一种普遍现象，如果采取和比特币一样处理方式，只有最长链上的区块才有出块奖励，对于那些挖到区块而最终不在最长链上的矿工来说，就很不公平，而且这种“不公平”将是一个普遍情况。这会影响矿工们挖矿的积极性，甚至可能削弱以太坊网络的系统安全，也是对算力的一种浪费。因此，以太坊系统对不在最长链上的叔块，设置了**叔块奖励**。

叔块奖励也分成两部分：**奖励叔块的创建者**和 **奖励收集叔块的矿工**。

叔块创建者的奖励根据“近远”关系而不同，和当前区块隔得越远，奖励越少。

$$
\text{挖叔块奖励}=\frac{8-(当前区块高度-叔块高度)}{8} * \text{当前区块挖矿奖励}
$$

| 叔块   | 奖励 | 按挖矿奖励 2 ETH计算 |
| ------ | ---- | -------------------- |
| 第一代 | 7/8  | 1.75 ETH             |
| 第二代 | 6/8  | 1.5 ETH              |
| 第三代 | 5/8  | 1.25 ETH             |
| 第四代 | 4/8  | 1 ETH                |
| 第五代 | 3/8  | 0.75 ETH             |
| 第六代 | 2/8  | 0.5 ETH              |
| 第七代 | 1/8  | 0.25 ETH             |

注意叔块中所产生的交易费是不返给创建者的，毕竟叔块中的交易是不能作数的。

**收录叔块的矿工**

每收录一个叔块将到多得 1/32 的区块挖矿奖励。
$$
收集叔块奖励 = 数量数量 \times  \frac{新块奖励}{32}
$$

### 第三部分：矿工费
矿工处理交易，并校验和打包到区块中去。此时交易签名者需要支付矿工费给矿工。每笔交易收多少矿工费，取决于交易消耗了多少燃料，它等于用户所自主设置的燃料单价GasPrice 乘以交易所消耗的燃料。
$$
Fee = \text{tx.gasPrice} \times \text{tx.gasUsed}
$$

## 二、何时何地奖励
奖励是在挖矿打包好一个区块时，便已在其中完成了奖励的发放，相当于是实时结算。

矿工费的发放是在处理完一笔交易时，便根据交易所消耗的 Gas 直接存入到矿工账户中；区块奖励和叔块奖励，则是在处理完所有交易后，进行奖励实时计算。

## 三、代码展示
**实时结算交易矿工费**

```go
// core/state_transition.go
// TransitionDb将通过应用当前消息来过渡状态，并返回带有以下字段的EVM执行结果。
//
//   - used gas: 使用的总气体（包括退款的气体）
//   - returndata: 来自EVM的返回数据
//   - 具体执行错误: 各种终止执行的EVM错误，例如ErrOutOfGas，ErrExecutionReverted
//
// 但是，如果遇到任何共识问题，则直接返回错误和空的EVM执行结果。
func (st *StateTransition) TransitionDb() (*ExecutionResult, error) {
	// 在应用消息之前，首先检查此消息是否满足所有共识规则。这些规则包括以下条款：
	//
	// 1. 消息调用者的nonce是正确的
	// 2. 调用者有足够的余额来支付交易费用（gaslimit * gasprice）
	// 3. 所需的气体数量在区块中可用
	// 4. 购买的气体足以覆盖内在使用
	// 5. 计算内在气体时没有溢出
	// 6. 调用者有足够的余额来支付**顶层**调用的资产转移

	// 检查条款1-3，如果一切正确则购买气体
	if err := st.preCheck(); err != nil {
		return nil, err
	}

	if tracer := st.evm.Config.Tracer; tracer != nil {
		tracer.CaptureTxStart(st.initialGas)
		defer func() {
			tracer.CaptureTxEnd(st.gasRemaining)
		}()
	}

	var (
		msg              = st.msg
		sender           = vm.AccountRef(msg.From)
		rules            = st.evm.ChainConfig().Rules(st.evm.Context.BlockNumber, st.evm.Context.Random != nil, st.evm.Context.Time)
		contractCreation = msg.To == nil
	)

	// 检查条款4-5，如果一切正确则减去内在气体
	gas, err := IntrinsicGas(msg.Data, msg.AccessList, contractCreation, rules.IsHomestead, rules.IsIstanbul, rules.IsShanghai)
	if err != nil {
		return nil, err
	}
	if st.gasRemaining < gas {
		return nil, fmt.Errorf("%w: 拥有 %d，期望 %d", ErrIntrinsicGas, st.gasRemaining, gas)
	}
	st.gasRemaining -= gas

	// 检查条款6
	if msg.Value.Sign() > 0 && !st.evm.Context.CanTransfer(st.state, msg.From, msg.Value) {
		return nil, fmt.Errorf("%w: 地址 %v", ErrInsufficientFundsForTransfer, msg.From.Hex())
	}

	// 检查初始化代码大小是否超过限制。
	if rules.IsShanghai && contractCreation && len(msg.Data) > params.MaxInitCodeSize {
		return nil, fmt.Errorf("%w: 代码大小 %v 限制 %v", ErrMaxInitCodeSizeExceeded, len(msg.Data), params.MaxInitCodeSize)
	}

	// 执行状态转换的准备步骤，其中包括：
	// - 准备访问列表（后柏林）
	// - 重置瞬态存储（EIP 1153）
	st.state.Prepare(rules, msg.From, st.evm.Context.Coinbase, msg.To, vm.ActivePrecompiles(rules), msg.AccessList)

	var (
		ret   []byte
		vmerr error // vm错误不影响共识，因此不分配给err
	)
	if contractCreation {
		ret, _, st.gasRemaining, vmerr = st.evm.Create(sender, msg.Data, st.gasRemaining, msg.Value)
	} else {
		// 增加下一笔交易的nonce
		st.state.SetNonce(msg.From, st.state.GetNonce(sender.Address())+1)
		ret, st.gasRemaining, vmerr = st.evm.Call(sender, st.to(), msg.Data, st.gasRemaining, msg.Value)
	}

	if !rules.IsLondon {
		// EIP-3529之前：退款被限制为gasUsed / 2
		st.refundGas(params.RefundQuotient)
	} else {
		// EIP-3529之后：退款被限制为gasUsed / 5
		st.refundGas(params.RefundQuotientEIP3529)
	}
	effectiveTip := msg.GasPrice
	if rules.IsLondon {
		effectiveTip = cmath.BigMin(msg.GasTipCap, new(big.Int).Sub(msg.GasFeeCap, st.evm.Context.BaseFee))
	}

	if st.evm.Config.NoBaseFee && msg.GasFeeCap.Sign() == 0 && msg.GasTipCap.Sign() == 0 {
		// 当设置了NoBaseFee并且费用字段为0时，跳过费用支付。
		// 这避免了在模拟调用时将负的effectiveTip应用于coinbase。
	} else {
		fee := new(big.Int).SetUint64(st.gasUsed())
		fee.Mul(fee, effectiveTip)
		st.state.AddBalance(st.evm.Context.Coinbase, fee)
	}

	return &ExecutionResult{
		UsedGas:    st.gasUsed(),
		Err:        vmerr,
		ReturnData: ret,
	}, nil
}
```

**实时结算挖矿奖励和叔块奖励**
```go
// Finalize函数实现了consensus.Engine接口，用于累积块和叔块的奖励。
func (ethash *Ethash) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
    // 累积任何块和叔块的奖励
    accumulateRewards(chain.Config(), state, header, uncles)
}

// AccumulateRewards函数给给定块的coinbase添加挖矿奖励。
// 总奖励由静态块奖励和包含的叔块奖励组成。每个叔块块的coinbase也会得到奖励。
func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {
	// 根据链的进展选择正确的块奖励
	blockReward := FrontierBlockReward
	if config.IsByzantium(header.Number) {
		blockReward = ByzantiumBlockReward
	}
	if config.IsConstantinople(header.Number) {
		blockReward = ConstantinopleBlockReward
	}
	// 累积矿工和任何包含的叔块的奖励
	reward := new(big.Int).Set(blockReward)
	r := new(big.Int)
	for _, uncle := range uncles {
		r.Add(uncle.Number, big8)
		r.Sub(r, header.Number)
		r.Mul(r, blockReward)
		r.Div(r, big8)
		state.AddBalance(uncle.Coinbase, r)

		r.Div(blockReward, big32)
		reward.Add(reward, r)
	}
	state.AddBalance(header.Coinbase, reward)
}
```

# 区块存储
这篇文章所说的挖矿环节中的**存储**环节，当矿工通过穷举计算找到了符合难度要求的区块 Nonce 后，标志着新区块已经成功被挖掘。

此时，矿工将在本地将这个合法的区块直接在本地存储，下面具体讲讲，在 geth 中矿工是如何存储自己挖掘的新区块的。

![image](https://img.learnblockchain.cn/book_geth/20200905150822.png)

在上一环节“PoW 寻找 Nonce” 后，已经拥有了完整的区块信息。

![image](https://img.learnblockchain.cn/book_geth/20200905150823.png)

而在“处理本地交易”和“处理远程交易”后，便拥有了完整的区块交易回执清单：

![image](https://img.learnblockchain.cn/book_geth/20200905150824.png)

区块中的每一笔交易在处理后，都会存在一份交易回执。在交易回执中记录着这边交易的执行结果信息，对于交易回执，我们已经在前面的课程有讲解，这里不再复述。

同时在“发放区块奖励”后，区块的状态不会再发生变化，此时，我们就已经拿到了一个可以代表该区块的状态数据。状态`state`，在内存中将记录着本次区块中交易执行后状态所发送的变化信息，包括新增、变更和删除的数据。

前面所说的区块（Block）、交易回执（Receipt）、状态（State）就是本次挖矿的产物，在本地需要存储的也只有这三部分数据。

![image](https://img.learnblockchain.cn/book_geth/20200905150825.png)

这些数据，在挖矿中处理存储的代码如下：
```go
// resultLoop 是一个独立的 goroutine，用于处理封装结果的提交并将相关数据刷新到数据库中。
func (w *worker) resultLoop() {
	...
			// 不同的区块可能共享相同的 sealhash，在此进行深拷贝以防止写-写冲突。
			var (
				receipts = make([]*types.Receipt, len(task.receipts))
				logs     []*types.Log
			)
			for i, taskReceipt := range task.receipts { //❶
				receipt := new(types.Receipt)
				receipts[i] = receipt
				*receipt = *taskReceipt

				// 添加区块位置字段
				receipt.BlockHash = hash
				receipt.BlockNumber = block.Number()
				receipt.TransactionIndex = uint(i)

				// 更新所有日志中的区块哈希，因为现在可用，而不是在创建各个交易的收据/日志时。
				receipt.Logs = make([]*types.Log, len(taskReceipt.Logs))
				for i, taskLog := range taskReceipt.Logs {
					log := new(types.Log)
					receipt.Logs[i] = log
					*log = *taskLog
					log.BlockHash = hash
				}
				logs = append(logs, receipt.Logs...) //❷
			}
			// 将区块和状态提交到数据库。 //❸
			_, err := w.chain.WriteBlockAndSetHead(block, receipts, logs, task.state, true)
			if err != nil {
				log.Error("将区块写入链失败", "err", err)
				continue
			}
			log.Info("成功封装新区块", "number", block.Number(), "sealhash", sealhash, "hash", hash,
				"elapsed", common.PrettyDuration(time.Since(task.createdAt)))
	
	...
}
```

- ❶ 遍历交易回执，给每一个交易回执添加本次区块信息（blockHash，BlockNumber、TransactionIndex），这样就可以在本地记录交易回执和区块间的查找关系。
- ❷ 同时将交易回执中生成的日志信息提取到一个大集合中，以便作为一个区块日志整体存储。
- ❸ 开始提交区块（Block）、交易回执（Receipt）、状态（State）和日志（log）到本地数据库中。

在`writeBlockWithState`中，是将所有数据以一个批处理事务写入到数据库中：
```go
// writeBlockAndSetHead是WriteBlockAndSetHead的内部实现。
// 这个函数期望chain被持有。
func (bc *BlockChain) writeBlockAndSetHead(block *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB, emitHeadEvent bool) (status WriteStatus, err error) {
    if err := bc.writeBlockWithState(block, receipts, state); err != nil {
        return NonStatTy, err
    }
	
	...

	// Set new head.
	if status == CanonStatTy {
        bc.writeHeadBlock(block)
    }
	
	...
}

// writeBlockWithState writes block, metadata and corresponding state data to the
// database.
func (bc *BlockChain) writeBlockWithState(block *types.Block, receipts []*types.Receipt, state *state.StateDB) error {
	...
    // 无论规范状态如何，将区块本身写入数据库。
    //
    // 注意，区块的所有组成部分（td、hash->number映射、头、体、收据）应该是原子性写入的。BlockBatch用于包含所有组件。
    blockBatch := bc.db.NewBatch()
    rawdb.WriteTd(blockBatch, block.Hash(), block.NumberU64(), externTd) // 将td写入数据库
    rawdb.WriteBlock(blockBatch, block) // 将区块写入数据库
    rawdb.WriteReceipts(blockBatch, block.Hash(), block.NumberU64(), receipts) // 将收据写入数据库
    rawdb.WritePreimages(blockBatch, state.Preimages()) // 将预图像写入数据库
    if err := blockBatch.Write(); err != nil {
        log.Crit("Failed to write block into disk", "err", err) // 写入数据库失败时记录错误日志
    }
    // 将所有缓存的状态更改提交到底层内存数据库。
    root, err := state.Commit(bc.chainConfig.IsEIP158(block.Number()))
    if err != nil {
        return err
    }
    // 如果我们运行的是归档节点，总是刷新
    if bc.cacheConfig.TrieDirtyDisabled {
        return bc.triedb.Commit(root, false)
    }
    // 完整但不是归档节点，进行适当的垃圾回收
    bc.triedb.Reference(root, common.Hash{}) // 元数据引用以保持trie存活
    bc.triegc.Push(root, -int64(block.NumberU64())) // 将根节点推入垃圾回收队列，设置负数表示该根节点不可删除
    ...
}
```

在一个事务中，分别向数据库中写入了区块难度、区块、交易回执、Preimages（key映射），最后将 state 提交。

那么，geth 是如何在本地将这些数据存放到键值数据库 levelDB 中的呢？这里，给大家整理一份键值信息表。

| Key                       | Value                             | 说明                                           |
| ------------------------- | --------------------------------- | ---------------------------------------------- |
| “b”.blockNumber.blockHash | blockBody： uncles + transactions | 通过区块哈希和高度存储对应的区块叔块和交易信息 |
| "H".blockHash             | blockNumber                       | 通过区块哈希记录对于的区块高度                 |
| “h”.blockNumber.blockHash | blockHeader                       | 通过区块哈希和高度存储对于的区块头             |
| ”r“.blockNumber           | receipts                          | 通过区块高度记录区块的交易回执记录             |
| "h".blockNumber           | blockHash                         | 区块高度对应的区块哈希                         |
| ”l“.txHash                | blockNumber                       | 记录交易哈希所在的区块高度                     |
| ”LastBlock“               | blockHash                         | 更新最后一个区块哈希值                         |
| ”LastHeader“              | blockHash                         | 更新最后一个区块头所在位置                     |

> 注意，上面的 value 信息，是需要序列化为 bytes 才能存储到 leveldb 中，序列化是以太坊自定义的 RLP 编码技术。你有没有想过它为何要添加一个前缀呢？比如”b“、”H“等等，第一个好处是将不同数据分类，另一个重要的原因是在leveldb中数据是以 key 值排序存储的，这样在按顺序遍历区块头、查询同类型数据时，读的性能会更好。

正是因为在我们在本地保存了区块数据的一些映射关系，我们才能快速的从本地数据库中只需要提供少量的信息就就能组合一个或者多个键值关系查询到目标数据。下面我列举了一些常见的以太坊API，你觉得该如何从DB中查找出数据呢？

1. 通过交易哈希获取交易信息：
    ```js
    eth_getTransactionByHash("0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238")
    ```
2. 查询最后一个区块信息：
    ```js
    eth_getBlockByNumber("latest")
    ```
3. 通过交易哈希获取交易回执
    ```js
    eth_getTransactionReceipt("0x444172bef57ad978655171a8af2cfd89baa02a97fcb773067aef7794d6913374")
    ```


![image](https://img.learnblockchain.cn/book_geth/20200905150826.png)






























