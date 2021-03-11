/*
 * main_thread.h
 *
 *  Created on: 2019年8月27日
 *      Author: fxh7622
 */

#ifndef FILE_EXPRESS_SERVER_H_
#define FILE_EXPRESS_SERVER_H_

#include <thread>
#include <mutex>
#include "protocol.h"
#include "interface.h"

#include "write_log.h"
#include "udp_manager.h"

#include "rudp_def.h"
#include "rudp_public.h"
#include "file.h"

const std::string Version = "2.0.1";

struct log_record
{
	int level_;
	char log_char_[1024];
};

struct server_message_buffer
{
	unsigned short message_id_;
	int linker_handle_;
	char remote_ip_[16];
	int remote_port_;
	char *data_;
	int size_;
	server_message_buffer *next_;
};

struct file_config_info
{
	int linker_handle_;
	std::string local_path_;
	std::string remote_path_;
	int64_t file_size_;
	harq_file_handle file_config_handle_;
	int64_t current_block_;
	time_t last_timer_;
};
typedef std::map<int, std::shared_ptr<file_config_info>> file_config_map;

class file_server_thread
{
public:
	std::string bind_ip_ = "0.0.0.0";
	int listen_port_ = ERROR_PORT;
	std::string log_path_ = "";
	std::string libso_path_ = "";
	bool delete_same_named_ = false;

public:
	thread_state_type current_state_ = tst_init;
	file_server_thread();
	~file_server_thread(void);

public:
	ON_EXPRESS_LOGIN on_login_ = nullptr;
	ON_EXPRESS_FINISH on_finished_ = nullptr;
	ON_EXPRESS_PROGRESS on_progress_ = nullptr;
	ON_EXPRESS_BUFFER on_buffer_ = nullptr;
	ON_EXPRESS_DISCONNECT on_disconnect_ = nullptr;
	ON_EXPRESS_ERROR on_error_ = nullptr;
	void init(std::string bind_ip, int listen_port, std::string log_path, std::string so_path, std::string base_path, bool delete_same_named,
		ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finished, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error);

public:
	udp_manager *udp_manager_ = nullptr;
	std::thread thread_ptr_;
	void execute();

public:
	std::string path_base_ = "";
	std::string get_absolute_path(const std::string &relative_path, std::string &name);

private:
	void business_dispense();
	void express_login(server_message_buffer *message_buffer);
	void express_config(server_message_buffer *message_buffer);
	void express_file(server_message_buffer *message_buffer);
	void express_logout(server_message_buffer *message_buffer);
	void express_buffer(server_message_buffer *message_buffer);

public:
	server_message_buffer *first_ = nullptr, *last_ = nullptr;
	std::recursive_mutex messasge_lock_;
	void add_queue(header head_ptr, char *data, int size, int linker_handle, const char* remote_ip, const int &remote_port);
	void free_queue();

public:
	//文件管理;
	std::recursive_mutex file_config_lock_;
	file_config_map file_config_map_;
	void add_file_config(const int &linker_handle, const std::string &local_path, const std::string &remote_path, const int64_t &file_size, const int64_t &cur);
	std::shared_ptr<file_config_info> find_file_config(const int &linker_handle);
	void delete_file_config(const int &linker_handle);
	void free_file_config();

public:
	//文件检测;
	time_t last_check_file_timer_;
	void check_file_configs();
	void check_time_out();

public:
	std::recursive_mutex log_lock_;
	ustd::log::write_log *write_log_ptr_ = nullptr;
	void add_log(const int &log_type, const char *log_text_format, ...);

private:
	//返回信息;
	void response_config(const std::string &remote_path, int64_t &cur, const int &result, const int &linker_handle);
	void response_progress(const std::string &remote_path, const int &max_id, const int &cur_id, const int &linker_handle);
	void response_finished(const std::string &remote_path, const int &linker_handle, const long long &file_size);

public:
	int send_buffer_to_client(const unsigned short &message_id, char *data, const int &size, const int &linker_handle);

public:
	static file_server_thread *get_instance()
	{
		static file_server_thread *m_pInstance = nullptr;
		if (m_pInstance == nullptr)
		{
			m_pInstance = new file_server_thread();
		}
		return m_pInstance;
	}
};

#endif /* FILE_EXPRESS_SERVER_H_ */
