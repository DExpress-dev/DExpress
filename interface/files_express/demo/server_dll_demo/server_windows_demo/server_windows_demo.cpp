// server_windows_demo.cpp : 定义控制台应用程序的入口点。
//

#include "stdafx.h"
#include "interface.h"
#include "ini.h"
#include "path.h"

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
#include "write_log.h"

/*接口函数定义*/

// 启动服务端;
typedef bool(*OPEN_SERVER)(
	char* bind_ip,
	int listen_port,
	char *log_path,
	char *harq_path,
	char *base_path,
	ON_EXPRESS_LOGIN on_login,
	ON_EXPRESS_PROGRESS on_progress,
	ON_EXPRESS_FINISH on_finished,
	ON_EXPRESS_DISCONNECT on_disconnect,
	ON_EXPRESS_ERROR on_error);
typedef void(*CLOSE_SERVER)();
typedef char* (*VERSION)();

/*回调函数实现*/
void on_login(int express_handle, char* remote_ip, int remote_port, char* session)
{
	printf("new connect %s %d\n", remote_ip, remote_port);
}

bool on_progress(int express_handle, char* file_path, int max, int cur)
{
	return false;
}

void on_finish(int express_handle, char* file_path, long long size)
{

}

void on_disconnect(int express_handle, char* remote_ip, int remote_port)
{
	return;
}

void on_error(int express_handle, int error, char* remote_ip, int remote_port)
{
	return;
}

void add_log(ustd::log::write_log *write_log_ptr, int log_type, char *log_text_format, ...)
{
	const int array_length = 1024 * 10;
	char log_text[array_length];
	memset(log_text, 0x00, array_length);

	va_list arg_ptr;
	va_start(arg_ptr, log_text_format);
	int result = vsprintf(log_text, log_text_format, arg_ptr);
	va_end(arg_ptr);
	if (result <= 0)
		return;

	if (result > array_length)
		return;

	if (nullptr != write_log_ptr)
		write_log_ptr->write_log3(log_type, log_text);
}

int main()
{
	ustd::log::write_log *write_log_ptr_ = nullptr;
	write_log_ptr_ = new ustd::log::write_log(true);
	write_log_ptr_->init("server_demo", "test", 1);

	//读取配置文件;
	std::string config_string = "config.ini";
	std::string current_path = ustd::path::get_app_path();
	std::string config_path = current_path + "/" + config_string;
	bool exist = ustd::path::is_file_exist(config_path);
	if (!exist)
	{
		add_log(write_log_ptr_, LOG_TYPE_ERROR, "not found config file file_path=%s", config_path.c_str());
		delete write_log_ptr_;
		write_log_ptr_ = nullptr;
		return -1;
	}

	//读取配置文件;
	ini *config_ini = new ini(config_path);
	
	//读取server的dll
	std::string server_path = config_ini->read_string("Server", "Dll", "server.dll");
	add_log(write_log_ptr_, LOG_TYPE_INFO, "server_path=%s", server_path.c_str());
	if (!ustd::path::is_file_exist(server_path))
	{
		add_log(write_log_ptr_, LOG_TYPE_ERROR, "not found server.dll file");
		delete write_log_ptr_;
		write_log_ptr_ = nullptr;
		return -1;
	}

	//读取Harq的dll
	std::string harq_path = config_ini->read_string("Server", "Harq", "harq.dll");
	if (!ustd::path::is_file_exist(harq_path))
	{
		add_log(write_log_ptr_, LOG_TYPE_ERROR, "not found harq.dll file");
		delete write_log_ptr_;
		write_log_ptr_ = nullptr;
		return -1;
	}

	//读取绑定的ip地址
	std::string bind_ip = config_ini->read_string("Server", "BindIp", "0.0.0.0");
	//读取日志保存路径（注意这里是相对路径）
	std::string log_path = config_ini->read_string("Server", "Log", "log");
	//读取监听端口
	int listen_port = config_ini->read_int("Server", "Listen", 41002);
	//读取保存路径
	std::string base_path = config_ini->read_string("Server", "SavePath", "C:/Base");
	if (!ustd::path::is_directory_exist(base_path))
	{
		add_log(write_log_ptr_, LOG_TYPE_ERROR, "not found save path ");
		delete write_log_ptr_;
		write_log_ptr_ = nullptr;
		return -1;
	}
	delete config_ini;
	config_ini = nullptr;

	add_log(write_log_ptr_, LOG_TYPE_INFO, "---->>>>\n server_path=%s harq_path=%s bind_ip=%s log_path=%s listen_port=%d base_path=%s",
		server_path.c_str(), harq_path.c_str(), bind_ip.c_str(), log_path.c_str(), listen_port, base_path.c_str());

	//接口定义;
	lib_handle lib_handle_ = nullptr;
	OPEN_SERVER open_server_ptr_ = nullptr;
	CLOSE_SERVER close_server_ptr_ = nullptr;
	VERSION version_ptr_ = nullptr;

	//加载dll
	lib_handle_ = lib_load(server_path.c_str());
	if (nullptr == lib_handle_)
	{
		DWORD s = NULL;
		s = GetLastError();
		add_log(write_log_ptr_, LOG_TYPE_ERROR, "lib_load error %d", s);
		delete write_log_ptr_;
		write_log_ptr_ = nullptr;
		return -1;
	}

	//加载so透出的函数;
	open_server_ptr_ = (OPEN_SERVER)lib_function(lib_handle_, "open_server");
	if (nullptr == open_server_ptr_)
	{
		add_log(write_log_ptr_, LOG_TYPE_ERROR, "lib_function OPEN_SERVER error");
		lib_close(lib_handle_);
		delete write_log_ptr_;
		write_log_ptr_ = nullptr;
		return -1;
	}

	close_server_ptr_ = (CLOSE_SERVER)lib_function(lib_handle_, "close_server");
	if (nullptr == close_server_ptr_)
	{
		add_log(write_log_ptr_, LOG_TYPE_ERROR, "lib_function CLOSE_SERVER error");
		lib_close(lib_handle_);
		delete write_log_ptr_;
		write_log_ptr_ = nullptr;
		return -1;
	}

	version_ptr_ = (VERSION)lib_function(lib_handle_, "version");
	if (nullptr == version_ptr_)
	{
		add_log(write_log_ptr_, LOG_TYPE_ERROR, "lib_function VERSION error");
		lib_close(lib_handle_);
		delete write_log_ptr_;
		write_log_ptr_ = nullptr;
		return -1;
	}

	//显示版本
	char *version = version_ptr_();
	add_log(write_log_ptr_, LOG_TYPE_INFO, "server.dll Version is %s", version);

	//调用透出函数;
	bool open_ret = open_server_ptr_((char*)bind_ip.c_str(), 41002, (char*)log_path.c_str(), (char*)harq_path.c_str(), (char*)base_path.c_str(), &on_login, &on_progress, &on_finish, &on_disconnect, &on_error);
	if (!open_ret)
	{
		add_log(write_log_ptr_, LOG_TYPE_ERROR, "open_server error\n");
		lib_close(lib_handle_);
		delete write_log_ptr_;
		write_log_ptr_ = nullptr;
		return -1;
	}
	add_log(write_log_ptr_, LOG_TYPE_INFO, "Start server.dll Success\n");

	::Sleep(1);

	int postion = 0;
	while (1)
	{
		::Sleep(1);
	}
	delete write_log_ptr_;
	write_log_ptr_ = nullptr;
	return 0;
}

