#ifndef EXPRESS_PROTOCOL_H_
#define EXPRESS_PROTOCOL_H_
#pragma once


#if defined(_WIN32)

	#include <Winsock2.h>
#else
	#include <netinet/in.h>
	#include <arpa/inet.h>
#endif

#include <string>
#include <functional>

//消息;
const int MESSAGE_BASE = 1000;

//HARQ消息;
const int HARQ_CONNECT = MESSAGE_BASE + 1;
const int HARQ_DISCONNECT = MESSAGE_BASE + 2;

//业务消息
const int EXPRESS_REQUEST_ONLINE		= MESSAGE_BASE + 100;
const int EXPRESS_RESPONSE_ONLINE		= MESSAGE_BASE + 101;

const int EXPRESS_REQUEST_CONFIG		= MESSAGE_BASE + 110;
const int EXPRESS_RESPONSE_CONFIG		= MESSAGE_BASE + 111;

const int EXPRESS_REQUEST_FILE			= MESSAGE_BASE + 120;
const int EXPRESS_RESPONSE_PROGRESS		= MESSAGE_BASE + 121;
const int EXPRESS_RESPONSE_FINISHED		= MESSAGE_BASE + 122;
const int EXPRESS_REQUEST_BUFFER			= MESSAGE_BASE + 123;

//本地消息;
const int LOCAL_CONNECT_SERVER = MESSAGE_BASE + 102;
const int LOCAL_LOGIN = MESSAGE_BASE + 101;
const int LOCAL_SEND_FILE = MESSAGE_BASE + 100;

#if defined(_WIN32)
#else
	const int ERROR_HANDLE = -1;
#endif

const int ERROR_PORT	= -1;

const int SUCCESS = 0;
const int MAGPIE_SERVER_BUSYING					= SUCCESS + 1;	//服务端忙；
const int MAGPIE_CLIENT_LIMIT					= SUCCESS + 2;	//客户端被限制；
const int MAGPIE_CLIENT_NOTFOUND				= SUCCESS + 2;	//客户端未找到；
const int MAGPIE_SERVER_NOTFOUND				= SUCCESS + 3;	//服务端未找到；
const int MAGPIE_FOUND							= SUCCESS + 4;	//已经存在；
const int MAGPIE_SERVER_SESSION_EXIST			= SUCCESS + 5;	//服务端的Session已经存在；
const int MAGPIE_SERVER_SESSION_NOTFOUND		= SUCCESS + 6;	//服务端的Session不存在；
const int MAGPIE_JSON_ERROR						= SUCCESS + 7;	//解析JSON格式出错；

const int SINGLE_BLOCK = 4 * 32 * 1024;

#pragma pack(push, 1)

struct header
{
	int protocol_id_;
};

const int SESSION_LEN = 100;
struct request_login
{
	header header_;
	char session_[SESSION_LEN];
};
struct reponse_login
{
    header header_;
    int result_;
};

const int PATH_LEN = 4 * 1024;
struct request_config
{
	header header_;
	int64_t file_size_;
	char local_path_[PATH_LEN];		//发送端的文件绝对路径
	char remote_path_[PATH_LEN];		//接收端的相对路径
	char remote_name_[PATH_LEN];		//接收端的文件名称

};

struct reponse_config
{
	header header_;
	int64_t cur_;
	char absolute_path_[PATH_LEN];
	int result_;
};

struct request_file
{
	header header_;
	int64_t max_;
	int64_t cur_;
	int size_;
	char data_[SINGLE_BLOCK];
};

struct reponse_file
{
	header header_;
	int result_;
};

struct reponse_file_progress
{
	header header_;
	int max_id_;
	int cur_id_;
	char absolute_path_[PATH_LEN];
};

struct reponse_file_finished
{
	header header_;
	char absolute_path_[PATH_LEN];
	long long file_size_;
};

struct request_buffer
{
	header header_;
	int size_;
};

struct request_logout
{
	header header_;
};

#pragma pack(pop)

#endif	//EXPRESS_PROTOCOL_H_
