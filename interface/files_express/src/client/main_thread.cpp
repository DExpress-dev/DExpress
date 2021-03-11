/*
 * main_thread.cpp
 *
 *  Created on: 2019年8月27日
 *      Author: fxh7622
 */

#if defined(_WIN32)
	#include "winsock2.h"
#endif
#include <time.h>
#include <string.h>
#include <stdarg.h>

#include "fcntl.h"

#include "path.h"
#include "file.h"
#include "write_log.h"
#include "rudp_public.h"

#include "udp_manager.h"
#include "main_thread.h"

file_client_thread::file_client_thread(main_thread *main_thread_ptr, int express_handle)
{
	parent_ = main_thread_ptr;
	express_handle_ = express_handle;

	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Create express_client_thread express_handle=%d", express_handle_);
	set_state(ccs_inited);
}

void file_client_thread::init()
{
	set_state(ccs_inited);
	memset(&current_sending_file_, 0, sizeof(current_sending_file_));
	current_sending_file_.hased_ = false;

	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Init express_client_thread");
	thread_ptr_ = std::thread(&file_client_thread::execute, this);
	thread_ptr_.detach();
}

file_client_thread::~file_client_thread(void)
{
	if(current_state_ == tst_runing)
	{
		current_state_ = tst_stoping;
		time_t last_timer = time(nullptr);
		int timer_interval = 0;
		while((timer_interval <= 2))
		{
			time_t current_timer = time(nullptr);
			timer_interval = static_cast<int>(difftime(current_timer, last_timer));
			if(current_state_ == tst_stoped)
			{
				break;
			}
			ustd::rudp_public::sleep_delay(100, Millisecond);
		}
	}
	free_queue();
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "express_client_thread::~express_client_thread");
}

void file_client_thread::add_queue(header head_ptr, char *data, int size, int linker_handle, const char* remote_ip, const int &remote_port)
{
	server_message_buffer *buffer_ptr = new server_message_buffer();
	buffer_ptr->message_id_ = head_ptr.protocol_id_;
	buffer_ptr->linker_handle_ = linker_handle;
	buffer_ptr->next_ = nullptr;
	buffer_ptr->size_ = size;
	memset(buffer_ptr->remote_ip_, 0, sizeof(buffer_ptr->remote_ip_));
	memcpy(buffer_ptr->remote_ip_, remote_ip, strlen(remote_ip));
	buffer_ptr->remote_port_ = remote_port;
	buffer_ptr->data_ = nullptr;

	if (size > 0)
	{
		buffer_ptr->data_ = new char[size];
		memcpy(buffer_ptr->data_, data, size);
	}

	{
		std::lock_guard<std::recursive_mutex> gurad(messasge_lock_);
		if(first_ != nullptr)
			last_->next_ = buffer_ptr;
		else
			first_ = buffer_ptr;

		last_ = buffer_ptr;
	}
}

void file_client_thread::free_queue()
{
	server_message_buffer *next_ptr = nullptr;
	while(first_ != nullptr)
	{
		next_ptr = first_->next_;
		if(first_->data_ != nullptr)
		{
			delete[] first_->data_;
		}
		delete first_;
		first_ = next_ptr;
	}
	first_ = nullptr;
	last_ = nullptr;
}

void file_client_thread::execute()
{
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "RUDP Start Create RUDP Manager Remote_ip=%s Remote_port=%d Session=%s Log=%s", remote_ip_.c_str(), remote_port_, session_.c_str(), parent_->log_.c_str());
	udp_manager_ = new udp_manager();

	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Create RUDP Client Object");
	if(nullptr == udp_manager_)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "New RUDP Client Failed");
		return;
	}
	udp_manager_->parent_ = this;
	udp_manager_->bind_ip_ = bind_ip_;
	udp_manager_->remote_ip_ = remote_ip_;
	udp_manager_->remote_port_ = remote_port_;
	udp_manager_->session_ = session_;
	udp_manager_->log_ = parent_->log_;
	udp_manager_->delay_ = delay_;
	udp_manager_->delay_interval_ = delay_timer_;

	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "RUDP Client Object init");
	udp_manager_->init();
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "RUDP Client Object init Success");

	current_state_ = tst_runing;
	while(tst_runing == current_state_)
	{
		business_dispense();
		check_send_queue();
		check_connect();
		check_state();
		ustd::rudp_public::sleep_delay(5, Millisecond);
	}

	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "main_thread current_state=%d", current_state_);
	if(nullptr != udp_manager_)
	{
		delete udp_manager_;
	}
	udp_manager_ = nullptr;
	current_state_ = tst_stoped;
}

