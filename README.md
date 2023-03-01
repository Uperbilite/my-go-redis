# godis
- 项目是用golang写的一个简略版本的redis-server，参照redis1.0、1.2、2.6
- 没有使用net库、goroutine、channel等golang特色工具。使用unix包的系统调用实现ae事件库，目的是为了复刻redis的设计
- ae事件库仅实现了epoll版本，所以只能在linux系统中运行
- dict实现参照redis2.6，完整实现了渐进式rehash
- 实现了RESP流式协议下命令的读取与返回结果
- 项目编写了完善的单元测试

# 项目文档
- [redis源码解读](https://www.yuque.com/uperbilite/sihot5/fm866mf0nbhypbq6?singleDoc# 《redis源码解读》) 
- [godis项目设计](https://www.yuque.com/uperbilite/sihot5/ifbzzgzs8ssmokwm?singleDoc# 《Godis》) 

# 项目启动
```
go run .
```
