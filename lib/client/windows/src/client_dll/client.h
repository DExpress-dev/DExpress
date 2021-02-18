#ifndef EXPRESS_CLIENT_H_
#define EXPRESS_CLIENT_H_

#include <string>
#include <stdio.h>
#include <stdarg.h>
#include <memory>
#include "interface.h"

#ifdef EXPRESS_CLIENT_LIB
	#define EXPRESS_CLIENT_LIB extern "C" _declspec(dllimport) 
#else
	#define EXPRESS_CLIENT_LIB extern "C" _declspec(dllexport) 
#endif

// 打开客户端;
// 参数：remote_ip：服务端IP地址，remote_port：服务端端口，log：日志保存路径，harq_so_path：harq的dll存放路径 session：session信息（这里可以忽略） encrypted：传输中是否动态加密（默认为不加密）
EXPRESS_CLIENT_LIB int open_client(char* bind_ip, 
	char* remote_ip, 
	int remote_port, 
	char* log, 
	char *harq_so_path, 
	char* session, 
	bool encrypted,
	ON_EXPRESS_LOGIN on_login,
	ON_EXPRESS_PROGRESS on_progress, 
	ON_EXPRESS_FINISH on_finish, 
	ON_EXPRESS_DISCONNECT on_disconnect, 
	ON_EXPRESS_ERROR on_error);

// 发送文件
// 参数：
//express_handle：传输句柄
//file_path：传输的本地文件（绝对路径）
//save_relative_path：对方保存的路径（相对路径）
EXPRESS_CLIENT_LIB bool send_file(int express_handle, char* file_path, char* save_relative_path);

// 发送目录
// 参数：
//express_handle：传输句柄
//dir_path：传输的目录（绝对路径）
//save_relative_path：对方保存的路径（相对路径）
EXPRESS_CLIENT_LIB bool send_dir(int express_handle, char* dir_path, char* save_relative_path);

//停止发送
// 参数：
//express_handle：传输句柄
//file_path：需要删除的文件
EXPRESS_CLIENT_LIB void stop_send(int express_handle, char* file_path);

// 关闭连接;
// 参数：
//express_handle：传输句柄
EXPRESS_CLIENT_LIB void close_client(int express_handle);

// 版本信息
EXPRESS_CLIENT_LIB char* version(void);


#endif	//EXPRESS_CLIENT_H_