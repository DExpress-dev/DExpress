# DExpress(Data Express)

------

DExpress是一款低延时安全数据传输产品。这款产品采用HARQ（混合自动重传请求）技术，实现了底层数据的可靠传输，FEC冗余包控制在10%之内。同时可选择动态加密方式进行传输数据，为每次数据传输均采用了动态加密方式，保证了数据在传输过程中的安全、私密性。拥塞控制采用的拥塞算法使用了线性斜率算法和丢包算法同时进行控制，保证了在不同网络环境下发送数据占用带宽能够非常平滑的进行扩充或者收敛。


## 目录说明：

### core
	存放着DExpress的内核调用库，其中x86目录下存放的是32位库、x64下存放的是64位库。

### client
	存放着DExpress针对不同引用实现的不同客户端封装库，针对不同的引用场景使用不同的封装接口，其中包含：
	file_express:
		针对文件传输的接口封装。
	media_express:
		针对视频流传输的接口封装。

### server
	存放着DExpress调用的服务端库，和客户端接口封装类似，其中包括file_express的接口封装和media_express的接口封装。

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


