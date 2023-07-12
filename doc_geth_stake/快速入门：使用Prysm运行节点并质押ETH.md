# 快速入门：使用Prysm运行节点并质押ETH
## 配置
- Operating system : Windows
- Network : Goerli-Prater
- Execution client : Geth
- EN-BN connection : IPC

# 介绍
Prysm是以太坊权益证明共识规范的一个实现。在这个快速入门中，您将使用Prysm来运行一个以太坊节点，并可选择运行一个验证器。这将让您使用自己管理的硬件质押32个ETH。

这是一个面向初学者的指南。我们期望您熟悉命令行，但除此之外，本指南不假设您具备任何技术技能或先前知识。

在一个高层次上，我们将按照以下步骤进行：

1. 使用执行层客户端配置执行节点。
2. 使用Prysm（一个共识层客户端）配置信标节点。
3. 使用Prysm配置验证器并质押ETH（可选）。

## 步骤1：审查先决条件和最佳实践
| Node type	 | Benefits	                                                                                                                                                                                           |Requirements|
|------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---|
| 执行层 + 信标层  | - 有助于保障以太坊生态系统的安全性。<br>- 让您直接访问以太坊网络，无需依赖第三方服务的信任。<br>- 让您在合并后运行验证器。|- 软件要求：执行客户端（execution client）,信标节点客户端（beacon node client）（下面提供了客户端的安装说明）,curl<br> - 操作系统要求：64位Linux,Mac OS X 10.14+,Windows 10+ 64位<br> - CPU要求：4个以上的核心，主频2.8GHz以上<br> - 内存要求：16GB以上的RAM<br> - 存储要求：至少2TB的可用空间的SSD<br> - 网络要求：宽带连接，最低8 MBit/sec的带宽|
| 验证器层	 |让您可以质押ETH，提议和验证区块，获得权益奖励和交易手续费小费。| 除了之前提到的内容外，还需要以下内容：<br> - 软件要求：验证器客户端（validator client）,基于浏览器的加密钱包（下面提供了钱包的安装说明）<br> - 硬件要求：（推荐）一台全新的机器，从未连接过互联网，用于安全地生成助记词和密钥对<br> - 资产要求：32个ETH（主网）,32个测试网络ETH（测试网）|

### 最佳实践：
1. 如果您作为验证器质押ETH，请先在测试网络上尝试此指南，然后再尝试在主网上进行。
2. 保持简单。本指南假设所有客户端软件都在一台机器上运行。
3. 查看您使用的网络（如Goerli-Prater、主网）的最新警报和建议。
4. 查阅我们发布的所有安全最佳实践。
5. 加入社区 - 加入我们的邮件列表、Prysm Discord服务器、r/ethstaker（Reddit社区）和EthStaker Discord服务器，获取更新和支持。

## 步骤2: 安装 Prysm
在您的SSD上创建一个名为ethereum的文件夹，然后在其中创建两个子文件夹：consensus和execution。

进入您的consensus目录并运行以下命令：

    mkdir prysm && cd prysm
    curl https://raw.githubusercontent.com/prysmaticlabs/prysm/master/prysm.bat --output prysm.bat
    reg add HKCU\Console /v VirtualTerminalLevel /t REG_DWORD /d 1

这将下载Prysm客户端并更新您的注册表以启用详细日志记录。

## 步骤3：运行一个执行客户端。
在这一步中，您将安装一个执行层客户端，供Prysm的信标节点连接。

从Geth下载页面下载并运行适用于您操作系统的最新64位稳定版本的Geth安装程序。

进入您的执行目录，并运行以下命令启动执行节点：

    geth --goerli --http --http.api eth,net,engine,admin 

请参阅Geth的命令行选项以获取参数定义。

同步可能需要很长时间 - 从几小时到几天不等。您可以在执行节点同步的同时继续进行下一步。

恭喜！您现在正在运行以太坊执行层中的执行节点。

## 步骤4：使用Prysm运行信标节点。
在这一步中，您将使用Prysm运行信标节点。

从Github下载Prater的创世状态文件到您的consensus/prysm目录中。然后使用以下命令启动一个连接到本地执行节点的信标节点：

> --http-web3provider已被弃用，并已被--execution-endpoint取代，但是在Windows上，IPC目前仅通过--http-web3provider工作。这个问题将在我们的下一个版本中修复。在此期间，您可以安全地忽略任何与“deprecated flag”相关的警告。

    prysm.bat beacon-chain --http-web3provider=//./pipe/<your.ipc> --prater --genesis-state=genesis.ssz --suggested-fee-recipient=0x01234567722E6b0000012BFEBf6177F1D2e9758D9

如果您正在运行一个验证器，指定一个建议的手续费接收地址将允许您获得以前的矿工交易手续费小费。有关此功能的更多信息，请参阅如何配置手续费接收方。

您的信标节点现在将开始同步。这通常需要几天时间，但根据您的网络和硬件规格，可能需要更长的时间。

恭喜！您现在正在运行一个完整的、准备好合并的以太坊节点。要检查节点的状态，请访问“检查节点和验证器的状态”。

## 步骤5：使用Prysm运行验证器。
接下来，我们将使用以太坊Staking Deposit CLI创建您的验证器密钥。

从Staking Deposit CLI Releases页面下载最新稳定版本的deposit CLI。

运行以下命令来创建您的助记词（一个唯一且高度敏感的24个单词短语）和密钥：

    deposit.exe new-mnemonic --num_validators=1 --mnemonic_language=english --chain=prater

请按照CLI提示生成您的密钥。这将给您以下文件：
1. 一个新的助记词种子短语。这是非常敏感的信息，不应该暴露给其他人或网络硬件。
2. 一个validator_keys文件夹。该文件夹包含两个文件：
   1. deposit_data-*.json - 包含您将稍后上传到以太坊启动器的存款数据。
   2.  keystore-m_*.json - 包含您的公钥和加密的私钥。

将validator_keys文件夹复制到您的主要机器的consensus文件夹中。运行以下命令导入您的密钥库，将<YOUR_FOLDER_PATH>替换为consensus/validator_keys文件夹的完整路径：

    prysm.bat validator accounts import --keys-dir=<YOUR_FOLDER_PATH> --prater



