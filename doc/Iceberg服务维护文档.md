# 前言
本文档描述Iceberg体系中服务程序通用的部署和维护说明。Iceberg的运行会依赖etcd和supervisor这两个开源项目。关于这两个产品的部署本文档只描述和我们的使用场景有关的部分，关于它们的更详细的说明请查阅项目官网的文档。Iceberg框架应尽可能简单，好用，性能高。单独定制。

## 缩略语及名词解释
- Iceberg

参见《Iceberg服务发现系统说明.md》

- etcd

一个开源的基于Raft算法的分布式一致性解决方案。它受zookeeper的理念影响，提供和zookeeper类型的功能和数据模型。用go语言实现。

- supervisor

一个进程监控服务，可用于linux进程的监控，自动重启


# 部署

Iceberg程序的部署路径是：/usr/iceberg

如无特别说明，Iceberg中的服务都是支持多实例的，可以在根据需要将实例部署在一台或者多台服务器上。如果要将多种服务部署在同一台服务器上，应该考虑各个服务对资源消耗的特别。合理搭配，比如IO密集型的服务搭配内存消耗型的服务或者CPU消耗型的服务。同种消耗特性的服务最好不要部署在一起。

## 实例数量，主备关系
大部分是无状态且等价的。考虑到可用性，最少应该部署两个实例。GateWay会按照一致性哈希在各实例之间做负载均衡。

# 操作手册
## 配置文件
配置文件采用json格式，暂时不支持注释。

```
{
    "IP":"",
    "Port":"3201",
    "EtcdCfg":{
        "EndPoints":["http://localhost:2379"],
        "User":"",
        "Psw":"",
        "Timeout":3
    }
}
```

以上示例是iceberg体系中的服务的最小配置项。可以看出GateWay对etcd的依赖。关于etcd，我们要在配置文件中指定etcd的节点列表。实践中，这里并不要求把所有的etcd结点都写上，因为只有和一个etcd节点连接后，程序会得到etcd所有可用的节点。不过最好还是多写几个，防止万一有节点出现临时性故障。

## 启动
通过Supervisor来管理进程。supervisor会在服务器开机时自动启动我们注册的服务，并且会检测进程的运行状态，一旦发生异常退出的情况它会自动重启我们的进程。

supervisor提供了Web管理界面让我们方便的远程管理进程的启动和关闭。关于supervisor的使用，请进一步参阅supervisor的文档

iceberg体系中的服务依赖两个配置文件，分别是上文提到的conf.json和seelog配置文件。两个配置的默认名称分别是conf.json和seelog.xml如果默认文件和程序在同一个目录下，可以在程序目录中不指定任何参数启动程序：

也可以指定不同的路径下的配置文件
```
--cfg $YOURPATH/$YOUCFG.json --logcfg YOURPATH/$YOUCFG.xml
```
## 停止
向程序发送SIGTERM信号可以正常的结束程序。这种方式不会造成数据丢失。

```
pidof GateWay|xargs kill 
```

## 重启
目前GateWay没有实现专门的graceful reboot功能。所以：

重启=停止+启动

## 扩容
修改好配置文件，指定和其他实例不一样的监听地址和端口后，直接启动服务就好。新实例会通过服务发现机制被iceberg体系中的所有的服务感知。

## 收缩
关闭实例即可。

## 服务降级
暂无

# 搭建Iceberg环境
## etcd
目前我们是以单点的方式使用etcd。所以只要在一台机器上安装和配置etcd即可。如果切换到集群方式，那么就要在多台机器上安装并配置etcd

### 安装etcd

```
cd /usr/local/iceberg
curl -L  https://github.com/coreos/etcd/releases/download/v2.2.5/etcd-v3.2.5-linux-amd64.tar.gz -o etcd-v3.2.5-linux-amd64.tar.gz
tar xzvf etcd-v2.2.5-linux-amd64.tar.gz
cd etcd-v3.2.5-linux-amd64
```

## supervisor
supervisor要在每一个虚拟机上配置。根据iceberg服务的拓扑结构，不同的机器上会将不同的程序添加到supervisor的监控当中。

### 安装supervisor
- 通过easy_install安装

```
easy_install supervisor
```

- 通过pip安装

```
pip install supervisor
```

### 配置supervisor
- 生成配置文件

```
echo_supervisord_conf > /etc/supervisord.conf
```

- 更改基本配置

在配置中找到以下内容。去掉行首的分号（;），127.0.0.1:9001替换成运行本机的IP地址。端口号如没有冲突可保留默认的。username和password应该要重新设定。

```
;[inet_http_server]         ; inet (TCP) server disabled by default
;port=127.0.0.1:9001        ; (ip_address:port specifier, *:port for all iface)
;username=user              ; (default is no username (open server))
;password=123               ; (default is no password (open server))
```

例如：

```
[inet_http_server]         ; inet (TCP) server disabled by default
port=222.73.69.121:9001    ; (ip_address:port specifier, *:port for all iface)
username=admin             ; (default is no username (open server))
password=a1d2m3i4n5        ; (default is no password (open server))
```

- 配置对etcd的监控，在配置中添加如下内容：

```
[program:etcd]
command = /usr/local/iceberg/etcd-v2.2.5-linux-amd64/%(program_name)s --listen-client-urls 'http://:2379,http://localhost:4001' --advertise-client-urls 'http://:2379,http://localhost:4001'
autorestart = true
```

*以上配置是etcd的单点模式的配置，如etcd要以集群方式运行还要指定其他的peer, 详细的配置参见etcd的官网文档*

- 将Iceberg的程序增加到supervisor的监控当中，在配置中添加如下内容：

```
[program:gateWay]
command = /usr/local/iceberg/%(program_name)s/%(program_name)s
autorestart = true
```

其中program:gateWay中的gateWay表明这段配置是把gateWay添加到supervisor的监控中，其他的程序依此类推。

## 初始化服务体系树
- 使用上一步骤中配置的etcd的启动命令手动开启etcd
- 初始化服务体系树型结构

```
cd /usr/local/iceberg/etcd-v3.2.5-linux-amd64
./etcdctl mkdir /services/
```

- 关闭第一步开启的etcd

## 启动服务

```
supervisord -c /etc/supervisord.conf
```

# 部署服务checklist
- <input type="checkbox"/>前一个版本的程序和配置是否已经备份？
- <input type="checkbox"/>各程序的配置文件是否更新，特别是配置中的IP，端口，路径等内容
- <input type="checkbox"/>能否建立数据库连接？相关的表是否都已经赋予了合适的权限？
- <input type="checkbox"/>是否所有的配置中指定的端口都已经在iptables中打开了？


# 后端服务响应
建议使用iceberg内置响应格式

```golang
type Message struct{
    Errcode int `json:"errcode"`
    ErrMsg string `json:"errmsg"`
    Data interface{} `json:"data"`
}
```
# 排查故障Guideline
暂无

# 维护日志
## 重要操作日志
暂无

## 故障备案
暂无
