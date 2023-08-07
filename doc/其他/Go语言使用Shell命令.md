Go 语言中执行外部命令主要的方法是使用包 `os/exec`。

此包的详细文档见 exec package - os/exec - pkg.go.dev，这里只介绍几种常用操作。

执行命令也分几种情况：
1. 仅执行命令；
2. 执行命令，获取结果，不区分 stdout 和 stderr；
3. 执行命令，获取结果，区分 stdout 和 stderr。

另外，默认的命令执行是在 go 进程当前的目录下执行的，我们可能还需要指定命令执行目录。

下面我们逐个说。

### 1. 仅执行命令

执行命令，首先要拼接一下命令和参数，然后运行命令。

- 拼接命令与参数使用 exec.Command()，其会返回一个 *Cmd；
    ```go
    func Command(name string, arg ...string) *Cmd
    ```

    执行命令使用 *Cmd 中的 Run() 方法，Run() 返回的只有 error。
    ```go
    func (c *Cmd) Run() error
    ```

    我们直接看代码:
    ```go
    package main
    
    import (
        "log"
        "os/exec"
    )
    
    func ExecCommand(name string, args ...string) {
        cmd := exec.Command(name, args...) // 拼接参数与命令
        if err := cmd.Run(); err != nil {  // 执行命令，若命令出错则打印错误到 stderr
            log.Println(err)
        }
    }
    
    func main() {
        ExecCommand("ls", "-l")
    }
    ```

    执行代码，没有任何输出。
    
    上面的代码中，我们执行了命令 ls -l，但是没有得到任何东西。

### 2. 获取结果
#### 2.1 不区分 stdout 和 stderr

要组合 stdout 和 stderr 输出，，Cmd 中有方法：
```go
func (c *Cmd) CombinedOutput() ([]byte, error)
```

用这个方法来执行命令（即这个方法是已有 Run() 方法的作用的，无需再执行 Run()）。

我们修改上述代码
```go
package main

import (
    "fmt"
    "log"
    "os/exec"
)

func ExecCommand(name string, args ...string) {
    cmd := exec.Command(name, args...) // 拼接参数与命令
    
    var output []byte
    var err error
    
    if output, err = cmd.CombinedOutput(); err != nil {
        log.Println(err)
    }
	
	fmt.Print(string(output)) // output 是 []byte 类型，这里最好转换成 string
}

func main() {
    ExecCommand("ls", "-l")
}
```

我们得到了 ls -l 这条命令的输出.

#### 2.2 区分 stdout 和 stderr
区分 stdout 和 stderr，要先给 cmd 中的成员指定一个输出 buffer，然后执行 Run() 就可以。
```go
package main

import (
    "bytes"
    "fmt"
    "log"
    "os/exec"
)

func ExecCommand(name string, args ...string) {
    cmd := exec.Command(name, args...) // 拼接参数与命令

    var stdout bytes.Buffer
    var stderr bytes.Buffer
    var err error

    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err = cmd.Run(); err != nil {
        log.Println(err)
    }
  
    fmt.Print(stdout.String())
    fmt.Print(stderr.String())
}

func main() {
    ExecCommand("ls", "-l")
}
```
