# DExpress(Data Express)

------

DExpress是一款低延时安全数据传输产品。这款产品采用HARQ（混合自动重传请求）技术，实现了底层数据的可靠传输，FEC冗余包控制在10%之内。同时可选择动态加密方式进行传输数据，为每次数据传输均采用了动态加密方式，保证了数据在传输过程中的安全、私密性。拥塞控制采用的拥塞算法使用了线性斜率算法和丢包算法同时进行控制，保证了在不同网络环境下发送数据占用带宽能够非常平滑的进行扩充或者收敛。


## 目录说明：

### src
	存放接口库的源码。

### windows
	存放着接口库的Windows版本代码，其中透出的接口包含有客户端和服务端。
	
### linux
	存放着接口库的linux版本代码，其中透出的接口包含有客户端和服务端。

```python

	c++：
		windows：Windows下编写的Demo和传输模块。
		linux：Linux下编写的Demo和传输模块。

```


## HLS低延时安全传输架构图
![image](E:/Github/DExpress/image/framework_hls.jpg)

## UDP组播公网传输架构图
![image](https://github.com/Tinachain/DExpress/blob/master/image/framework_udp.jpg)

## 性能导图
![image](https://github.com/Tinachain/DExpress/blob/master/image/performance.jpg)


