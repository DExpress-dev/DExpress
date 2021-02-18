#pragma once

#include <thread>
#include <mutex>
#include <stdint.h>

#include "load.h"
#include "protocol.h"
#include "interface.h"
#include "harq_interface.h"
#include "harq.h"

class udp_manager
{
public:
	void *parent_ = nullptr;
	udp_manager(void *parent);
	~udp_manager(void);

public:
	void set_option(const std::string &attribute, const std::string &value);
	void set_option(const std::string &attribute, const int &value);
	void set_option(const std::string &attribute, const bool &value);

public:
	std::string bind_ip_ = "0.0.0.0";
	std::string log_ = "log";
	std::string harq_so_path_ = "";
	int listen_port_ = 41002;
	bool delay_ = false;
	int delay_interval_ = 2000;

public:
	int express_handle_ = -1;
	lib_handle lib_handle_ = nullptr;
	HARQ_START_SERVER harq_start_server_ptr_ = nullptr;
	HARQ_SEND_BUFFER_HANDLE harq_send_buffer_handle_ptr_ = nullptr;
	HARQ_CLOSE_HANDLE harq_close_handle_ptr_ = nullptr;
	HARQ_VERSION harq_version_ptr_ = nullptr;
	HARQ_END_SERVER harq_end_server_ptr_ = nullptr;

public:
	bool start_server();
	bool init();
	int send_buffer(char* data, int size, int linker_handle);
	int send_login_response(const int &linker_handle);

private:
	bool load_so();
};
