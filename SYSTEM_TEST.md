## 1. RPC 服务接口测试

- **业务方注册**：
    1. 业务方携带自己的 `requestId` 进行注册，系统会根据 `requestId` 为其生成独立的 `address`、`balance`、`transactions`、`deposits`、`withdraw`、`internal`、`tokens` 表
    2. 注册成功后，其所有业务都需要携带 `requestId` 进行请求，数据独立在其自己的表中。
    ```
  	BusinessRegister(context.Context, *BusinessRegisterRequest) (*BusinessRegisterResponse, error)
   ```
- **批量导出地址**：
    1. 业务方通过 “`signature-machine`” 项目（项目方自己部署，自己掌控私钥和签名流程）批量生成公钥，将公钥传入此接口，批量获取地址。
    2. 此接口中，会根据用户方传入的地址类型，保存该地址信息到 `address_{requestId}` 表中,并初始化 `balances`
    ```
  	ExportAddressesByPublicKeys(context.Context, *ExportAddressesRequest) (*ExportAddressesResponse, error)
  ```
- **构建未签名交易**：
    1. 在此接口中，业务方传入关键参数：`from`、`to`、`amount`、`chainId` 等信息，调用该接口。该接口会调用 “`chains-union-rpc`” 项目去获取地址的 `nonce`、`gasFee` 等。
    2. 然后构建 `EIP-1159` 的交易，调用 “`chains-union-rpc`” 项目去构建交易，返回 `16` 进制的未签名交易 `messageHash`（`32` 字节）、将交易信息保存在表中。返回 `messageHash` 和请求的 `transactionId`
    ```
      BuildUnSignTransaction(context.Context, *UnSignTransactionRequest) (*UnSignTransactionResponse, error)
  ```
- **构建已签名交易**：
    1. 项目方持有上述的未签名交易的 `messageHash`，调用 “`signature-machine`” 使用该交易对应的 `from` 地址私钥进行对此 `messageHash` 签名，返回 `signature` （`65` 字节） 信息
    2. 项目方拿到 `signature`、`transactionId`。 由 `transactionId` 从表中查出这笔交易，然后重新构造出来相同交易。调用 “`chains-union-rpc`” 的构建已签名接口，使用 `signature` 和 原交易信息发起调用 `BuildSignedTransaction`接口。
    3. 在“`chains-union-rpc`”中，会将 `signature` 拆分出 `r、s、v` 值和原交易组合起来，格式化返回一个已签名的交易（`16` 进制，`base64` 编码）
    4. 在拿到这个已签名交易的 `16` 进制数据后，即可调用 “`chains-union-rpc`” 里面的 `sendTx` 接口，将这笔交易公布到 `rpc` 网络中即可
    ```
  	BuildSignedTransaction(context.Context, *SignedTransactionRequest) (*SignedTransactionResponse, error)
  ```
  
- **联调** `exchange-wallet-service`、`signature-machine`、 `chains-union-rpc` **三个项目**
    1. exchange-wallet-service 业务方注册

  ![img.png](images/businessRegistRequest.png)
  ![img.png](images/businessRegistResponse.png)

    2. signature-machine 批量公钥生成

  ![img.png](images/keyPairRequest.png)
  ![img.png](images/keyPairResponse.png)

    3. exchange-wallet-service 公钥转地址

  ![img.png](images/addressRequest.png)
  ![img.png](images/addressResponse.png)

    4. 转资金进这个地址

  ![img.png](images/metamask.png)

    5. exchange-wallet-service 构建未签名交易

  ![img.png](images/unsignTransactionRequest.png)
  ![img.png](images/unsignTransactionResponse.png)

    6. signature-machine 中签名操作

  ![img.png](images/signatureRequest.png)
  ![img.png](images/signatureResponse.png)

    7. exchange-wallet-service 构建已签名交易

  ![img.png](images/signedTxRequest.png)
  ![img.png](images/signedTxResponse.png)

    8. chains-union-rpc 发送出去交易

  ![img.png](images/sendRequest.png)
  ![img.png](images/sendResponse.png)

    9. holesky 区块浏览器中查看这笔交易

  ![img.png](images/success.png)

## 2. 扫链同步器扫块测试
- 启动扫链同步器服务

![img.png](images/scanBlocksRequest.png)
![img_1.png](images/scanBlocksResponse.png)

## 3. 充值业务测试