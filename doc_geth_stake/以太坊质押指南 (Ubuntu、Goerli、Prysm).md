# 以太坊质押指南 (Ubuntu/Goerli/Prysm)
这是一个通过执行客户端和Prysm共识客户端在以太坊Goerli测试网络上进行质押的逐步指南。它基于以下技术：
- 服务器：Ubuntu v20.04（LTS）x64
- 执行客户端：Besu / Erigon / Geth / Nethermind
- 共识客户端：Prysmatic Labs Prysm
- 加密钱包：MetaMask
- Prometheus指标
- Grafana仪表板
- 官方多客户端公共测试网络，Goerli

## 先决条件
本指南假设您对以太坊、ETH、质押ETH、Linux和MetaMask有一定了解。

在开始之前，本指南还需要以下准备工作：

- 在本地计算机或云上安装并运行Ubuntu服务器v20.04（LTS）amd64。鼓励使用本地计算机以实现更大的去中心化。
- 在具有桌面（Mac、Windows、Linux等）和网络浏览器（Brave、Safari、FireFox等）的计算机上安装并配置MetaMask加密钱包的Web浏览器扩展程序。

## 要求
要获得良好的质押性能，通常需要满足以下要求。更多信息请参考这里和这里。
- 相对较新的多核CPU
- 8GB RAM（16GB更好，并在某些情况下是必需的）
- 至少1TB的SSD（建议使用2TB的NVMe）
- 稳定的互联网连接，具有足够的下载速度和每月的数据配额

> 注意：检查您的可用磁盘空间。即使您有一个大的SSD，有时Ubuntu报告的可用空间只有200GB。如果适用于您，请参考附录K-扩展逻辑卷。

## 概述
下面的简化图示说明了质押设置。黄色框表示本指南涵盖的领域。

<img src="./img/以太坊质押指南1.webp">

共识客户端（以前称为Eth2客户端）是为执行客户端提供PoS共识机制的软件。它包括Beacon链节点和验证器。

执行客户端（以前称为Eth1客户端）是负责各种以太坊网络操作的软件，例如从内存池中选择/执行交易。

> 注意：为了成功质押，需要同时使用共识客户端和执行客户端。

本指南的概念流程如下：
1. 生成质押验证器密钥和质押存款数据
2. 准备Ubuntu服务器（更新、防火墙、安全等）
3. 设置一个执行客户端节点并与以太坊区块链（Goerli测试网络）同步
4. 配置Prysm共识客户端并与其他Beacon节点同步
5. 存入Goerli测试网络的ETH以激活质押验证器密钥
6. 通过Prometheus度量和Grafana仪表板设置服务器监控

## 步骤1 — 生成质押数据
为了参与Goerli测试网络的质押，需要使用工具根据您想要资助和运营的验证者数量生成数据文件。每个验证者需要存入32个Goerli测试网络ETH才能在Goerli测试网络上激活。

> 注意：每个人可以使用的Goerli测试网络ETH数量有限。目前，最实际的做法是最多资助一个到两个验证者。

在本指南的后面，设置质押软件后将提交存款来资助您的验证者。获取Goerli测试网络ETH的说明将在那里提供。

