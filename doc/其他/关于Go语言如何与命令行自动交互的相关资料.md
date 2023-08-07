# Expect
Expect是Unix系统中用来进行自动化控制和测试的软件工具，由Don Libes制作，作为Tcl脚本语言的一个扩展，应用在交互式软件中如telnet，ftp，Passwd，fsck，rlogin，tip，ssh等等。该工具利用Unix伪终端包装其子进程，允许任意程序通过终端接入进行自动化控制；也可利用Tk工具，将交互程序包装在X11的图形用户界面中。

## 基本介绍
Expect被用于自动化控制Telnet、FTP、passwd、fsck、rlogin、tip、SSH等交互式应用程序的操作。Expect使用伪终端（Unix）或模拟控制台（Windows），启动目标程序，然后通过终端或控制台界面与其进行通信，就像人类一样。另一个Tcl扩展Tk可以用于提供图形用户界面（GUI）。

## 用法
Expect作为一个"胶水"，将现有的工具连接在一起。总体思路是找出如何让Expect使用系统现有的工具，而不是在Expect内部解决问题。

Expect的一个关键用途是商业软件产品。许多这些产品提供某种类型的命令行界面，但通常缺乏编写脚本所需的功能。它们是为了为使用该产品的用户提供服务而构建的，但公司通常不会投入资源来完全实现一个强大的脚本语言。Expect脚本可以生成一个shell，查找环境变量，执行一些Unix命令来获取更多信息，然后使用所需的信息进入产品的命令行界面，以实现用户的目标。在通过与产品的命令行界面交互获取信息之后，脚本可以根据情况做出智能决策，决定采取什么行动，如果需要的话。

每当完成一个Expect操作时，结果都会存储在一个名为$expect_out的本地变量中。这使得脚本可以收集信息并反馈给用户，同时还可以根据情况决定下一步发送什么内容。

Expect的常见用途是为程序、实用工具或嵌入式系统设置测试套件。DejaGnu是一个使用Expect编写的测试套件，用于测试。它已经被用于测试GCC和嵌入式开发等远程目标。

Expect脚本可以使用一个名为'autoexpect'的工具自动化。该工具观察您的操作并使用启发式算法生成一个Expect脚本。生成的代码可能很大且有些晦涩，但可以通过调整生成的脚本来获得精确的代码。

```shell
# 假设 $remote_server, $my_user_id, $my_password 和 $my_command 在脚本中的其他地方已经读取。

# 打开一个 Telnet 会话到远程服务器，并等待用户名提示。
spawn telnet $remote_server
expect "username:"

# 发送用户名，并等待密码提示。
send "$my_user_id\r"
expect "password:"

# 发送密码，并等待 shell 提示符。
send "$my_password\r"
expect "%"

# 发送预先构建好的命令，并等待另一个 shell 提示符。
send "$my_command\r"
expect "%"

# 将命令的结果捕获到一个变量中。这可以显示出来，或写入到磁盘中。
set results $expect_out(buffer)

# 退出 Telnet 会话，并等待一个特殊的文件结束符。
send "exit\r"
expect eof
```

另一个例子是一个自动化FTP的脚本：
```shell
# 设置超时参数为合适的值。
# 例如，如果文件大小确实很大，并且网络速度确实是一个问题，最好将此参数设置为一个值。
set timeout -1

# 打开与远程服务器的FTP会话，并等待用户名提示。
spawn ftp $remote_server
expect "username:"

# 发送用户名，然后等待密码提示。
send "$my_user_id\r"
expect "password:"

# 发送密码，然后等待'ftp'提示符。
send "$my_password\r"
expect "ftp>"

# 切换到二进制模式，然后等待'ftp'提示符。
send "bin\r"
expect "ftp>"

# 关闭提示。
send "prompt\r"
expect "ftp>"

# 获取所有文件
send "mget *\r"
expect "ftp>"

# 退出FTP会话，并等待特殊的文件结束符。
send "bye\r"
expect eof
```

