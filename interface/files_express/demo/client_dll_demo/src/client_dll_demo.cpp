// server_windows_demo.cpp : 定义控制台应用程序的入口点。
//

#include "stdafx.h"
#include "interface.h"

#if defined(_WIN32)

	#include <windows.h>
	#include <Mmsystem.h>

	#define lib_load(a) LoadLibrary(a)
	#define lib_handle HINSTANCE
	#define lib_error() GetLastError()
	#define lib_function(a, b) GetProcAddress(a, b)
	#define lib_close(a) FreeLibrary(a)
#else

	#include <dlfcn.h>
	#define lib_load(a) dlopen(a, RTLD_LAZY)
	#define lib_handle void*
	#define lib_error() dlerror()
	#define lib_function(a, b) dlsym(a, b)
	#define lib_close(a) dlclose(a)
#endif

#include <string>
#include <stdio.h>
#include <stdarg.h>
#include <memory>
#include <vector>
#include <time.h>
#include "path.h"

/*接口函数定义*/
typedef int(*OPEN_CLIENT)(char* bind_ip, char* remote_ip, int remote_port, char* log, char *harq_so_path, char* session, bool encrypted,
	ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finish, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error);

typedef bool(*SEND_FILE)(int express_handle, char* local_file_path, char* remote_relative_path, char* file_name);
typedef bool(*SEND_DIR)(int express_handle, char* dir_path, char* save_relative_path);
typedef void(*STOP_SEND)(int express_handle, char* file_path);
typedef void(*CLOSE_CLIENT)(int express_handle);
typedef char* (*VERSION)(void);

/*回调函数实现*/
void on_login(int express_handle, char* remote_ip, int remote_port, char* session)
{
	return;
}

void on_finish(int express_handle, char* file_path, long long size)
{
	printf("on_finish file_path=%s size=%lld\n", file_path, size);
}

void on_progress(int express_handle, char* file_path, int max, int cur)
{
}

void on_buffer(int express_handle, char* data, int size)
{

}

void on_disconnect(int express_handle, char* remote_ip, int remote_port)
{
	return;
}

void on_error(int express_handle, int errorid, char* remote_ip, int remote_port)
{
	return;
}

int main()
{
	//接口定义;
	lib_handle lib_handle_ = nullptr;
	OPEN_CLIENT open_client_ptr = nullptr;
	SEND_FILE send_file_ptr = nullptr;
	SEND_DIR send_dir_ptr = nullptr;
	CLOSE_CLIENT close_client_ptr = nullptr;
	VERSION version_ptr = nullptr;

	//加载dll
	std::string current_path = "D:\\linux\\project\\CPlus\\public\\files_express\\demo\\client_dll_demo\\Debug/express_client.dll";
	lib_handle_ = lib_load(current_path.c_str());
	if (nullptr == lib_handle_)
	{
		printf("lib_load error\n");
		return -1;
	}

	//加载so透出的函数;
	open_client_ptr = (OPEN_CLIENT)lib_function(lib_handle_, "open_client");
	if (nullptr == open_client_ptr)
	{
		printf("lib_function OPEN_CLIENT error\n");
		lib_close(lib_handle_);
		return -1;
	}

	send_file_ptr = (SEND_FILE)lib_function(lib_handle_, "send_file");
	if (nullptr == send_file_ptr)
	{
		printf("lib_function SEND_FILE error\n");
		lib_close(lib_handle_);
		return -1;
	}

	send_dir_ptr = (SEND_DIR)lib_function(lib_handle_, "send_dir");
	if (nullptr == send_dir_ptr)
	{
		printf("lib_function SEND_DIR error\n");
		lib_close(lib_handle_);
		return -1;
	}

	close_client_ptr = (CLOSE_CLIENT)lib_function(lib_handle_, "close_client");
	if (nullptr == close_client_ptr)
	{
		printf("lib_function CLOSE_CLIENT error\n");
		lib_close(lib_handle_);
		return -1;
	}

	version_ptr = (VERSION)lib_function(lib_handle_, "version");
	if (nullptr == version_ptr)
	{
		printf("lib_function VERSION error\n");
		lib_close(lib_handle_);
		return -1;
	}

	std::vector<std::string> remote_array;
	remote_array.push_back("10.10.50.136");
	std::string log = "log";
	std::string session = "123456";

	std::vector<std::string> send_file_array;
	send_file_array.push_back("E:/tools/radstudio10_1_upd2_esd.iso");

	std::vector<std::string> send_file_array2;
	send_file_array2.push_back("E:/tools/go1.14.4.linux-amd64.tar.gz");
	send_file_array2.push_back("E:/tools/qt-opensource-linux-x64-5.14.0.run");
	send_file_array2.push_back("E:/tools/livego.tar.gz");
	send_file_array2.push_back("E:/tools/qtxmlpatterns-everywhere-src-5.15.0.zip");
	send_file_array2.push_back("E:/tools/vs_Community_2017.exe");

	std::vector<int> remote_handle_array;
	for (auto iter = remote_array.begin(); iter != remote_array.end(); iter++)
	{
		std::string remote_ip_string = *iter;
		std::string libharqpath = "D:/projects/Chainware/最终产品/windows/lib/harq/harq_32.dll";
		int express_handle = open_client_ptr("0.0.0.0", (char*)remote_ip_string.c_str(), 41002, (char*)log.c_str(), (char*)libharqpath.c_str(), (char*)session.c_str(), false, &on_login, &on_progress, &on_finish, &on_buffer, &on_disconnect, &on_error);
		if (express_handle <= 0)
		{
			printf("open_client_ptr error remote_ip=%s\n", remote_ip_string.c_str());
			lib_close(lib_handle_);
			return -1;
		}
		remote_handle_array.push_back(express_handle);
	}

	::Sleep(1000);

	time_t last_send_time = time(nullptr);
	int postion = 0;
	while (1)
	{
		time_t current_timer = time(nullptr);
		int second = static_cast<int>(difftime(current_timer, last_send_time));
		if (second >= 1)
		{
			//循环发送文件给对应的服务器
			int handle_postion = 0;
			for (auto iter = remote_handle_array.begin(); iter != remote_handle_array.end(); iter++)
			{
				handle_postion++;
				int handle = *iter;

				//对端目的地;
				std::string save_relative_path = "20201219/debug";
				std::string file_path = "";
				if (handle_postion == 1)
				{
					file_path = send_file_array[postion];
				}
				else if (handle_postion == 2)
				{
					file_path = send_file_array2[postion];
				}

				//本地文件;
				std::string file_name = ustd::path::get_filename(file_path);
				bool sended = send_file_ptr(handle, (char*)file_path.c_str(), (char*)save_relative_path.c_str(), (char*)file_name.c_str());

				//发送日志
				printf("send_file_ptr handle=%d file=%s\n", handle, file_path.c_str());			
			}
			postion++;
			if (postion >= send_file_array.size())
				break;
			else
				last_send_time = time(nullptr);
		}
	}

	int checkPostion = 0;
	while (1)
	{
		checkPostion++;
		::Sleep(1000);
	}
	return 0;
}

