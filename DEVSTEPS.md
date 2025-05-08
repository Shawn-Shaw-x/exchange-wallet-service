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

## 3. rpc 搭建

## 4. 数据库设计、gorm 搭建

## 5. 扫链同步器搭建

## 6. 充值业务实现

## 7. 提现业务实现

## 8. 归集业务实现

## 9. 热转冷、冷转热业务实现

## 10. 接口实现

## 11. 通知业务实现

