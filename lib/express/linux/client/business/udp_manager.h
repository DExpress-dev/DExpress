#pragma once

#include <thread>
#include <mutex>
#include <stdint.h>

#include "protocol.h"
#include "interface.h"
#include "harq_interface.h"
#include "harq.h"

#if defined(_WIN32)

	#include <windows.h>
	#include <Mmsystem.h>
	#include <time.h>
	#include "windows/timer_windows.h"

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

class udp_manager
{
public:
	void *parent_ = nullptr;
	udp_manager(void *parent, const std::string &bind_ip, const std::string &remote_ip, const int &remote_port, const std::string &session, const std::string &log, const bool &delay, const int &delay_timer);
	~udp_manager(void);

public:
	std::recursive_mutex consume_timer_lock_;
	int min_consume_timer_ = 0;
	int max_consume_timer_ = 0;
	void init_consume_timer();
	void set_consume_timer(const int &consume_timer);

public:
	int remote_handle_ = ERROR_HANDLE;
	int linker_handle_ = ERROR_HANDLE;

	std::string bind_ip_ = "0.0.0.0";
	std::string remote_ip_ = "";
	int remote_port_ = ERROR_PORT;
	std::string session_ = "";
	std::string log_ = "";

	bool delay_ = false;
	int delay_interval_ = 2000;

	bool init();
	int connect_server();

public:
	int connect();
	int send_login(const std::string &session);
	int send_config(const int64_t &size, const std::string &absolute_path_, const std::string &remote_path, std::string &file_name);
	int send_file_block(int64_t max, int64_t cur, char* data, int size);
	int send_buffer(char* data, int size);

public:
	lib_handle lib_handle_ = nullptr;
	HARQ_START_CLIENT harq_start_client_ptr_ = nullptr;
	HARQ_SEND_BUFFER_HANDLE harq_send_buffer_handle_ptr_ = nullptr;
	HARQ_CLOSE_HANDLE harq_close_handle_ptr_ = nullptr;
	HARQ_VERSION harq_version_ptr_ = nullptr;
	HARQ_END_SERVER harq_end_server_ptr_ = nullptr;
};
