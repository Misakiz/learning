### operator-learn

学习operator的资料
b站视频         https://www.bilibili.com/video/BV1uS4y1V7iZ/?spm_id_from=333.788&vd_source=d5ee327d9a6b29e6c35103c4f4dec7d5

博主分析        https://www.cnblogs.com/luozhiyun/tag/%E6%B7%B1%E5%85%A5k8s/

K8s官方文档     https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23

(本地存储，减少与apiServer访问次数)
Indexer        https://blog.csdn.net/li_101357/article/details/89743569

(一个队列，用户存储资源对象的变化，类生产者(reflector)消费者()模型)
deltaFIFO      https://cloud.tencent.com/developer/article/1692474

(生产者消费者模型，权衡worker处理慢和生产者生产数据过快的速度)
workQueue      https://jishuin.proginn.com/p/763bfbd2d0c7
               https://blog.pangsq.cn/2020/05/05/workQueue/#Interface

1.安装minikube模拟
https://www.voidking.com/dev-macos-minikube/

minikube start --image-mirror-country='cn'
--kubernetes-version=v1.23.3
 
