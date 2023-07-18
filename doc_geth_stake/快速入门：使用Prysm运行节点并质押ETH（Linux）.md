# 快速入门：使用Prysm运行节点并质押ETH
## 配置
- Operating system : Linux
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
```shell
mkdir prysm && cd prysm
curl https://raw.githubusercontent.com/prysmaticlabs/prysm/master/prysm.sh --output prysm.sh && chmod +x prysm.sh
```

这将下载Prysm客户端并更新您的注册表以启用详细日志记录。

## 步骤3：运行一个执行客户端。
在这一步中，您将安装一个执行层客户端，Prysm的信标节点将连接到该客户端。

从Geth下载页面下载并运行适用于您操作系统的最新的64位稳定版本的Geth安装程序。

转到您的执行目录，并运行以下命令启动您的执行节点：
```shell
geth --goerli --http --http.api eth,net,engine,admin 
```

请参考Geth的命令行选项以了解参数定义。

同步可能需要很长时间，从几个小时到几天不等。在您的执行节点同步期间，您可以继续进行下一步操作。

恭喜您！您现在正在运行以太坊执行层的一个执行节点。

## 第四步：使用Prysm运行信标节点
在这一步中，您将使用Prysm运行一个信标节点。

从Github下载Prater的创世状态文件到您的consensus/prysm目录中。然后使用以下命令启动一个连接到本地执行节点的信标节点：
```shell
./prysm.sh beacon-chain --execution-endpoint=$HOME/.ethereum/<your.ipc> --prater --genesis-state=genesis.ssz --suggested-fee-recipient=0x01234567722E6b0000012BFEBf6177F1D2e9758D9
```

如果您正在运行一个验证者节点，可以指定一个建议的手续费接收钱包地址，以便您可以获得以前矿工交易费小费。有关此功能的更多信息，请参阅如何配置手续费接收者。

您的信标节点现在将开始同步。这通常需要几天的时间，但根据您的网络和硬件规格，可能需要更长的时间。

恭喜您！您现在正在运行一个完整的、准备好合并的以太坊节点。要检查节点的状态，请访问"Check node and validator status"。

## 第五步：使用Prysm运行验证者节点
接下来，我们将使用以太坊质押存款CLI创建您的验证者密钥。

从质押存款CLI发布页面下载最新稳定版本的存款CLI。

运行以下命令来创建您的助记词和密钥：
```shell
./deposit new-mnemonic --num_validators=1 --mnemonic_language=english --chain=prater
```

按照CLI提示生成您的密钥。这将给您以下文件：

1. 一个新的助记词种子短语。这是非常敏感的信息，绝不能暴露给其他人或网络设备。
2. 一个validator_keys文件夹。该文件夹将包含两个文件：
   - deposit_data-.json - 包含您稍后将上传到以太坊发射台的存款数据。
   - keystore-m_.json - 包含您的公钥和加密的私钥。

将validator_keys文件夹复制到您主机的consensus文件夹中。运行以下命令导入您的密钥库，将<YOUR_FOLDER_PATH>替换为consensus/validator_keys文件夹的完整路径：
```shell
./prysm.sh validator accounts import --keys-dir=<YOUR_FOLDER_PATH> --prater
```

您将被提示两次指定钱包目录。在两个提示中都提供您的共识文件夹的路径。当您的账户成功导入到Prysm后，您应该会看到成功导入了1个账户，通过运行accounts list命令可以查看所有账户。

接下来，转到Goerli-Prater Launchpad的存款数据上传页面，并上传您的deposit_data-*.json文件。您将被提示连接您的钱包。

如果您需要GöETH，请前往以下其中一个Discord服务器：
- r/EthStaker Discord
- Prysm Discord服务器

有人应该能够提供您所需的GöETH。然后，您可以通过Launchpad页面将32个GöETH存入Prater测试网的存款合约。在整个过程中要非常小心，绝不要向测试网存款合约发送真正的ETH。最后，运行以下命令启动您的验证者，将<YOUR_FOLDER_PATH>替换为共识文件夹的完整路径：

```shell
./prysm.sh validator --wallet-dir=<YOUR_FOLDER_PATH> --prater
```

您的验证者完全激活可能需要很长时间（从几天到几个月不等）。要了解有关验证者激活过程的更多信息，请参阅存款流程。有关详细的状态监控指南，请参阅检查节点和验证者状态。

您可以保持执行客户端、信标节点和验证者客户端终端窗口保持打开和运行。一旦您的验证者被激活，它将自动开始提议和验证区块。