void file_client_thread::reponse_login_fun(server_message_buffer *message)
{
	current_client_state_ = ccs_logined;
	if (nullptr != parent_->on_login_)
	{
		parent_->on_login_(server_handle_, (char*)remote_ip_.c_str(), remote_port_, (char*)session_.c_str());
	}

	//需要续传之前没有传输完毕的文件;
	std::string local_path(current_sending_file_.local_path_, sizeof(current_sending_file_.local_path_));
	std::string remote_path(current_sending_file_.remote_path_, sizeof(current_sending_file_.remote_path_));
	std::string file_name(current_sending_file_.file_name_, sizeof(current_sending_file_.file_name_));
	if(current_sending_file_.hased_)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Continue Send Local File local_path=%s remote_path=%s relative_path=%s ret=%d",
				local_path.c_str(), remote_path.c_str(), file_name.c_str());

		//判断文件是否存在;
		if (!ustd::path::is_file_exist(local_path))
		{
			main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Continue Send Local File Failed Not Found absolute_path=%s", local_path.c_str());
			return;
		}

		//得到文件大小;
		long long file_size = ustd::path::get_file_size(local_path.c_str());

		//记录当前发送的文件信息;
		current_file_state_ = cfs_configing;
		udp_manager_->send_config(file_size, local_path, remote_path, file_name);
	}
}

void file_client_thread::reponse_config_fun(server_message_buffer *message_buffer)
{
	current_file_state_ = cfs_configed;

	//解析返回;
	reponse_config* config_ptr = (reponse_config*)(message_buffer->data_);

	//得到文件名;
	std::string absolute_path(config_ptr->absolute_path_);

	//判断文件是否存在;
	if (!ustd::path::is_file_exist(absolute_path))
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Send Local File Failed Not Found absolute_path=%s", absolute_path.c_str());
		return;
	}

	//得到文件大小;
	long long file_size = ustd::path::get_file_size(absolute_path.c_str());

	//判断网络是否正常;
	if (nullptr == udp_manager_ || ccs_logined != current_client_state_)
	{
		//异常退出;
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Send Local File Failed Network Is Not Startup absolute_path=%s", absolute_path.c_str());
		return;
	}

	//打开文件
	harq_file_handle file_handle = harq_open_file(absolute_path);
	if (file_handle == INVALID_HANDLE_VALUE)
	{
		return;
	}

	int64 max = (file_size + (SINGLE_BLOCK - 1)) / SINGLE_BLOCK;
	current_file_state_ = cfs_sending;

	int len = 0;
	int64 postion = config_ptr->cur_;
	int64 offset = postion * SINGLE_BLOCK;

	//移动位置
	harq_set_postion(file_handle, offset);

	//循环读取文件
	while (true)
	{
		memset(buffer, 0, SINGLE_BLOCK);
		postion++;

		//通知进度
		if (postion % 10 == 2)
		{
			ustd::rudp_public::sleep_delay(1, Millisecond);
			if (parent_->on_progress_ != nullptr)
			{
				parent_->on_progress_(server_handle_, config_ptr->absolute_path_, static_cast<int>(max), static_cast<int>(postion));
			}
		}

		if (offset + SINGLE_BLOCK > file_size)
			len = static_cast<int>(file_size - offset);
		else
			len = SINGLE_BLOCK;


		if (len <= 0)
			break;

		int readlen = harq_read_file(file_handle, buffer, len);
		if (readlen > 0)
		{
			int ret = udp_manager_->send_file_block(max, postion, buffer, len);
			if (ret <= 0)
			{
				set_state(ccs_disconnected);
				main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Send Local File Failed send_file_block Result=%d", ret);
				harq_close_file(file_handle);
				file_handle = INVALID_HANDLE_VALUE;
				return;
			}
			offset += len;
		}
		else
		{
			set_state(ccs_disconnected);
			main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Send Local File Failed harq_read_file Result=%d", readlen);
			harq_close_file(file_handle);
			file_handle = INVALID_HANDLE_VALUE;
			return;
		}
	}

	harq_close_file(file_handle);
	file_handle = INVALID_HANDLE_VALUE;
	if (parent_->on_finish_ != nullptr)
	{
		parent_->on_finish_(server_handle_, config_ptr->absolute_path_, file_size);
	}

	current_file_state_ = cfs_inited;
	memset(&current_sending_file_, 0, sizeof(current_sending_file_));
	current_sending_file_.hased_ = false;
}

