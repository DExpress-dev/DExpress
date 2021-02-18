/*
 * main_thread.h
 *
 *  Created on: 2019年8月27日
 *      Author: fxh7622
 */

#ifndef MAIN_THREAD_H_
#define MAIN_THREAD_H_

#include <thread>
#include <mutex>
#include "protocol.h"
#include "interface.h"
#include "udp_manager.h"
#include "write_log.h"
#include "rudp_def.h"
#include "rudp_public.h"

const int CHECK_TIME 		= 5;
const int RECONNECT_TIME 	= 30;
const int MAX_WAIT_SIZE		= 50;

#pragma pack(push, 1)

enum client_state
{
	ccs_inited,				//初始化
	ccs_conneced,			//连接成功
	ccs_logined,			//已经登录
	ccs_disconnected,		//已经断开连接
	ccs_notify_disconnected	//已经通知上层断开了连接
};

enum client_file_state
{
	cfs_inited,			//初始化状态
	cfs_configing,		//正在发送config
	cfs_configed,		//config已经返回
	cfs_sending			//正在发送文件
};

const std::string Version = "1.0.1";
struct log_record
{
	int level_;
	char log_char_[1024];
};

struct server_message_buffer
{
	int message_id_;
	int linker_handle_;
	char *data_;
	int size_;
	char remote_ip_[16];
	int remote_port_;
	server_message_buffer *next_;
};


//****************
//Local
struct local_message_buffer
{
	int message_id_;
	char *data_;
	int size_;
	local_message_buffer *next_;
};

struct login_udp_server
{
	char bind_ip_[16];
	char remote_ip_[16];
	int remote_port_;
	char session_[255];
	bool encrypted_;
	bool delay_;
	int delay_timer_;
};

struct wait_send_file
{
	int server_handle_;
	bool hased_;
	char local_path_[PATH_LEN];		//发送端的文件绝对路径
	char remote_path_[PATH_LEN];		//接收端的相对路径
	char file_name_[PATH_LEN];			//接收端的文件名称
};


struct message_buffer
{
	int message_id_;
	int express_handle_;
	char *data_;
	int size_;
	message_buffer *next_;
};

#pragma pack(pop)

class main_thread;
class file_client_thread
{
public:
	std::string bind_ip_ = "0.0.0.0";
	std::string remote_ip_ = "";
	int remote_port_ = ERROR_PORT;
	std::string session_ = "";
	std::string libso_path_ = "";
	bool delay_ = false;
	int delay_timer_ = 2000;
	bool encrypted_ = false;

private:
	int current_linker_handle_ = ERROR_HANDLE;
	time_t login_timer_ = time(nullptr);

public:

	std::recursive_mutex state_lock_;
	client_state current_client_state_ = ccs_inited;
	time_t last_disconnct_timer_ = time(nullptr);
	time_t max_disconnect_timer_ = time(nullptr);
	void set_state(client_state state);
	void check_state();

public:
	time_t last_check_timer_;
	void check_connect();

public:
	int express_handle_ = 0;
	main_thread *parent_ = nullptr;
	file_client_thread(main_thread *main_thread_ptr, int express_handle);
	~file_client_thread(void);

public:
	char buffer[SINGLE_BLOCK] = { 0 };
	client_file_state current_file_state_ = cfs_inited;
	wait_send_file current_sending_file_;

public:
	void init();

public:
	thread_state_type current_state_ = tst_init;
	std::thread thread_ptr_;
	void execute();

public:
	server_message_buffer *first_ = nullptr, *last_ = nullptr;
	std::recursive_mutex messasge_lock_;
	void add_queue(header head_ptr, char *data, int size, int linker_handle, const char* remote_ip, const int &remote_port);
	void free_queue();

private:
	void business_dispense();
	void harq_connect(server_message_buffer *message_buffer);
	void harq_disconnect(server_message_buffer *message_buffer);

	void reponse_login_fun(server_message_buffer *message_buffer);
	void reponse_config_fun(server_message_buffer *message_buffer);
	void reponse_progress_fun(server_message_buffer *message_buffer);
	void reponse_finished_fun(server_message_buffer *message_buffer);
	void reponse_buffer_fun(server_message_buffer *message_buffer);

public:
	std::recursive_mutex file_lock_;
	std::list<std::shared_ptr<wait_send_file>> file_list_;
	void add_file(const std::string &absolute_path, const std::string &remote_path, const std::string &file_name);
	void delete_file(const std::string &absolute_path);
	int get_size();
	void free_file();

private:
	time_t last_check_send_timer_;
	void check_send_queue();

public:
	std::string get_relative_path(const std::string &absolute_path, const std::string &base_path);

public:
	//RUDP传输相关管理;
	int server_handle_ = -1;
	udp_manager *udp_manager_ = nullptr;
};


struct remote_server
{
	int server_handle_;
	file_client_thread *server_ptr_;
};

class main_thread
{
public:
	std::string log_ = "";
	std::string libso_path_ = "";
	bool start_ = false;

public:
	int current_server_handle_ = 10000;
	int get_server_handle();

public:
	std::recursive_mutex server_lock_;
	std::list<std::shared_ptr<remote_server>> server_list_;
	std::shared_ptr<remote_server> get_server(const int &server_handle);
	std::shared_ptr<remote_server> get_server(const std::string &remote_ip, const int &remote_port);
	void add_server(std::shared_ptr<remote_server> remote_server_ptr);
	void delete_server(const int &server_handle);
	void free_server();

public:
	thread_state_type current_state_ = tst_init;
	main_thread();
	~main_thread(void);

public:
	ON_EXPRESS_LOGIN on_login_ = nullptr;
	ON_EXPRESS_PROGRESS on_progress_ = nullptr;
	ON_EXPRESS_FINISH on_finish_ = nullptr;
	ON_EXPRESS_BUFFER on_buffer_ = nullptr;
	ON_EXPRESS_DISCONNECT on_disconnect_ = nullptr;
	ON_EXPRESS_ERROR on_error_ = nullptr;
	void init();

public:
	std::thread thread_ptr_;
	void execute();

public:
	//日志相关管理;
	ustd::log::write_log *write_log_ptr_ = nullptr;
	std::recursive_mutex log_lock_;
	void add_log(const int &log_type, const char *log_text_format, ...);

public:
	static main_thread *get_instance()
	{
		static main_thread *m_pInstance = nullptr;
		if (m_pInstance == nullptr)
		{
			m_pInstance = new main_thread();
		}
		return m_pInstance;
	}
};


#endif /* MAIN_THREAD_H_ */
