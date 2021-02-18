# DExpress(Data Express)

------

DExpress是一款低延时安全数据传输产品。这款产品采用HARQ（混合自动重传请求）技术，实现了底层数据的可靠传输，FEC冗余包控制在10%之内。同时可选择动态加密方式进行传输数据，为每次数据传输均采用了动态加密方式，保证了数据在传输过程中的安全、私密性。拥塞控制采用的拥塞算法使用了线性斜率算法和丢包算法同时进行控制，保证了在不同网络环境下发送数据占用带宽能够非常平滑的进行扩充或者收敛。


## 目录说明：


### server
	存放着接口库的Windows版本服务端代码，其中透出的接口包含有：
	
	
	接口：
		open_server：	打开服务端
	参数：
		char *bind_ip：				绑定的IP地址（常用0.0.0.0）
		int listen_port, 			监听端口
		char *log_path, 			日志保存路径
		char *harq_path, 			内核库（libharq.so、harq.dll）存放的绝对路径
		char *base_path, 			文件保存的基础目录
		bool same_name_deleted,		目录下的同名文件是否删除
		ON_EXPRESS_LOGIN on_login, 				客户端登录的回调
		ON_EXPRESS_PROGRESS on_progress,		接收文件进度的回调 
		ON_EXPRESS_FINISH on_finished, 			接收完文件的回调
		ON_EXPRESS_BUFFER on_buffer, 			接收到数据的回调
		ON_EXPRESS_DISCONNECT on_disconnect,	连接断开的回调 
		ON_EXPRESS_ERROR on_error				产生错误的回调
		
	close_server：
		关闭服务端
	version：
		获取当前版本

### client
	存放着接口库的客户端代码，其中透出的接口包含有：

	open_client：
		打开客户端
	参数：
		char* bind_ip：				绑定的IP地址（常用0.0.0.0）
		char* remote_ip：			服务端IP地址		
		int remote_port：			服务端监听的端口
		char* log：					日志保存路径
		char *harq_so_path：			内核库（libharq.so、harq.dll）存放的绝对路径	
		char* session：				登录服务端所用的session
		bool encrypted：				数据是否需要加密		
		ON_EXPRESS_LOGIN on_login：				登录完成回调
		ON_EXPRESS_PROGRESS on_progress：		接收文件进度的回调
		ON_EXPRESS_FINISH on_finish：			接收完文件的回调
		ON_EXPRESS_BUFFER on_buffer：			接收到数据的回调
		ON_EXPRESS_DISCONNECT on_disconnect：	连接断开的回调
		ON_EXPRESS_ERROR on_error：				产生错误的回调
	返回：
		int：返回客户端传输使用的句柄
	

	send_file：
		发送指定文件
	参数：
		int express_handle：			客户端传输使用的句柄			 
		char* local_file_path：		传输文件的本地路径
		char* remote_relative_path：传输文件的远端相对路径（相对于服务端设置中的Base参数）
		char* file_name：			传输文件的文件名
	返回：
		bool：是否可以发送指定文件

	send_dir：
		发送指定目录
	参数：
		int express_handle：			客户端传输使用的句柄 
		char* dir_path：				需要传输的目录
		char* remote_relative_path：	传输目录的远端相对路径（相对于服务端设置中的Base参数）
	返回：
		bool：是否可以发送目录

	send_buffer：
		发送指定数据
	参数：
		int express_handle：			客户端传输使用的句柄
		char* data：					需要传输的数据指针
		int size：					需要传输的数据大小
	返回：
		bool：是否可以传输数据

	cur_waiting_size：
		当前等待发送的文件数量
	参数：
		int express_handle：			客户端传输使用的句柄
	返回：
		int：等待发送的文件数量
			
	stop_send：
		停止发送文件
	参数：
		int express_handle：			客户端传输使用的句柄 
		char* local_file_path：		需要停止的文件本地路径

	close_client：
		关闭客户端
	参数：
		int express_handle

	version：
		获取当前版本
	返回：
		char*：当前使用的客户端版本

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