void file_client_thread::reponse_progress_fun(server_message_buffer *message_buffer)
{
	reponse_file_progress* progress_ptr = (reponse_file_progress*)(message_buffer->data_);
	if (parent_->on_progress_ != nullptr)
	{
		parent_->on_progress_(server_handle_, progress_ptr->absolute_path_, progress_ptr->max_id_, progress_ptr->cur_id_);
	}
}

void file_client_thread::reponse_finished_fun(server_message_buffer *message_buffer)
{
	reponse_file_finished* finished_ptr = (reponse_file_finished*)(message_buffer->data_);
	if (parent_->on_finish_ != nullptr)
	{
		parent_->on_finish_(server_handle_, finished_ptr->absolute_path_, finished_ptr->file_size_);
	}
}

void file_client_thread::reponse_buffer_fun(server_message_buffer *message_buffer)
{
	request_buffer* buffer_ptr = (request_buffer*)(message_buffer->data_);
	if (parent_->on_buffer_ != nullptr)
	{
		parent_->on_buffer_(server_handle_, message_buffer->data_ + sizeof(request_buffer), buffer_ptr->size_);
	}
}

void file_client_thread::add_file(const std::string &local_path, const std::string &remote_path, const std::string &file_name)
{
	std::lock_guard<std::recursive_mutex> gurad(file_lock_);

	std::shared_ptr<wait_send_file> wait_send_file_ptr(new wait_send_file);

	//绝对路径
	memset(wait_send_file_ptr->local_path_, 0, sizeof(wait_send_file_ptr->local_path_));
	memcpy(wait_send_file_ptr->local_path_, local_path.c_str(), local_path.length());
	//远端路径
	memset(wait_send_file_ptr->remote_path_, 0, sizeof(wait_send_file_ptr->remote_path_));
	memcpy(wait_send_file_ptr->remote_path_, remote_path.c_str(), remote_path.length());
	//相对路径
	memset(wait_send_file_ptr->file_name_, 0, sizeof(wait_send_file_ptr->file_name_));
	memcpy(wait_send_file_ptr->file_name_, file_name.c_str(), file_name.length());

	file_list_.push_back(wait_send_file_ptr);
}

void file_client_thread::delete_file(const std::string &local_path)
{
	std::lock_guard<std::recursive_mutex> gurad(file_lock_);

	for (auto iter = file_list_.begin(); iter != file_list_.end();)
	{
		std::shared_ptr<wait_send_file> wait_send_file_ptr = *iter;
		if (wait_send_file_ptr->local_path_ == local_path)
		{
			file_list_.erase(iter++);
		}
		else
		{
			iter++;
		}
	}
}

int file_client_thread::get_size()
{
	std::lock_guard<std::recursive_mutex> gurad(file_lock_);

	return file_list_.size();
}

void file_client_thread::free_file()
{
	std::lock_guard<std::recursive_mutex> gurad(file_lock_);

	file_list_.clear();
}

std::string file_client_thread::get_relative_path(const std::string &absolute_path, const std::string &base_path)
{
	std::string result = absolute_path;
	result = result.substr(base_path.length() + 1, absolute_path.length() - base_path.length());
	return result;
}

