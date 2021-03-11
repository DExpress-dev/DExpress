#include "harq_c_interface.h"
#include <dlfcn.h>
#include <stddef.h>
#include <stdbool.h>

//回调函数定义
void OnConnect(char* remote_ip, int remote_port, int linker_handle, long long time_stamp);
bool OnReceive(char* data, int size, int linker_handle, char* remote_ip, int remote_port, int consume_timer);
void OnDisconnect(int linker_handle, char* remote_ip, int remote_port);
void OnError(int error, int linker_handle, char* remote_ip, int remote_port);
void OnRto(char* remote_ip, int remote_port, int local_rto, int remote_rto);
void OnRate(char* remote_ip, int remote_port, unsigned int send_rate, unsigned int recv_rate);

HARQ_START_CLIENT harq_start_client_ptr_ = NULL;
HARQ_SEND_BUFFER_HANDLE harq_send_buffer_handle_ptr_ = NULL;
HARQ_CLOSE_HANDLE harq_close_handle_ptr_ = NULL;
HARQ_VERSION harq_version_ptr_ = NULL;
HARQ_END_SERVER harq_end_server_ptr_ = NULL;

bool init_client(char* library_path)
{
	void* lib_handle_;
	lib_handle_ = dlopen(library_path, RTLD_LAZY);
	if(!lib_handle_) 
	{
		return false;
	}
	harq_start_client_ptr_ = (HARQ_START_CLIENT)dlsym(lib_handle_, "harq_start_client");
	harq_send_buffer_handle_ptr_ = (HARQ_SEND_BUFFER_HANDLE)dlsym(lib_handle_, "harq_send_buffer_handle");
	harq_close_handle_ptr_ = (HARQ_CLOSE_HANDLE)dlsym(lib_handle_, "harq_close_handle");
	harq_version_ptr_ = (HARQ_VERSION)dlsym(lib_handle_, "harq_version");
	harq_end_server_ptr_ = (HARQ_END_SERVER)dlsym(lib_handle_, "harq_end_server");

	if (NULL == harq_start_client_ptr_)
	{
		dlclose(lib_handle_);
		return false;
	}
	if (NULL == harq_send_buffer_handle_ptr_)
	{
		dlclose(lib_handle_);
		return false;
	}
	if (NULL == harq_close_handle_ptr_)
	{
		dlclose(lib_handle_);
		return false;
	}
	if (NULL == harq_version_ptr_)
	{
		dlclose(lib_handle_);
		return false;
	}
	if (NULL == harq_end_server_ptr_)
	{
		dlclose(lib_handle_);
		return false;
	}
	return true;
}

int start_client(char* log, char* remote_ip, int remote_port, bool encrypted)
{
   if(NULL != harq_start_client_ptr_)
       return harq_start_client_ptr_(log, "0.0.0.0", remote_ip, remote_port, 5, false, 2000, encrypted, OnConnect, OnReceive, OnDisconnect, OnError, OnRto, OnRate); 
   else
       return -1;
}

int send_buffer(char* data, int size, int handle)
{
    if(NULL != harq_send_buffer_handle_ptr_)
        return harq_send_buffer_handle_ptr_(data, size, handle);    
    else
        return -1;
}

void close_client(int handle)
{
    if(NULL != harq_close_handle_ptr_)
        return harq_close_handle_ptr_(handle);    
}

char* version()
{
    if(NULL != harq_version_ptr_)
        return harq_version_ptr_();    
    else
        return "0.0.0";
}


