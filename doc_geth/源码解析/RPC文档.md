包rpc在多个传输上实现双向JSON-RPC 2.0。

它提供对对象的导出方法的访问，这些对象通过网络或其他I/O连接可见为“服务”。遵循特定约定的导出方法可以远程调用。它还支持发布/订阅模式。

# RPC方法
满足以下标准的方法可用于远程访问：
- 方法必须导出
- 方法返回0、1（响应或错误）或2（响应和错误）个值

示例方法：

	func(s *CalcService) Add(a, b int)(int, error)

当返回的错误不为nil时，返回的整数将被忽略，并将错误发送回客户端。否则将返回的整数发送回客户端。

通过接受指针值作为参数支持可选参数。例如，如果我们想在可选有限域中进行加法运算，我们可以接受mod参数作为指针值。

	func(s *CalcService) Add(a, b int, mod *int)(int, error)

可以使用2个整数和第三个参数为空值调用此RPC方法。在这种情况下，mod参数将为nil。或者可以使用3个整数进行调用，在这种情况下，mod将指向给定的第三个参数。由于可选参数是最后一个参数，因此RPC包还将接受2个整数作为参数。它将把mod参数作为nil传递给RPC方法。

服务器提供ServeCodec方法，该方法接受ServerCodec实例。它将从编解码器中读取请求，处理请求并使用编解码器将响应发送回客户端。服务器可以并发执行请求。响应可以按任意顺序发送回客户端。

使用JSON编解码器的示例服务器：

	 type CalculatorService struct {}

	 func (s *CalculatorService) Add(a, b int) int {
		return a + b
	 }

	 func (s *CalculatorService) Div(a, b int) (int, error) {
		if b == 0 {
			return 0, errors.New("divide by zero")
		}
		return a/b, nil
	 }

	 calculator := new(CalculatorService)
	 server := NewServer()
	 server.RegisterName("calculator", calculator)
	 l, _ := net.ListenUnix("unix", &net.UnixAddr{Net: "unix", Name: "/tmp/calculator.sock"})
	 server.ServeListener(l)

# 订阅
该程序还通过使用订阅支持发布订阅模式。被认为适合通知的方法必须满足以下条件：
- 方法必须导出
- 第一个方法参数类型必须为context.Context
- 方法必须具有返回类型（rpc.Subscription，error）

示例方法：

	func(s *BlockChainService) NewBlocks(ctx context.Context)(rpc.Subscription, error) {
		...
	}

当包含订阅方法的服务注册到服务器时，例如在“blockchain”命名空间下，通过调用“blockchain_subscribe”方法创建订阅。

当用户发送取消订阅请求或用于创建订阅的连接关闭时，将删除订阅。这可以由客户端和服务器发起。对于任何写错误，服务器将关闭连接。

有关订阅的更多信息，请参见https://github.com/ethereum/go-ethereum/wiki/RPC-PUB-SUB。

# 反向调用

在任何方法处理程序中，都可以通过ClientFromContext方法访问rpc.Client的实例。使用此客户端实例，可以在RPC连接上执行服务器到客户端的方法调用。