#include "express_c_interface.h"
#include <dlfcn.h>
#include <stddef.h>
#include <stdbool.h>

//回调函数定义
void OnLogin(int express_handle, char* remote_ip, int remote_port, char* session);
bool OnProgress(int express_handle, char* file_path, int max, int cur);
void OnFinish(int express_handle, char* file_path, long long size);
void OnDisconnect(int express_handle, char* remote_ip, int remote_port);
void OnClientError(int express_handle, int error, char* remote_ip, int remote_port);

OPEN_CLIENT open_client_ptr = NULL;
SEND_FILE send_file_ptr = NULL;
SEND_DIR send_dir_ptr = NULL;
STOP_SEND stop_send_file_ptr = NULL;
CLOSE_CLIENT close_client_ptr = NULL;
VERSION_CLIENT version_client_ptr = NULL;
void* lib_handle_;

bool init_client(char* library_path)
{
	lib_handle_ = dlopen(library_path, RTLD_LAZY);
	if(!lib_handle_) 
	{
		return false;
	}
	open_client_ptr = (OPEN_CLIENT)dlsym(lib_handle_, "open_client");
	send_file_ptr = (SEND_FILE)dlsym(lib_handle_, "send_file");
	send_dir_ptr = (SEND_DIR)dlsym(lib_handle_, "send_dir");
	stop_send_file_ptr = (STOP_SEND)dlsym(lib_handle_, "stop_send");
	close_client_ptr = (CLOSE_CLIENT)dlsym(lib_handle_, "close_client");
	version_client_ptr = (VERSION_CLIENT)dlsym(lib_handle_, "version");

	if (NULL == open_client_ptr)
	{
		dlclose(lib_handle_);
		return false;
	}
	if (NULL == send_file_ptr)
	{
		dlclose(lib_handle_);
		return false;
	}
	if (NULL == send_dir_ptr)
	{
		dlclose(lib_handle_);
		return false;
	}
	if(NULL == stop_send_file_ptr)
	{
		dlclose(lib_handle_);
		return false;
	}
	if (NULL == close_client_ptr)
	{
		dlclose(lib_handle_);
		return false;
	}
	if (NULL == version_client_ptr)
	{
		dlclose(lib_handle_);
		return false;
	}
	return true;
}

int start_client(char* bind_ip, char* remote_ip, int remote_port, char* log, char *harq_so_path, char* session, bool encrypted)
{
	if(NULL != open_client_ptr)
	{
		int express_handle = open_client_ptr(bind_ip, remote_ip, remote_port, log, harq_so_path, session, encrypted, OnLogin, OnProgress, OnFinish, OnDisconnect, OnClientError);
		if (express_handle <= 0)
			return -1;
		else
			return express_handle;
	}
    return -1;
}

bool send_file(int express_handle, char* file_path, char* save_relative_path)
{
    if(NULL != send_file_ptr)
	{
		return send_file_ptr(express_handle, file_path, save_relative_path);
	}
	return false;
}

bool send_dir(int express_handle, char* dir_path, char* save_relative_path)
{
    if(NULL != send_dir_ptr)
	{
		return send_dir_ptr(express_handle, dir_path, save_relative_path);
	}
	return false;
}

void stop_send_file(int express_handle, char* file_path)
{
    if(NULL != stop_send_file_ptr)
	{
		stop_send_file_ptr(express_handle, file_path);
	}
}

void close_client(int express_handle)
{
    if(NULL != close_client_ptr)
	{
		close_client_ptr(express_handle);
	}
}

char* version()
{
    if(NULL != version_client_ptr)
        return version_client_ptr();    
    else
        return "0.0.0";
}