void file_client_thread::check_send_queue()
{
	time_t curr_time = time(nullptr);
	int second = static_cast<int>(difftime(curr_time, last_check_send_timer_));
	if (second >= 1)
	{
		last_check_send_timer_ = time(nullptr);
		if (file_list_.empty() || current_file_state_ != cfs_inited || current_client_state_ != ccs_logined)
			return;

		std::shared_ptr<wait_send_file> wait_send_file_ptr = nullptr;
		{
			std::lock_guard<std::recursive_mutex> gurad(file_lock_);
			wait_send_file_ptr = file_list_.front();
			file_list_.pop_front();
		}

		if (wait_send_file_ptr != nullptr)
		{
			std::string local_path(wait_send_file_ptr->local_path_, sizeof(wait_send_file_ptr->local_path_));
			std::string remote_path(wait_send_file_ptr->remote_path_, sizeof(wait_send_file_ptr->remote_path_));
			std::string file_name(wait_send_file_ptr->file_name_, sizeof(wait_send_file_ptr->file_name_));

			//判断文件是否存在;
			if (!ustd::path::is_file_exist(local_path))
			{
				main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Send Local File Failed Not Found local_path=%s", local_path.c_str());
				return;
			}

			//得到文件大小;
			long long file_size = ustd::path::get_file_size(local_path.c_str());

			//判断网络是否正常;
			if (nullptr == udp_manager_ || ccs_logined != current_client_state_)
			{
				//异常退出;
				main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Send Local File Failed Network Is Not Startup local_path=%s", local_path.c_str());
				return;
			}

			//记录当前发送的文件信息;
			current_file_state_ = cfs_configing;
			memset(&current_sending_file_, 0, sizeof(current_sending_file_));
			memcpy(current_sending_file_.local_path_, wait_send_file_ptr->local_path_, sizeof(wait_send_file_ptr->local_path_));
			memcpy(current_sending_file_.remote_path_, wait_send_file_ptr->remote_path_, sizeof(wait_send_file_ptr->remote_path_));
			memcpy(current_sending_file_.file_name_, wait_send_file_ptr->file_name_, sizeof(wait_send_file_ptr->file_name_));
			current_sending_file_.hased_ = true;
			udp_manager_->send_config(file_size, local_path, remote_path, file_name);
		}
	}
}

void file_client_thread::business_dispense()
{
	server_message_buffer *work_ptr = nullptr, *next_ptr = nullptr;
	{
		std::lock_guard<std::recursive_mutex> gurad(messasge_lock_);
		if(work_ptr == nullptr && first_ != nullptr)
		{
			work_ptr = first_;
			first_ = nullptr;
			last_ = nullptr;
		}
	}
	while(work_ptr != nullptr)
	{
		next_ptr = work_ptr->next_;
		switch(work_ptr->message_id_)
		{
			case HARQ_CONNECT:							//连接成功消息;
			{
				harq_connect(work_ptr);
				break;
			}
			case HARQ_DISCONNECT:						//断开连接
			{
				harq_disconnect(work_ptr);
				break;
			}
			case EXPRESS_RESPONSE_ONLINE:				//登录返回;
			{
				reponse_login_fun(work_ptr);
				break;
			}
			case EXPRESS_RESPONSE_CONFIG:				//config返回
			{
				reponse_config_fun(work_ptr);
				break;
			}
			case EXPRESS_RESPONSE_PROGRESS:				//文件传输进度返回
			{
				reponse_progress_fun(work_ptr);
				break;
			}
			case EXPRESS_RESPONSE_FINISHED:				//文件传输完毕返回
			{
				reponse_finished_fun(work_ptr);
				break;
			}
			case EXPRESS_REQUEST_BUFFER:				//接收到流数据信息
			{
				reponse_buffer_fun(work_ptr);
				break;
			}
			default:
			{
				main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Express Dispense Not Found Message Id %d", work_ptr->message_id_);
				break;
			}
		}
		if (work_ptr->data_ != nullptr && work_ptr->size_ > 0)
		{
			delete[] work_ptr->data_;
			work_ptr->data_ = nullptr;
		}
		delete work_ptr;
		work_ptr = next_ptr;
	}
}

