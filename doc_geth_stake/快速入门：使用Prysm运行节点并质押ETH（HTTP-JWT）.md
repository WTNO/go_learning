# 快速入门：使用Prysm运行节点并质押ETH
## 配置
- Operating system : Windows
- Network : Goerli-Prater
- Execution client : Geth
- EN-BN connection : HTTP-JWT

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

### 生成JWT密钥
在Beacon节点和执行节点之间的HTTP连接需要使用JWT令牌进行身份验证。有几种方法可以生成这个JWT令牌：
1. 使用类似OpenSSL的工具通过命令生成令牌：openssl rand -hex 32 | tr -d "\n" > "jwt.hex"。
2. 使用执行客户端生成jwt.hex文件。
3. 使用Prysm生成jwt.hex文件：

```shell
## Optional. This command is necessary only if you've previously configured USE_PRYSM_VERSION
SET USE_PRYSM_VERSION=v4.0.0

## Required.
prysm.bat beacon-chain generate-auth-secret
```

Prysm将输出一个jwt.hex文件的路径。

> 注意：  
> 确保用于创建和访问JWT令牌的脚本、用户或终端窗口具有所需的权限。Windows用户可能需要以管理员身份运行命令窗口。

本指南假设您已将jwt.hex文件放置在共识目录中，但您可以将其放置在任何位置，并根据需要修改以下命令。

## 步骤3：运行一个执行客户端。
在这一步中，您将安装一个执行层客户端，Prysm的Beacon节点将连接到该客户端。

从Geth下载页面下载并运行适用于您操作系统的最新的64位稳定版本的Geth安装程序。

导航到您的执行目录，并运行以下命令启动执行节点：

    geth --goerli --http --http.api eth,net,engine,admin --authrpc.jwtsecret /path/to/jwt.hex 

请参阅Geth的命令行选项以获取参数定义。

同步可能需要很长时间 - 从几个小时到几天不等。您可以在执行节点同步的同时进行下一步操作。

恭喜您 - 您现在正在以太坊的执行层中运行一个执行节点。

## 步骤4：使用Prysm运行信标节点。
在这一步中，您将使用Prysm运行一个Beacon节点。

从Github下载Prater创世状态文件并将其放置在您的consensus/prysm目录中。然后使用以下命令启动一个连接到本地执行节点的Beacon节点：
```shell
prysm.bat beacon-chain --execution-endpoint=http://localhost:8551 --prater --jwt-secret=path/to/jwt.hex --genesis-state=genesis.ssz --suggested-fee-recipient=0x01234567722E6b0000012BFEBf6177F1D2e9758D9
```

如果您正在运行验证器，指定一个建议的费用接收者钱包地址将使您能够获得以前的矿工交易费小费。有关此功能的更多信息，请参阅如何配置费用接收者。

您的Beacon节点现在开始同步。这通常需要几天的时间，但根据您的网络和硬件规格，可能需要更长的时间。

恭喜您 - 您现在正在运行一个完整的、准备好合并的以太坊节点。要检查节点的状态，请访问检查节点和验证器状态。

## 步骤5：使用Prysm运行验证器。
接下来，我们将使用以太坊质押存款命令行界面创建您的验证器密钥。

请从质押存款CLI版本发布页面下载最新稳定版本的存款CLI。

运行以下命令来创建您的助记词（一个独特且高度敏感的24个单词短语）和密钥：

    deposit.exe new-mnemonic --num_validators=1 --mnemonic_language=english --chain=prater

请按照CLI提示生成您的密钥。这将给您以下文件：
1. 一个新的助记词种子短语。这是非常敏感的信息，不应该暴露给其他人或网络硬件。
2. 一个validator_keys文件夹。该文件夹包含两个文件：
    1. deposit_data-*.json - 包含您将稍后上传到以太坊启动器的存款数据。
    2.  keystore-m_*.json - 包含您的公钥和加密的私钥。

将validator_keys文件夹复制到您的主要机器的consensus文件夹中。运行以下命令导入您的密钥库，将<YOUR_FOLDER_PATH>替换为consensus/validator_keys文件夹的完整路径：

    prysm.bat validator accounts import --keys-dir=<YOUR_FOLDER_PATH> --prater

您将被要求两次指定钱包目录。对于这两个提示，请提供consensus文件夹的路径。当您的账户成功导入到Prysm后，您应该会看到“成功导入1个账户，通过运行accounts list查看所有账户”。

接下来，前往Goerli-Prater Launchpad的存款数据上传页面，上传您的deposit_data-*.json文件。您将被要求连接您的钱包。

如果您需要GöETH，请前往以下其中一个Discord服务器：
- r/EthStaker Discord
- Prysm Discord server

有人应该能够为您提供所需的 GöETH。然后，您可以通过Launchpad页面将32个GöETH存入Prater测试网络的存款合约中。在整个过程中要非常小心，永远不要将真正的ETH发送到测试网络的存款合约中。最后，运行以下命令以启动您的验证器，将<YOUR_FOLDER_PATH>替换为您共识文件夹的完整路径：

      prysm.bat validator --wallet-dir=<YOUR_FOLDER_PATH> --prater

> 恭喜！您现在已经运行了一个完整的以太坊节点和一个验证器。

您的验证器要完全激活可能需要很长时间（从几天到几个月不等）。要了解有关验证器激活过程的更多信息，请参阅存款过程。请参阅检查节点和验证器状态以获取详细的状态监控指南。

您可以保持执行客户端、信标节点和验证器客户端终端窗口保持打开和运行。一旦您的验证器被激活，它将自动开始提议和验证区块。

> 获取goerli测试币：
> 
> 每次只需支付0.0001 GoETH即可进行Goerli验证人存款。首先调用“/cheap-goerli-deposit”斜杠命令以获得白名单，然后按照机器人的指示进行操作。您需要开始输入斜杠命令，它将显示在您可以使用的输入框上方。请在https://goerli.launchpad.ethstaker.cc/上使用我们的自定义发射台。
>
> 由于我们现在强制提现地址必须是由我们的机器人控制，您无法从这个过程中获得实质性利润。
> 
> 您现在可以在https://goerli.launchpad.ethstaker.cc/上使用您的钱包地址0x840348C11C2B2Da494C8246c938fd20cF14ccca2进行2次廉价存款。请确保查阅指南和工具，以配置您的设备在Goerli上运行验证人。
> 
> 在创建您的验证人密钥和存款文件时，您必须将提现地址设置为0x4D496CcC28058B1D74B7a19541663E21154f9c84，以便使用此过程并完成存款。这仅适用于此发射台。在Mainnet上，如果您想使用提现地址，您应该使用您自己控制的地址。
> 
> 在高燃气费时，执行此存款交易可能会比实际廉价存款成本0.0001 Goerli ETH更昂贵。如果您遇到这种情况，您可以尝试从https://faucetlink.to/goerli获取更多的Goerli ETH，或者等待燃气费降低（请参阅https://goerli.beaconcha.in/gasnow以监控Goerli上的燃气费），或者使用自定义低燃气费广播您的交易，并等待网络接收。