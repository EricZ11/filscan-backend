
# Filscan_Lotus - LotusExplorer

## 运行
#### 环境：
golang >= v1.13

mongo >= v4.2

系统环境 linux 或 mac 暂不支持windows

一个运行完备的lotus节点

以及编译lotus需要的其他依赖

```cassandraql
git clone (githuburl)
cd Backend
make build-lotus
go build 
```
此时应生成 filscan_lotus 可执行文件

#### 配置

打开 conf下app.conf文件
```cassandraql
mongoHost = "127.0.0.1:27017"
mongoUser = "root"
mongoPass = "admin"
mongoDB   = "filscan"
lotusGetWay="192.168.1.111:1234"
```
配置mongo服务，以及lotus节点IP(注意:lotus节点1234端口默认不对外开放需自行配置)

#### 运行

运行目录结构
```cassandraql
|-- conf
   |-- app.conf
|-- filscan_lotus

```
运行指令
```cassandraql
./filscan_lotus
``` 

运行启动将会检查mondo以及lotus连通性。连接失败系统将终止启动。

若检测成功系统将正常启动，第一启动需要同步链上数据，请耐心等待。