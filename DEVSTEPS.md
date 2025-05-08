## 1. 项目搭建
- 新建项目 exchange-wallet-service
- 目录如下：
```
├── cmd                 主程序入口、命令行程序框架
├── common              通用工具库
├── config              配置文件管理代码
├── database            数据库代码
├── flags               环境变量管理代码
├── migrations          数据库迁移
├── notifier            回调通知管理
├── protobuf            grpc 接口及生成代码
├── rpcclient           grpc 连接客户端
├── services            grpc 服务管理及接口实现
├── sh                  shell 命令
├── worker              核心工作代码（充值、提现、归集、热转冷）
├── exchange.go         主程序生命周期管理
├── Makefile  shell     命令管理
├── devops.md           开发步骤
├── go.mod              依赖管理
├── README.md         
  ```
## 2. 控制台应用整合
- main.go
```
func main() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stdout, log.LevelInfo, true)))
	app := NewCli(GitCommit, GitData)
	ctx := opio.WithInterruptBlocker(context.Background())
	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Error("Application failed")
		os.Exit(1)
	}
}
```
- cli.go
```
func NewCli(GitCommit string, GitData string) *cli.App {
	//flags := flags2.Flags
	return &cli.App{
		Version:              params.VersionWithCommit(GitCommit, GitData),
		Description:          "An exchange wallet scanner services with rpc and rest api server",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:        "version",
				Description: "Show project version",
				Action: func(ctx *cli.Context) error {
					cli.ShowVersion(ctx)
					return nil
				},
			},
		},
	}
}
```

## 3. 数据库设计、gorm 搭建
- Business(businessId,notifyUrl...): 注册商户表
- Blocks(hash,parentHash,number...): 区块信息表
- ReorgBlocks(hash,parentHash,number): 回滚区块表（回滚时处理交易使用）
- Address(address,addressType,publicKey...): 钱包地址表
- Balance(address,tokenAddress,balance,lockBalance...): 地址余额表
- Deposit(from,to,amount,confirms,blockHash...): 充值表
- Withdraw(from,to,amount,blockHash...): 提现表
- Internals(from,to,amount,blockHash...): 内部交易表（热转冷、冷转热）
- Transactions(from,to,amount,fee,hash...): 交易流水表
- Token(tokenAddress,decimals,collectAmount...): token合约表

- 数据库迁移脚本：`migrations` 文件夹中 
- 执行数据库迁移：执行 `make` 编译程序，然后 `./exchange-wallet-service migrate`
- 实现每一个表对应结构体、新增表、增删改查接口
## 4. rpc 搭建
- 编写 `exchange-wallet.proto`文件，定义消息和接口
- `make protogo` 生成对应的 protobuf 代码
-
## 5. 扫链同步器搭建

## 6. 充值业务实现

## 7. 提现业务实现

## 8. 归集业务实现

## 9. 热转冷、冷转热业务实现

## 10. 接口实现

## 11. 通知业务实现

