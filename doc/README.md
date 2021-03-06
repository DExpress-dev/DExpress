# DExpress(Data Express)

------

DExpress是一款低延时安全数据传输产品。这款产品采用HARQ（混合自动重传请求）技术，实现了底层数据的可靠传输，FEC冗余包控制在10%之内。同时可选择动态加密方式进行传输数据，为每次数据传输均采用了动态加密方式，保证了数据在传输过程中的安全、私密性。拥塞控制采用的拥塞算法使用了线性斜率算法和丢包算法同时进行控制，保证了在不同网络环境下发送数据占用带宽能够非常平滑的进行扩充或者收敛。


## 目录说明：

### file_express
	file_express：保存使用DExpress进行编写的文件传输的Demo与模块。

```python

	c++：
		windows：Windows下编写的Demo和传输模块。
		linux：Linux下编写的Demo和传输模块。

```



### media_express

## Demo实测截图

###	说明：	

- 以下截图是我编写的一个直播Demo的截图，一个用户手机A采集画面并将采集到的画面发送给中转服务器。中转服务器接收到数据后，将画面数据转发给另外的一个用户手机B。用户手机B将接收到的数据绘制到手机中。

- 为了保证数据的连贯性，在B手机端人为设置了50毫秒的延迟。
	
- 因此如果只是做云游戏从服务端将画面传到客户端进行展现，实际的时间应该为：云游戏延迟时间 = （测量时间 - 50）/ 2;
	
第一张截图：

![image](https://github.com/DExpress-dev/DExpress/blob/main/doc/live0.jpg)

第二张截图：

![image](https://github.com/DExpress-dev/DExpress/blob/main/doc/live3.jpg)

第三张截图：

![image](https://github.com/DExpress-dev/DExpress/blob/main/doc/live8.jpg)

## HLS低延时安全传输架构图
![image](E:/Github/DExpress/image/framework_hls.jpg)

## UDP组播公网传输架构图
![image](https://github.com/Tinachain/DExpress/blob/master/image/framework_udp.jpg)

## 性能导图
![image](https://github.com/Tinachain/DExpress/blob/master/image/performance.jpg)


