# ats_check

本工具主要用作ats前置节点(非parent节点)的监控检测.

由于ats节点使用URL一致性哈希把请求打到上层节点的时候，如果上层节点挂掉，ats节点本身会做rehash,但rehash的次数是有限制的,极端情况下,当所有的上层节点不可用时，我们希望ats节点能直接回源操作，保障可用性.

本工具除了能监控测试外，还增加了容错和故障恢复等功能


### 服务场景

一、边缘节点依赖多个上层节点,当出现某上层节点服务不可用时,ats_check会自动发现并迅速调整 ATS 的parent.config的配置

二、当所有上层节点都不可用时,ats_check 会进一步调整 records.config 关闭parent proxy功能,并自动调整 remap,让请求能直接通过balance，回到源站


### 使用方式：

主要用于边缘节点

```
sh atscheck.sh start
```

运行日志：

/apps/logs/ats_check/ats_check.log