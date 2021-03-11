#pragma once

#include <thread>
#include <mutex>
#include <stdint.h>
#include <list>

#include "load.h"
#include "protocol.h"
#include "interface.h"
#include "harq_interface.h"
#include "harq.h"

struct handle_ipport
{
	int linker_handle_;
	std::string remote_ip_;
	int remote_port_;
};
typedef std::list<std::shared_ptr<handle_ipport>> handle_ipport_list;

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
	//客户端连接管理;
	std::recursive_mutex handle_ipport_lock_;
	handle_ipport_list handle_ipport_list_;
	void add_handle_ipport(const int &linker_handle, const char* remote_ip, const int &remote_port);
	std::shared_ptr<handle_ipport> find_handle_ipport(const char* remote_ip, const int &remote_port);
	void delete_handle_ipport(const int &linker_handle);
	void free_handle_ipport();

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

public:
	bool handle_checkip(const char* remote_ip, const int &remote_port);
	bool handle_recv(const char* data, const int &size, const int &linker_handle, const char* remote_ip, const int &remote_port, const int &consume_timer);
	void handle_disconnect(const int &linker_handle, const char* remote_ip, const int &remote_port);
	void handle_error(const int &error, const int &linker_handle, const char* remote_ip, const int &remote_port);
	void handle_rto(const char* remote_ip, const int &remote_port, const int &local_rto, const int &remote_rto);
	void handle_rate(const char* remote_ip, const int &remote_port, const unsigned int &send_rate, const unsigned int &recv_rate);

private:
	bool load_so();
};