void file_client_thread::harq_connect(server_message_buffer *message_buffer)
{
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Message ---->>>> harq_connect Success");
	set_state(ccs_conneced);

	//登录;
	std::string session = "12345";
	udp_manager_->send_login(session.c_str());
}

void file_client_thread::harq_disconnect(server_message_buffer *message_buffer)
{
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Message ---->>>> harq_disconnect");
	set_state(ccs_disconnected);
}

void file_client_thread::set_state(client_state state)
{
	if (current_client_state_ == state)
		return;

	std::lock_guard<std::recursive_mutex> gurad(state_lock_);
	current_client_state_ = state;

	if (current_client_state_ == ccs_inited)
	{
		last_disconnct_timer_ = time(nullptr);
		last_check_timer_ = time(nullptr);
		last_check_send_timer_ = time(nullptr);
	}
	else if (current_client_state_ == ccs_disconnected)
	{
		last_disconnct_timer_ = time(nullptr);
		max_disconnect_timer_ = time(nullptr) + 600;
	}
}

void file_client_thread::check_state()
{
	time_t curr_time = time(nullptr);
	if (curr_time >= max_disconnect_timer_)
	{
		if (current_client_state_ == ccs_disconnected)
		{
			//向上通知
			if (parent_->on_disconnect_ != nullptr)
			{
				parent_->on_disconnect_(server_handle_, (char*)remote_ip_.c_str(), remote_port_);
				set_state(ccs_notify_disconnected);
			}
		}
	}
}

void file_client_thread::check_connect()
{
	if (current_client_state_ == ccs_inited)
	{
		//链接RUDP的服务端;
		main_thread::get_instance()->add_log(LOG_TYPE_INFO, "****First Connect****");
		int linker_handle = udp_manager_->connect();
		if (-1 != linker_handle)
		{
			main_thread::get_instance()->add_log(LOG_TYPE_INFO, "RUDP Connect Success");
			set_state(ccs_conneced);
		}
		else
		{
			main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "RUDP Connect Failed");
			set_state(ccs_disconnected);
		}
	}
	else
	{
		time_t curr_time = time(nullptr);
		int second = static_cast<int>(difftime(curr_time, last_check_timer_));
		if (second >= CHECK_TIME)
		{
			int interval = static_cast<int>(difftime(curr_time, last_disconnct_timer_));
			if (interval >= RECONNECT_TIME && current_client_state_ == ccs_disconnected)
			{
				//判断是否小于
				if (curr_time < max_disconnect_timer_)
				{
					//链接RUDP的服务端;
					main_thread::get_instance()->add_log(LOG_TYPE_INFO, "****ReConnect****");
					int linker_handle = udp_manager_->connect();
					if (-1 != linker_handle)
					{
						main_thread::get_instance()->add_log(LOG_TYPE_INFO, "RUDP Connect Success");
						set_state(ccs_conneced);
					}
					else
					{
						main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "RUDP Connect Failed");
						set_state(ccs_disconnected);
					}
				}
			}
			last_check_timer_ = time(nullptr);
		}
	}
}

//************************************************************

main_thread::main_thread()
{
#if defined(_WIN32)
	WSADATA wsa_data;
	if (WSAStartup(0x0202, &wsa_data) != 0)
	{
		return;
	}
#endif
	write_log_ptr_ = new ustd::log::write_log(true);
}

void main_thread::init()
{
	write_log_ptr_->init("express_server", log_.c_str(), 1);
	thread_ptr_ = std::thread(&main_thread::execute, this);
	thread_ptr_.detach();
}

