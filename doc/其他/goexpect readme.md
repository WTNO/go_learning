这个软件包是Go语言中Expect的实现。

### 特点：
- 使用真实的PTY（伪终端）生成和控制本地进程。
- 原生SSH生成器。
- 用于测试的Expect生成器。
- 通用生成器，使实现其他生成器变得简单。
- 具有批处理程序，可以实现工作流程而不必编写额外的逻辑和代码。

### 选项
所有Spawn函数都接受expect.Option类型的可变参数，用于更改Expecter的选项。

#### CheckDuration
Go Expecter默认每两秒检查一次新数据。可以使用CheckDuration函数CheckDuration(d time.Duration) Option来更改此设置。

#### Verbose
Verbose选项用于打开/关闭Expect/Send语句的详细日志记录。在故障排除工作流程时，此选项非常有用，因为它会记录与设备的每次交互。

#### VerboseWriter
VerboseWriter选项可用于更改详细会话日志的输出位置。使用此选项将开始将详细输出写入提供的io.Writer而不是默认日志。

请参阅ExampleVerbose代码，了解如何使用此选项的示例。

#### NoCheck
Go Expecter定期检查生成的进程/SSH会话/TELNET等会话是否仍然活动。此选项关闭该检查。

#### DebugCheck
DebugCheck选项为Expecter的活动检查添加调试功能，每次运行检查时都会记录信息。可用于Spawners的故障排除和调试。

#### ChangeCheck
ChangeCheck选项可以用全新的检查函数替换Spawner的Check函数。

#### SendTimeout
SendTimeout设置Send命令的超时时间，如果没有超时，Send命令将永远等待Expecter进程。

#### BufferSize
BufferSize选项提供了一种配置客户端IO缓冲区大小（以字节为单位）的机制。

