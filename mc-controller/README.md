# mc-controller 多云Controller

## 1. 简介

`mc-controller` 用于多云环境的使用。因srm架构的服务发现是基于K8s的svc。该项目方便创建deploy的同时，自动创建对应svc已经绑定到pod对应ep
但因可以直接使用chart达到效果，所以该项目胎死腹中

## 2. 项目结构

```shell
入口函数  
pkg 
-- controller
main.go
```

## 2. controller

实现对deploy资源更新和新增事件的监听，
> informer->eventHander->workQueue（限速worker消费速度比较慢）->worker
> 