main_thread::~main_thread(void)
{
	if (current_state_ == tst_runing)
	{
		current_state_ = tst_stoping;
		time_t last_timer = time(nullptr);
		int timer_interval = 0;
		while ((timer_interval <= 2))
		{
			time_t current_timer = time(nullptr);
			timer_interval = static_cast<int>(difftime(current_timer, last_timer));
			if (current_state_ == tst_stoped)
			{
				break;
			}
			ustd::rudp_public::sleep_delay(100, Millisecond);
		}
	}
	add_log(LOG_TYPE_INFO, "express_client_thread::~express_client_thread");

#if defined(_WIN32)
	::WSACleanup();
#endif
}

void main_thread::execute()
{
	current_state_ = tst_runing;
	while (tst_runing == current_state_)
	{
		ustd::rudp_public::sleep_delay(100, Millisecond);
	}

	add_log(LOG_TYPE_INFO, "main_thread current_state=%d", current_state_);
	current_state_ = tst_stoped;
}

std::shared_ptr<remote_server> main_thread::get_server(const int &server_handle)
{
	std::lock_guard<std::recursive_mutex> gurad(server_lock_);

	for (auto iter = server_list_.begin(); iter != server_list_.end(); iter++)
	{
		std::shared_ptr<remote_server> remote_server_ptr = *iter;
		if (remote_server_ptr != nullptr && remote_server_ptr->server_handle_ == server_handle)
		{
			return remote_server_ptr;
		}
	}
	return nullptr;
}

std::shared_ptr<remote_server> main_thread::get_server(const std::string &remote_ip, const int &remote_port)
{
	std::lock_guard<std::recursive_mutex> gurad(server_lock_);

	for (auto iter = server_list_.begin(); iter != server_list_.end(); iter++)
	{
		std::shared_ptr<remote_server> remote_server_ptr = *iter;
		if (remote_server_ptr != nullptr && remote_server_ptr->server_ptr_ != nullptr && remote_server_ptr->server_ptr_->remote_ip_ == remote_ip && remote_server_ptr->server_ptr_->remote_port_ == remote_port)
		{
			return remote_server_ptr;
		}
	}
	return nullptr;
}

void main_thread::free_server()
{
	std::lock_guard<std::recursive_mutex> gurad(server_lock_);

	for (auto iter = server_list_.begin(); iter != server_list_.end(); iter++)
	{
		std::shared_ptr<remote_server> remote_server_ptr = *iter;
		if (remote_server_ptr->server_ptr_ != nullptr)
		{
			delete remote_server_ptr->server_ptr_;
			remote_server_ptr->server_ptr_ = nullptr;
		}
	}
	server_list_.clear();
}

void main_thread::add_server(std::shared_ptr<remote_server> remote_server_ptr)
{
	std::lock_guard<std::recursive_mutex> gurad(server_lock_);

	server_list_.push_back(remote_server_ptr);
}

void main_thread::delete_server(const int &server_handle)
{
	std::lock_guard<std::recursive_mutex> gurad(server_lock_);

	for (auto iter = server_list_.begin(); iter != server_list_.end();)
	{
		std::shared_ptr<remote_server> remote_server_ptr = *iter;
		if (remote_server_ptr->server_handle_ == server_handle)
		{
			server_list_.erase(iter++);

			delete remote_server_ptr->server_ptr_;
			remote_server_ptr->server_ptr_ = nullptr;
		}
		else
		{
			iter++;
		}
	}
}

int main_thread::get_server_handle()
{

createHandle:

	current_server_handle_++;
	if (current_server_handle_ < 0)
	{
		current_server_handle_ = 10000;
	}

	std::lock_guard<std::recursive_mutex> gurad(server_lock_);

	for (auto iter = server_list_.begin(); iter != server_list_.end(); iter++)
	{
		std::shared_ptr<remote_server> remote_server_ptr = *iter;
		if (remote_server_ptr->server_handle_ == current_server_handle_)
		{
			goto createHandle;
		}
	}

	return current_server_handle_;
}

void main_thread::add_log(const int &log_type, const char *log_text_format, ...)
{
	std::lock_guard<std::recursive_mutex> guard(log_lock_);

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

	if (nullptr != write_log_ptr_)
	{
		write_log_ptr_->write_log3(log_type, log_text);
	}
}