### 基本示例
#### networkbit.ch
在networkbit.ch上有一篇关于goexpect的[文章](http://networkbit.ch/golang-ssh-client/)，其中提供了一些示例。

维基百科上的[Expect示例](https://en.wikipedia.org/wiki/Expect)。

#### Telnet
首先，我们尽可能地复制维基百科上的Telnet示例。

互动：
```diff
+ username:
- user\n
+ password:
- pass\n
+ %
- cmd\n
+ %
- exit\n
```

为了使示例简洁，省略了错误检查。
```go
package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/google/goexpect"
	"github.com/google/goterm/term"
)

const (
	timeout = 10 * time.Minute
)

var (
	addr = flag.String("address", "", "address of telnet server")
	user = flag.String("user", "", "username to use")
	pass = flag.String("pass", "", "password to use")
	cmd  = flag.String("cmd", "", "command to run")

	userRE   = regexp.MustCompile("username:")
	passRE   = regexp.MustCompile("password:")
	promptRE = regexp.MustCompile("%")
)

func main() {
	flag.Parse()
	fmt.Println(term.Bluef("Telnet 1 example"))

	e, _, err := expect.Spawn(fmt.Sprintf("telnet %s", *addr), -1)
	if err != nil {
		log.Fatal(err)
	}
	defer e.Close()

	e.Expect(userRE, timeout)
	e.Send(*user + "\n")
	e.Expect(passRE, timeout)
	e.Send(*pass + "\n")
	e.Expect(promptRE, timeout)
	e.Send(*cmd + "\n")
	result, _, _ := e.Expect(promptRE, timeout)
	e.Send("exit\n")

	fmt.Println(term.Greenf("%s: result: %s\n", *cmd, result))
}
```

基本上，要运行并附加到一个进程，可以使用expect.Spawn(<cmd>, <timeout>)。Spawn返回一个Expecter e，可以使用e.Expect和e.Send命令来匹配输出中的信息和发送信息。

请参阅https://github.com/google/goexpect/blob/master/examples/newspawner/telnet.go示例，了解稍微详细一些的版本。

#### FTP
对于FTP示例，我们使用expect.Batch进行以下交互。
```diff
+ username:
- user\n
+ password:
- pass\n
+ ftp>
- prompt\n
+ ftp>
- mget *\n
+ ftp>'
- bye\n
```

ftp_example.go
```go
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/goexpect"
	"github.com/google/goterm/term"
)

const (
	timeout = 10 * time.Minute
)

var (
	addr = flag.String("address", "", "address of telnet server")
	user = flag.String("user", "", "username to use")
	pass = flag.String("pass", "", "password to use")
)

func main() {
	flag.Parse()
	fmt.Println(term.Bluef("Ftp 1 example"))

	e, _, err := expect.Spawn(fmt.Sprintf("ftp %s", *addr), -1)
	if err != nil {
		log.Fatal(err)
	}
	defer e.Close()

	e.ExpectBatch([]expect.Batcher{
		&expect.BExp{R: "username:"},
		&expect.BSnd{S: *user + "\n"},
		&expect.BExp{R: "password:"},
		&expect.BSnd{S: *pass + "\n"},
		&expect.BExp{R: "ftp>"},
		&expect.BSnd{S: "bin\n"},
		&expect.BExp{R: "ftp>"},
		&expect.BSnd{S: "prompt\n"},
		&expect.BExp{R: "ftp>"},
		&expect.BSnd{S: "mget *\n"},
		&expect.BExp{R: "ftp>"},
		&expect.BSnd{S: "bye\n"},
	}, timeout)

	fmt.Println(term.Greenf("All done"))
}
```

使用expect.Batcher可以使标准的Send/Expect交互更加简洁和易于编写。

#### SSH
在SSH登录示例中，我们测试expect.Caser和Case标签。

此外，我们将使用Go Expect的本地SSH Spawner，而不是生成一个进程。

互动：
```diff
+ "Login: "
- user
+ "Password: "
- pass1
+ "Wrong password"
+ "Login"
- user
+ "Password: "
- pass2
+ router#
```

ssh_example.go
```go
package main

import (
	"flag"
	"fmt"
	"log"
	"regexp"
	"time"

	"golang.org/x/crypto/ssh"

	"google.golang.org/grpc/codes"

	"github.com/google/goexpect"
	"github.com/google/goterm/term"
)

const (
	timeout = 10 * time.Minute
)

var (
	addr  = flag.String("address", "", "address of telnet server")
	user  = flag.String("user", "user", "username to use")
	pass1 = flag.String("pass1", "pass1", "password to use")
	pass2 = flag.String("pass2", "pass2", "alternate password to use")
)

func main() {
	flag.Parse()
	fmt.Println(term.Bluef("SSH Example"))

	sshClt, err := ssh.Dial("tcp", *addr, &ssh.ClientConfig{
		User:            *user,
		Auth:            []ssh.AuthMethod{ssh.Password(*pass1)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		log.Fatalf("ssh.Dial(%q) failed: %v", *addr, err)
	}
	defer sshClt.Close()

	e, _, err := expect.SpawnSSH(sshClt, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer e.Close()

	e.ExpectBatch([]expect.Batcher{
		&expect.BCas{[]expect.Caser{
			&expect.Case{R: regexp.MustCompile(`router#`), T: expect.OK()},
			&expect.Case{R: regexp.MustCompile(`Login: `), S: *user,
				T: expect.Continue(expect.NewStatus(codes.PermissionDenied, "wrong username")), Rt: 3},
			&expect.Case{R: regexp.MustCompile(`Password: `), S: *pass1, T: expect.Next(), Rt: 1},
			&expect.Case{R: regexp.MustCompile(`Password: `), S: *pass2,
				T: expect.Continue(expect.NewStatus(codes.PermissionDenied, "wrong password")), Rt: 1},
		}},
	}, timeout)

	fmt.Println(term.Greenf("All done"))
}
```

### 通用的Spawner
Go Expect包支持使用`func SpawnGeneric(opt *GenOptions, timeout time.Duration, opts ...Option) (*GExpect, <-chan error, error)` 函数添加新的Spawner。

telnet spawner

来自newspawner示例。
```go
func telnetSpawn(addr string, timeout time.Duration, opts ...expect.Option) (expect.Expecter, <-chan error, error) {
	conn, err := telnet.Dial(network, addr)
	if err != nil {
		return nil, nil, err
	}

	resCh := make(chan error)

	return expect.SpawnGeneric(&expect.GenOptions{
		In:  conn,
		Out: conn,
		Wait: func() error {
			return <-resCh
		},
		Close: func() error {
			close(resCh)
			return conn.Close()
		},
		Check: func() bool { return true },
	}, timeout, opts...)
}
```

### Fake Spawner
Go Expect包中包含一个Fake Spawner函数 `func SpawnFake(b []Batcher, timeout time.Duration, opt ...Option) (*GExpect, <-chan error, error)`。这被期望用于简化测试和模拟交互式工作流程。

Fake Spawner
```go
// TestExpect tests the Expect function.
func TestExpect(t *testing.T) {
	tests := []struct {
		name    string
		fail    bool
		srv     []Batcher
		timeout time.Duration
		re      *regexp.Regexp
	}{{
		name: "Match prompt",
		srv: []Batcher{
			&BSnd{`
Pretty please don't hack my chassis

router1> `},
		},
		re:      regexp.MustCompile("router1>"),
		timeout: 2 * time.Second,
	}, {
		name: "Match fail",
		fail: true,
		re:   regexp.MustCompile("router1>"),
		srv: []Batcher{
			&BSnd{`
Welcome

Router42>`},
		},
		timeout: 1 * time.Second,
	}}

	for _, tst := range tests {
		exp, _, err := SpawnFake(tst.srv, tst.timeout)
		if err != nil {
			if !tst.fail {
				t.Errorf("%s: SpawnFake failed: %v", tst.name, err)
			}
			continue
		}
		out, _, err := exp.Expect(tst.re, tst.timeout)
		if got, want := err != nil, tst.fail; got != want {
			t.Errorf("%s: Expect(%q,%v) = %t want: %t , err: %v, out: %q", tst.name, tst.re.String(), tst.timeout, got, want, err, out)
			continue
		}
	}
}
```




