# GOPATH设置
go 命令依赖一个重要的环境变量：$GOPATH

Windows系统中环境变量的形式为`%GOPATH%`

Windows 设置如下，新建一个环境变量名称叫做GOPATH：

GOPATH=c:\mygo

GOPATH允许多个目录，当有多个目录时，请注意分隔符，多个目录的时候Windows是分号，Linux系统是冒号，当有多个GOPATH时，默认会将go get的内容放在第一个目录下。

以上 $GOPATH 目录约定有三个子目录：
* `src`：存放源代码（比如：.go .c .h .s等）
* `pkg`：编译后生成的文件（比如：.a）
* `bin`：编译后生成的可执行文件（为了方便，可以把此目录加入到 $PATH 变量中，如果有多个gopath，那么使用${GOPATH//://bin:}/bin添加所有的bin目录）

# 代码目录结构规划
GOPATH下的src目录就是接下来开发程序的主要目录，所有的源码都是放在这个目录下面，那么一般我们的做法就是一个目录一个项目，例如: `$GOPATH/src/mymath` 表示`mymath`这个应用包或者可执行应用，这个根据`package`是`main`还是其他来决定，`main`的话就是可执行应用，其他的话就是应用包，这个会在后续详细介绍`package`。

所以当新建应用或者一个代码包时都是在src目录下新建一个文件夹，文件夹名称一般是代码包名称，当然也允许多级目录，例如在`src`下面新建了目录`$GOPATH/src/github.com/astaxie/beedb` 那么这个包路径就是`"github.com/astaxie/beedb"`，包名称是最后一个目录`beedb`

下面就以mymath为例来讲述如何编写应用包，执行如下代码
```shell
cd $GOPATH/src
mkdir mymath
```
新建文件sqrt.go，内容如下
```go
// $GOPATH/src/mymath/sqrt.go源码如下：
package mymath

func Sqrt(x float64) float64 {
	z := 0.0
	for i := 0; i < 1000; i++ {
		z -= (z*z - x) / (2 * x)
	}
	return z
}
```
> 注意：一般建议`package`的名称和目录名保持一致

# 编译应用
## 安装
上面已经建立了自己的应用包，如何进行编译安装呢？有两种方式可以进行安装

1. 只要进入对应的应用包目录，然后执行go install，就可以安装了
2. 在任意的目录执行如下代码go install mymath

安装完之后，我们可以进入如下目录
```shell
cd $GOPATH/pkg/${GOOS}_${GOARCH}
//可以看到如下文件
mymath.a
```

## 调用
这个`.a`文件是应用包，那么我们如何进行调用呢？

接下来我们新建一个应用程序来调用这个应用包

新建应用包mathapp
```shell
cd $GOPATH/src
mkdir mathapp
cd mathapp
vim main.go
```

$GOPATH/src/mathapp/main.go源码：

```shell
package main

import (
	"mymath"
	"fmt"
)

func main() {
	fmt.Printf("Hello, world.  Sqrt(2) = %v\n", mymath.Sqrt(2))
}
```
可以看到这个的`package`是`main`，import里面调用的包是`mymath`，这个就是相对于`$GOPATH/src`的路径，如果是多级目录，就在import里面引入多级目录，如果你有多个`GOPATH`，也是一样，Go会自动在多个$GOPATH/src中寻找。

## 编译
如何编译程序呢？进入该应用目录，然后执行go build，那么在该目录下面会生成一个mathapp的可执行文件
```shell
./mathapp
```
输出如下内容
```shell
Hello, world.  Sqrt(2) = 1.414213562373095
```

如何安装该应用，进入该目录执行go install,那么在$GOPATH/bin/下增加了一个可执行文件mathapp, 还记得前面我们把$GOPATH/bin加到我们的PATH里面了，这样可以在命令行输入如下命令就可以执行
```shell
mathapp
```

也是输出如下内容
```shell
Hello, world.  Sqrt(2) = 1.414213562373095
```

这里展示了如何编译和安装一个可运行的应用，以及如何设计目录结构。

对于启用go module的Go版本，需要对mod.go进行配置。（使用replace的方式将远程包替换为本地包）

# 获取远程包
go语言有一个获取远程包的工具就是`go get`，目前`go get`支持多数开源社区(例如：GitHub、googlecode、bitbucket、Launchpad)
```shell
go get github.com/astaxie/beedb
```
> go get -u 参数可以自动更新包，而且当go get的时候会自动获取该包依赖的其他第三方包

通过这个命令可以获取相应的源码，对应的开源平台采用不同的源码控制工具，例如GitHub采用git、googlecode采用hg，所以要想获取这些源码，必须先安装相应的源码控制工具

通过上面获取的代码在我们本地的源码相应的代码结构如下
```shell
$GOPATH
  src
   |--github.com
		  |-astaxie
			  |-beedb
   pkg
	|--相应平台
		 |-github.com
			   |--astaxie
					|beedb.a
```
go get本质上可以理解为首先第一步是通过源码工具clone代码到src下面，然后执行go install

在代码中如何使用远程包，很简单的就是和使用本地包一样，只要在开头import相应的路径就可以
```shell
import "github.com/astaxie/beedb"
```

# 程序的整体结构
通过上面建立的本地的mygo的目录结构如下所示
```
bin/
	mathapp
pkg/
	平台名/ 如：darwin_amd64、linux_amd64
		 mymath.a
		 github.com/
			  astaxie/
				   beedb.a
src/
	mathapp
		  main.go
	mymath/
		  sqrt.go
	github.com/
		   astaxie/
				beedb/
					beedb.go
					util.go
```
从上面的结构我们可以很清晰的看到，`bin`目录下面存的是编译之后可执行的文件，`pkg`下面存放的是应用包，`src`下面保存的是应用源代码