### 下载存款工具（Staking Deposit CLI）
点击[这里](https://github.com/ethereum/eth2.0-deposit-cli/releases/)获取最新版本的质押存款命令行界面（CLI）工具。

<img src="./img/以太坊质押指南2.webp">

在**Assets**部分找到与所需平台匹配的版本。如果是Windows系统，请右键点击链接并下载。如果是Linux系统，请使用下面的命令下载压缩文件。

修改下面的URL以匹配最新版本的下载链接。
```shell
$ cd ~
$ curl -LO https://github.com/ethereum/staking-deposit-cli/releases/download/v2.3.0/staking_deposit-cli-76ed782-linux-amd64.tar.gz
```

如果是Windows系统，请解压缩该压缩文件并进入创建的文件夹。如果是Linux系统，请使用下面的命令解压缩tar压缩文件，并进入所创建的目录。

修改文件名以匹配已下载的版本。
```shell
$ tar xvf staking_deposit-cli-76ed782-linux-amd64.tar.gz
$ cd staking_deposit-cli-76ed782-linux-amd64
```

在压缩文件中应该有一个名为"deposit"的二进制文件（可执行文件）。

### 准备运行存款工具（质押存款CLI）
存款工具会生成一个助记词密钥。为了避免密钥泄露的风险，必须安全地处理该密钥 —— 对于测试网络来说可能不是很重要，但现在是练习主网的好时机。从这里开始，有两个选项可以选择。

- 选项1：离线机器（推荐） —— 将二进制文件复制到USB驱动器。将驱动器连接到完全离线的机器上（从未连接到网络或互联网）。将二进制文件复制到离线机器上。

- 选项2：当前机器（不推荐） —— 从当前机器上运行。互联网连接可能会导致助记词密钥泄露的风险。如果没有完全离线的机器可用，请在继续之前断开当前机器的网络/互联网连接。

### 运行存款工具（质押存款命令行界面）
在安全的机器上，在终端窗口（或Windows中的CMD）中运行二进制文件。例如，如果您想使用提款地址<YourWithdrawalAddress>创建2个验证人，请使用以下命令。

Linux:
```shell
$ ./deposit new-mnemonic --num_validators 2 --chain goerli --eth1_withdrawal_address <YourWithdrawalAaddress>
```

Windows:
```shell
deposit.exe new-mnemonic --num_validators 2 --chain goerli --eth1_withdrawal_address <YourWithdrawalAaddress>
```

请将<YourWithdrawalAddress>替换为您控制的Goerli测试网络以太坊地址。

> ***注意：如果您需要Goerli测试网络的ETH来为您的验证者提供资金，您必须将提款地址设置为0x4D496CcC28058B1D74B7a19541663E21154f9c84。这是EthStaker Goerli Bot的官方地址。每个钱包地址只允许进行2次存款，因此您应该只创建2个验证者（64个Goerli ETH）。如果您已经有足够的Goerli ETH，则可以忽略此信息。重要的是，此指令仅适用于测试网络设置。对于Mainnet，您应该使用您控制的地址。请在[EthStaker](https://discord.io/ethstaker) Discord的#cheap-goerli-validator频道中获取更多信息。***
> 
> ***注意：一旦设置，提款地址将无法更改，所以请务必确保它是您控制和正确指定的地址。例如：
> --eth1_withdrawal_address 0x4D496CcC28058B1D74B7a19541663E21154f9c84***
> 
> ***注意：标志--eth1_withdrawal_address允许您指定一个Goerli测试网络以太坊地址，您在超过32个质押的Goerli测试网络ETH的收益将自动提款到该地址（一旦启用提款）。这也是您退出验证者时32个质押的Goerli测试网络ETH将提款到的地址。更多信息请参阅[此处](https://notes.ethereum.org/@launchpad/withdrawals-faq)。***
> 
> ***注意：如果您当前不设置--eth1_withdrawal_address标志，您可以在后续的特殊流程中设置它（称为将提款凭据从0x00转换为0x01），当您准备开始提取质押收益或者想要退出验证者时。如果您不设置该标志，质押收益将不会自动提款，并且在退出验证者之前，您将无法取回32个质押的Goerli测试网络ETH，直到您转换提款凭据。***

一旦您在您选择的平台上执行了上述步骤并提供了您的语言偏好，您将被要求创建验证器密钥库的密码。请将其备份到安全的地方。您稍后需要使用它将验证器密钥加载到共识客户端验证器钱包中。

<img src="./img/以太坊质押指南3.webp">

接下来，将生成一个种子短语（助记词）。请将其备份到安全的地方。这非常重要。您最终将使用它来生成您在Goerli测试网络上抵押的ETH的提款密钥，或者用于添加额外的验证器。

> 注意：如果您丢失了助记词，将无法提取您的资金。

<img src="./img/以太坊质押指南4.webp">

一旦您确认了您的助记词，您的验证器密钥将被创建。

<img src="./img/以太坊质押指南5.webp">

验证器密钥和存款数据文件已在指定位置创建。目录的内容如下所示。

<img src="./img/以太坊质押指南6.webp">

关于这些文件的说明：
- deposit_data-[timestamp].json 文件包含验证器的公钥和抵押存款的信息。此文件将在本指南后面的步骤中用于完成Goerli测试网络ETH的抵押存款过程。
- keystore-[..].json 文件包含加密的验证器签名密钥。每个您要资助的验证器都有一个keystore文件。这些文件将在共识客户端验证器钱包中导入，用于验证操作。
- 您将在本指南后面将这些文件复制到您的Ubuntu服务器上（如果尚未存在）。
- 如果您丢失或意外删除了这些文件，可以使用抵押存款工具和您的助记词通过existing-mnemonic命令重新生成它们。更多信息请参阅此处。

### 最后的步骤
现在您已经生成了存款数据和keystore文件、验证器密码和助记词，请继续设置Ubuntu服务器。

此刻请不要存入任何Goerli测试网络ETH。

首先完成并验证您的质押设置非常重要。如果Goerli测试网络ETH存款变得有效，并且您的质押设置尚未准备好，将会从您质押的Goerli测试网络ETH余额中扣除不活跃惩罚。

## 步骤2 - 连接到服务器
使用SSH客户端连接到您的Ubuntu服务器。如果您已经以root用户登录，请创建一个具有管理员权限的用户级账户，因为以root用户登录是有风险的。

> 注意：如果您没有以root用户登录，则跳过此步骤，进入步骤3。

创建一个新用户。将 `<yourusername>` 替换为您选择的用户名。您将被要求创建一个强密码并提供其他一些可选信息。

```shell
adduser <yourusername>
```

通过将新用户添加到sudo组中，授予新用户管理员权限。这将允许用户在命令前键入sudo以使用超级用户权限执行操作。
```shell
usermod -aG sudo <yourusername>
```

可选：如果您使用SSH密钥通过root用户连接到Ubuntu实例，则需要将新用户与root用户的SSH密钥数据关联起来。
```shell
rsync --archive --chown=<yourusername>:<yourusername> ~/.ssh /home/<yourusername>

```

最后，退出root用户并以<yourusername>用户登录。

## 步骤3 - 更新服务器
确保您的系统使用最新的软件和安全更新。
```shell
$ sudo apt -y update && sudo apt -y upgrade
$ sudo apt dist-upgrade && sudo apt autoremove
$ sudo reboot
```

## 步骤4 — 保护服务器
安全是重要的。这不是一份全面的安全指南，只是一些基本的设置。

### 修改默认的SSH端口
端口22是默认的SSH端口，也是一种常见的攻击向量。为了避免这种情况，可以更改SSH端口。

选择一个1024–49151之间的端口号，并运行以下命令来检查该端口是否已被使用。
```shell
$ sudo ss -tulpn | grep ':YourSSHPortNumber'
```

示例：
```shell
$ sudo ss -tulpn | grep ':6673'
```
如果返回空白，则表示该端口未被使用；如果返回红色文本，则表示该端口已被使用：请尝试使用其他端口号。

如果确认端口可用，请通过更新服务器的SSH配置文件来修改默认的SSH端口号。打开配置文件：
```shell
$ sudo nano /etc/ssh/sshd_config
```

在文件中找到或添加（如果不存在）以下行：Port 22。如果有“#”符号，请将其删除，并按照下面的示例更改数值。
```shell
Port YourSSHPortNumber
```

请参考下面的截图。

<img src="./img/以太坊质押指南7.webp">

按下<CTRL> + X，然后按下Y，然后按下<ENTER>以保存并退出。

重新启动SSH服务以反映更改。
```shell
$ sudo systemctl restart ssh
```

记得更新您的SSH客户端设置以反映您配置的新SSH端口。注销并使用`YourSSHPortNumber`作为端口号通过SSH重新登录，以确保一切正常运行。

### 配置防火墙
Ubuntu 20.04服务器可以使用UFW防火墙来限制对服务器的入站流量。防火墙有助于防止不必要的连接到您的服务器。

#### 安装UFW
UFW应该已默认安装。以下命令将确保其已安装。
```shell
$ sudo apt install ufw
```

#### 应用UFW默认设置
明确应用默认设置：拒绝入站流量，允许出站流量。
```shell
$ sudo ufw default deny incoming
$ sudo ufw default allow outgoing
```

#### 允许SSH
允许在上面设置的YourSSHPortNumber上的入站流量。SSH需要使用TCP协议。

> 注意：如果您正在本地托管Ubuntu实例并希望远程访问服务器（出于安全原因不建议），您的互联网路由器可能需要配置以允许端口YourSSHPortNumber上的传入流量。

```shell
$ sudo ufw allow YourSSHPortNumber/tcp
```

示例：
```shell
$ sudo ufw allow 6673/tcp
```

#### 拒绝SSH端口22
如果您已更改SSH端口的值，则拒绝默认端口22/TCP的入站流量。
```shell
$ sudo ufw deny 22/tcp
```

#### 允许执行客户端端口30303
允许与执行客户端节点（端口30303）建立P2P连接。这是本指南中所有执行客户端常用的端口。

> 注意：如果您正在本地托管Ubuntu实例，则您的互联网路由器可能还需要配置以允许端口30303上的传入流量。

```shell
$ sudo ufw allow 30303
```

#### 允许Prysm
允许与共识客户端节点进行P2P连接，以执行Beacon Chain节点上的操作（端口13000/TCP和12000/UDP）。

> 注意：如果您在本地托管Ubuntu实例，则您的互联网路由器可能需要配置以允许端口13000/TCP和12000/UDP的传入流量。

```shell
$ sudo ufw allow 13000/tcp
$ sudo ufw allow 12000/udp
```

#### 允许Grafana
允许对Grafana Web服务器的请求进行传入访问（端口3000/TCP）。

> 注意：如果您在本地托管Ubuntu实例并希望远程访问Grafana仪表板（出于安全原因不建议），您的互联网路由器可能还需要配置以允许端口3000的传入流量。

```shell
$ sudo ufw allow 3000/tcp
```

### 启用防火墙
启用防火墙并验证规则是否已正确配置。

```shell
$ sudo ufw enable
$ sudo ufw status numbered
```

请参考下方的屏幕截图。

<img src="./img/以太坊质押指南8.webp">

注销并再次通过SSH登录，以确认一切是否正常运行。

























