#ifndef EXPRESS_INTERFACE_H_
#define EXPRESS_INTERFACE_H_
#pragma once


#if defined(_WIN32)
	#include <Winsock2.h>
#else
	#include <netinet/in.h>
	#include <arpa/inet.h>
#endif

#include <string>
#include <functional>

//接口回调函数
typedef void(*ON_EXPRESS_LOGIN)(int express_handle, char* remote_ip, int remote_port, char* session);
typedef void(*ON_EXPRESS_PROGRESS)(int express_handle, char* file_path, int max, int cur);
typedef void(*ON_EXPRESS_FINISH)(int express_handle, char* file_path, long long size);
typedef void(*ON_EXPRESS_BUFFER)(int express_handle, char* data, int size);
typedef void(*ON_EXPRESS_DISCONNECT)(int express_handle, char* remote_ip, int remote_port);
typedef void(*ON_EXPRESS_ERROR)(int express_handle, int errorid, char* remote_ip, int remote_port);

//服务端接口定义
typedef int(*EXPRESS_OPEN_SERVER)(char *bind_ip, int listen_port, char *log_path, char *harq_path, char *base_path, bool same_name_deleted, ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finished, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error);
typedef void(*EXPRESS_CLOSE_SERVER)();
typedef char*(*EXPRESS_VERSION_SERVER)();

//客户端接口定义
typedef int(*EXPRESS_OPEN_CLIENT)(char* bind_ip, char* remote_ip, int remote_port, char* log, char *harq_so_path, char* session, bool encrypted, ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finish, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error);
typedef bool(*EXPRESS_SEND_FILE)(int express_handle, char* local_file_path, char* remote_relative_path, char* file_name);
typedef bool(*EXPRESS_SEND_DIR)(int express_handle, char* dir_path, char* save_relative_path);
typedef int(*EXPRESS_WAIT_SIZE)(int express_handle);
typedef void(*EXPRESS_STOP_SEND)(int express_handle, char* file_path);
typedef void(*EXPRESS_CLOSE_CLIENT)(int express_handle);
typedef char*(*EXPRESS_VERSION_CLIENT)();

#endif	//EXPRESS_INTERFACE_H_
