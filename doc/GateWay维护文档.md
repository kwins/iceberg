# 前言
本文档主要描述GateWay的部署和维护事项。关于GateWay的更详细的说明请参考《GateWay设计文档.md》

## 缩略语及名词解释
无

# 部署
GateWay是的唯一职责就是中转请求和响应，它的资源消耗应该非常小。如果仅作为后端服务的入口，它是可以和其他服务部署在同一台机器上的。

## 实例数量，主备关系
GateWay所有实例都是无状态且等价的。考虑到可用性，最少应该部署两个实例。GateWay的负载均衡可以依赖dns解析和nginx反向代理或者LVS等手段。这些内容不在本文的讨论范围内。

# 操作手册
## 配置文件
配置文件采用json格式，暂时不支持注释。NotifySvr

```json
{
    "IP": "",
    "Port": "3201",
    "baseCfg": {
        "etcdCfg": {
            "EndPoints": ["http://127.0.0.1:2379"],
            "User": "iceberg",
            "Psw": "123456",
            "Timeout": 3
        }
    }
}
```

从以上示例可以看出GateWay对etcd的依赖。关于etcd，我们要在配置文件中指定etcd的节点列表。实践中，这里并不要求把所有的etcd结点都写上，因为只有和一个etcd节点连接后，程序会得到etcd所有可用的节点。不过最好还是多写几个，防止万一有节点出现临时性故障。

## 启动
Go程序目前不能完美支持用fork的方式以daemon的方式运行。可以通过Supervisor来管理进程。由于我们还没有使用supervisor, 所以我们可以用nohup的方式来让程序以daemon的形式运行

也可以指定不同的路径下的配置文件

```cmd
nohup ./GateWay --config-path $YOURPATH/$YOUCFG.json
```

## 停止
向程序发送SIGTERM信号可以正常的结束程序。这种方式不会造成数据丢失。

```cmd
pidof GateWay|xargs kill 
```

## 重启
目前GateWay没有实现专门的graceful reboot功能。所以：

重启=停止+启动

## 扩容
部署新的实例，修改DNS或者nginx，此处不展开。

## 收缩
修改DNS或者nginx，关闭实例，此处不展开。

## 服务降级
暂无

# 排查故障Guideline
暂无

# 维护日志
## 重要操作日志
暂无

## 故障备案
暂无