下面是一个自动化SFTP（使用密码）的示例
```shell
#!/usr/bin/env expect -f

# 尝试连接的过程；如果成功则返回0，否则返回1
proc connect {passw} {
  expect {
    "Password:" {
      send "$passw\r"
        expect {
          "sftp*" {
            return 0
          }
        }
    }
  }
  # 连接超时
  return 1
}

# 读取输入参数
set user [lindex $argv 0]
set passw [lindex $argv 1]
set host [lindex $argv 2]
set location [lindex $argv 3]
set file1 [lindex $argv 4]
set file2 [lindex $argv 5]

#puts "Argument data:\n";
#puts "user: $user";
#puts "passw: $passw";
#puts "host: $host";
#puts "location: $location";
#puts "file1: $file1";
#puts "file2: $file2";

# 检查是否提供了所有参数
if { $user == "" || $passw == "" || $host == "" || $location == "" || $file1 == "" || $file2 == "" }  {
  puts "用法: <用户> <密码> <主机> <位置> <要发送的文件1> <要发送的文件2>\n"
  exit 1
}

# Sftp到指定的主机并发送文件
spawn sftp $user@$host

set rez [connect $passw]
if { $rez == 0 } {
  send "cd $location\r"
  set timeout -1
  send "put $file2\r"
  send "put $file1\r"
  send "ls -l\r"
  send "quit\r"
  expect eof
  exit 0
}
puts "\n连接到服务器失败：主机：$host，用户：$user，密码：$passw！\n"
exit 1
```

使用密码作为命令行参数，就像在这个例子中一样，是一个巨大的安全漏洞，因为任何其他用户在机器上运行"ps"命令都可以读取到这个密码。然而，您可以添加代码，提示您输入密码，而不是将密码作为参数提供。这样应该更安全。请参考下面的示例：
```shell
stty -echo
send_user -- "Enter Password: "
expect_user -re "(.*)\n"
send_user "\n"
stty echo
set PASS $expect_out(1,string)
```

自动SSH登录到用户机器的另一个示例：
```shell
# Timeout是Expect中的一个预定义变量，默认设置为10秒。
# spawn_id是Expect中的另一个预定义变量。
# 关闭由spawn命令创建的spawn_id句柄是一个好习惯。
set timeout 60
spawn ssh $user@machine
while {1} {
  expect {
    eof                          {break}
    "The authenticity of host"   {send "yes\r"}
    "password:"                  {send "$password\r"}
    "*\]"                        {send "exit\r"}
  }
}
wait
close $spawn_id
```

## 替代方案
各种项目在其他语言中实现了类似Expect的功能，如C＃，Java，Scala，Groovy，Perl，Python，Ruby，Shell和Go。这些通常不是原始Expect的精确克隆，但概念往往非常相似。
- C＃
  - Expect.NET - 用于C＃（.NET）的Expect功能
  - DotNetExpect - 针对.NET的Expect风格的控制台自动化库
- Erlang
  - lux - 带有Expect风格执行命令的测试自动化框架
- Go
  - [GoExpect](https://github.com/google/goexpect) - 用于Go语言的类似Expect的包
  - [go-expect](https://github.com/Netflix/go-expect) - 用于Go语言的Expect风格库，用于自动控制基于终端或控制台的程序
- Groovy
  - expect4groovy - Expect工具的Groovy DSL实现
- Java
  - ExpectIt - Expect工具的纯Java 1.6+实现。它设计简单、易于使用且可扩展。
  - expect4j - 对原始Expect的Java克隆尝试
  - ExpectJ - Unix expect实用程序的Java实现
  - Expect-for-Java - Expect工具的纯Java实现
  - expect4java - Expect工具的Java实现，但支持嵌套闭包。还有适用于Groovy语言DSL的包装器。
- Perl
  - Expect.pm - Perl模块（最新版本在metacpan.org上）
- Python
  - Pexpect - 用于控制伪终端中交互式程序的Python模块
  - winpexpect - pexpect在Windows平台上的移植版
  - paramiko-expect - Paramiko SSH库的类似Expect的Python扩展，还支持日志追踪。
- Ruby
  - RExpect - 标准库中expect.rb模块的替代品。
  - Expect4r - 与Cisco IOS、IOS-XR和Juniper JUNOS CLI交互
- Rust
  - rexpect - 用于Rust语言的类似pexpect的包。
- Scala
  - scala-expect - Expect工具的Scala实现的一个非常小的子集。
- Shell
  - [Empty](http://empty.sourceforge.net) - 类似expect的实用程序，用于在Unix shell脚本中运行交互式命令
  - [sexpect](https://github.com/clarkwang/sexpect) - 用于shell的Expect。它以客户端/服务器模型实现，还支持附加/分离（类似GNU screen）。

## 引用
https://en.wikipedia.org/wiki/Expect

