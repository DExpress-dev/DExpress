/*
 * main_thread.cpp
 *
 *  Created on: 2019年8月27日
 *      Author: fxh7622
 */

#if defined(_WIN32)
#include "winsock2.h"
#endif

#include "main_thread.h"
#include <time.h>
#include <fstream>
#include <string.h>
#include "path.h"
#include <fcntl.h>
#include <stdarg.h>

#include "write_log.h"
#include "file.h"

file_server_thread::file_server_thread()
{
	write_log_ptr_ = new ustd::log::write_log(true);
}

void file_server_thread::init(std::string bind_ip, int listen_port, std::string log_path, std::string so_path, std::string base_path, bool delete_same_named,
	ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finished, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error)
{
	bind_ip_ = bind_ip;
	listen_port_ = listen_port;
	log_path_ = log_path;
	libso_path_ = so_path;
	path_base_ = base_path;
	delete_same_named_ = delete_same_named;
	write_log_ptr_->init("express_server", log_path_.c_str(), 1);

	add_log(LOG_TYPE_INFO, "Start Express Server Version=%s listen_port=%d log_path=%s libso_path=%s", Version.c_str(), listen_port_, log_path_.c_str(), libso_path_.c_str());
	on_login_ = on_login;
	on_finished_ = on_finished;
	on_progress_ = on_progress;
	on_buffer_ = on_buffer;
	on_disconnect_ = on_disconnect;
	on_error_ = on_error;

	add_log(LOG_TYPE_INFO, "Create Express Server Main Thread");
	thread_ptr_ = std::thread(&file_server_thread::execute, this);
	thread_ptr_.detach();
}

file_server_thread::~file_server_thread(void)
{
	if(current_state_ == tst_runing)
	{
		current_state_ = tst_stoping;
		time_t last_timer = time(nullptr);
		int timer_interval = 0;
		while((timer_interval <= 5))
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
	if(nullptr != udp_manager_)
	{
		delete udp_manager_;
		udp_manager_ = nullptr;
	}
	free_queue();
	free_file_config();
}

void file_server_thread::add_queue(header head_ptr, char *data, int size, int linker_handle, const char* remote_ip, const int &remote_port)
{
	server_message_buffer *buffer_ptr = new server_message_buffer();
	buffer_ptr->message_id_ = head_ptr.protocol_id_;
	buffer_ptr->linker_handle_ = linker_handle;
	buffer_ptr->next_ = nullptr;
	buffer_ptr->size_ = size;
	buffer_ptr->data_ = nullptr;
	memset(buffer_ptr->remote_ip_, 0, sizeof(buffer_ptr->remote_ip_));
	memcpy(buffer_ptr->remote_ip_, remote_ip, strlen(remote_ip));
	buffer_ptr->remote_port_ = remote_port;
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

void file_server_thread::free_queue()
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

void file_server_thread::execute()
{
	add_log(LOG_TYPE_INFO, "Create Express Server RUDP Manager");
	udp_manager_ = new udp_manager(this);
	udp_manager_->set_option("bind_ip", bind_ip_);
	udp_manager_->set_option("log", log_path_);
	udp_manager_->set_option("harq_so_path", libso_path_);
	udp_manager_->set_option("listen_port", listen_port_);
	udp_manager_->set_option("delay", false);
	udp_manager_->set_option("delay_interval", 2000);
	udp_manager_->init();

	current_state_ = tst_runing;
	while(tst_runing == current_state_)
	{
		business_dispense();
		check_file_configs();
		ustd::rudp_public::sleep_delay(5, Millisecond);
	}
	current_state_ = tst_stoped;
}

void file_server_thread::business_dispense()
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
			case EXPRESS_REQUEST_ONLINE:				//请求上线
			{
				express_login(work_ptr);
				break;
			}
			case EXPRESS_REQUEST_CONFIG:				//请求配置
			{
				express_config(work_ptr);
				break;
			}
			case EXPRESS_REQUEST_FILE:					//请求发送文件
			{
				express_file(work_ptr);
				break;
			}
			case EXPRESS_REQUEST_BUFFER:				//接收到流数据信息
			{
				express_buffer(work_ptr);
				break;
			}
			case HARQ_DISCONNECT:						//连接断开
			{
				express_logout(work_ptr);
				break;
			}
			default:
			{
				add_log(LOG_TYPE_ERROR, "Not Found Message Id %d", work_ptr->message_id_);
				break;
			}
		}
		if (work_ptr->data_ != nullptr && work_ptr->size_ > 0)
		{
			delete[] work_ptr->data_;
		}
		delete work_ptr;
		work_ptr = next_ptr;
	}
}

void file_server_thread::express_login(server_message_buffer *message_buffer)
{
	request_login* login = (request_login*)(message_buffer->data_);
	std::string session(login->session_);
	add_log(LOG_TYPE_INFO, "Express Client Login linker_handle=%d ip=%s port=%d session=%s",
		message_buffer->linker_handle_, message_buffer->remote_ip_, message_buffer->remote_port_, session.c_str());

	udp_manager_->send_login_response(message_buffer->linker_handle_);
	if(nullptr != on_login_)
	{
		on_login_(10001, message_buffer->remote_ip_, message_buffer->remote_port_, (char*)session.c_str());
	}
}

std::string file_server_thread::get_absolute_path(const std::string &relative_path, std::string &name)
{
	char buffer[4 * 1024] = {0};
	sprintf(buffer, "%s/%s", path_base_.c_str(), relative_path.c_str());
	std::string new_dir(buffer);
	new_dir  = path_base_ + path_separator + relative_path;
	if (!ustd::path::is_directory_exist(new_dir))
	{
		//生成新的目录;
		ustd::path::create_directory(new_dir.c_str());
	}

	char buffer2[4 * 1024] = { 0 };
	sprintf(buffer2, "%s/%s", new_dir.c_str(), name.c_str());
	std::string new_path(buffer2);

	return new_path;
}

void file_server_thread::response_config(const std::string &remote_path, int64_t &cur, const int &result, const int &linker_handle)
{
	reponse_config config;
	memset(&config, 0, sizeof(config));
	config.header_.protocol_id_ = EXPRESS_RESPONSE_CONFIG;
	memcpy(config.absolute_path_, remote_path.c_str(), remote_path.length());
	config.cur_ = cur;
	config.result_ = result;

	int ret = udp_manager_->send_buffer((char*)&config, sizeof(config), linker_handle);
	if (ret <= 0)
	{
		add_log(LOG_TYPE_ERROR, "response_config failed linker_handle=%d ret=%d", linker_handle, ret);
		return;
	}
}

void file_server_thread::response_progress(const std::string &remote_path, const int &max_id, const int &cur_id, const int &linker_handle)
{
	reponse_file_progress progress;
	memset(&progress, 0, sizeof(progress));
	progress.header_.protocol_id_ = EXPRESS_RESPONSE_PROGRESS;
	memcpy(progress.absolute_path_, remote_path.c_str(), remote_path.length());
	progress.cur_id_ = cur_id;
	progress.max_id_ = max_id;

	int ret = udp_manager_->send_buffer((char*)&progress, sizeof(progress), linker_handle);
	if (ret <= 0)
	{
		add_log(LOG_TYPE_ERROR, "response_progress failed linker_handle=%d ret=%d", linker_handle, ret);
		return;
	}
}

void file_server_thread::response_finished(const std::string &remote_path, const int &linker_handle, const long long &file_size)
{
	reponse_file_finished finished;
	memset(&finished, 0, sizeof(finished));
	finished.header_.protocol_id_ = EXPRESS_RESPONSE_FINISHED;
	memcpy(finished.absolute_path_, remote_path.c_str(), remote_path.length());
	finished.file_size_ = file_size;

	int ret = udp_manager_->send_buffer((char*)&finished, sizeof(finished), linker_handle);
	if (ret <= 0)
	{
		add_log(LOG_TYPE_ERROR, "response_finished failed absolute_path=%s linker_handle=%d ret=%d", remote_path.c_str(), linker_handle, ret);
		return;
	}
}

void file_server_thread::express_config(server_message_buffer *message_buffer)
{
	request_config* config = (request_config*)(message_buffer->data_);

	std::string client_local_path(config->local_path_, sizeof(config->local_path_));
	std::string remote_path(config->remote_path_, sizeof(config->remote_path_));
	std::string file_name(config->remote_name_, sizeof(config->remote_name_));

	//得到文件绝对路径;
	std::string local_path = get_absolute_path(remote_path, file_name);

	int64_t cur = 0;
	if(delete_same_named_)
	{
		//监测文件是否存在;
		bool exists = ustd::path::is_file_exist(local_path);
		if (exists)
		{
			ustd::path::remove_file(local_path);
		}
	}
	else
	{
		//监测文件是否存在;
		bool exists = ustd::path::is_file_exist(local_path);
		if (exists)
		{
			long long file_size = ustd::path::get_file_size(local_path);
			cur = file_size / SINGLE_BLOCK;
			add_log(LOG_TYPE_INFO, "Express Config Found File Exist file_path=%s cur=%lld", local_path.c_str(), cur);
		}
	}

	//判断绝对目录是否存在;
	std::string dest_dir = ustd::path::get_full_path(local_path);
	if (dest_dir != "")
	{
		if (!ustd::path::is_directory_exist(dest_dir))
		{
			ustd::path::create_directory(dest_dir.c_str());
		}
	}

	//判断是否已经存在;
	std::shared_ptr<file_config_info> file_config_ptr = find_file_config(message_buffer->linker_handle_);
	if (file_config_ptr != nullptr)
	{
		if (file_config_ptr->file_config_handle_ != INVALID_HANDLE_VALUE)
		{
			//已经传输了一个文件，需要关闭之前的文件信息;
			harq_close_file(file_config_ptr->file_config_handle_);
			file_config_ptr->file_config_handle_ = INVALID_HANDLE_VALUE;
		}

		file_config_ptr->file_size_ = config->file_size_;
		file_config_ptr->local_path_ = local_path;
		file_config_ptr->remote_path_ = client_local_path;
		file_config_ptr->last_timer_ = time(nullptr);
		file_config_ptr->current_block_ = cur;
	}
	else
	{
		add_file_config(message_buffer->linker_handle_, local_path, client_local_path, config->file_size_, cur);
	}
	response_config(client_local_path, cur, 0, message_buffer->linker_handle_);
}

void file_server_thread::add_file_config(const int &linker_handle, const std::string &local_path, const std::string &remote_path, const int64_t &file_size, const int64_t &cur)
{
	std::lock_guard<std::recursive_mutex> gurad(file_config_lock_);

	std::shared_ptr<file_config_info> file_config_ptr(new file_config_info);
	file_config_ptr->linker_handle_ = linker_handle;
	file_config_ptr->file_size_ = file_size;
	file_config_ptr->local_path_ = local_path;
	file_config_ptr->remote_path_ = remote_path;
	file_config_ptr->last_timer_ = time(nullptr);
	file_config_ptr->current_block_ = cur;
	file_config_ptr->file_config_handle_ = INVALID_HANDLE_VALUE;
	file_config_map_.insert(std::make_pair(linker_handle, file_config_ptr));
}

void file_server_thread::delete_file_config(const int &linker_handle)
{
	std::lock_guard<std::recursive_mutex> gurad(file_config_lock_);

	file_config_map::iterator iter = file_config_map_.find(linker_handle);
	if (iter != file_config_map_.end())
	{
		std::shared_ptr<file_config_info> file_config_ptr = iter->second;

		if (INVALID_HANDLE_VALUE != file_config_ptr->file_config_handle_)
		{
			harq_close_file(file_config_ptr->file_config_handle_);
			file_config_ptr->file_config_handle_ = INVALID_HANDLE_VALUE;
		}
		file_config_map_.erase(iter);
	}
}

std::shared_ptr<file_config_info> file_server_thread::find_file_config(const int &linker_handle)
{
	std::lock_guard<std::recursive_mutex> gurad(file_config_lock_);

	file_config_map::iterator iter = file_config_map_.find(linker_handle);
	if (iter != file_config_map_.end())
	{
		return iter->second;
	}
	return nullptr;
}

void file_server_thread::free_file_config()
{
	std::lock_guard<std::recursive_mutex> gurad(file_config_lock_);
	file_config_map_.clear();
}

void file_server_thread::check_time_out()
{
	std::lock_guard<std::recursive_mutex> gurad(file_config_lock_);

	time_t current_timer = time(nullptr);
	for (auto iter = file_config_map_.begin(); iter != file_config_map_.end();)
	{
		std::shared_ptr<file_config_info> file_config_ptr = iter->second;

		int seconds = abs(static_cast<int>(difftime(current_timer, file_config_ptr->last_timer_)));
		if (seconds > 10)
		{
			if (INVALID_HANDLE_VALUE != file_config_ptr->file_config_handle_)
			{
				harq_close_file(file_config_ptr->file_config_handle_);
				file_config_ptr->file_config_handle_ = INVALID_HANDLE_VALUE;
			}
			iter = file_config_map_.erase(iter);

			add_log(LOG_TYPE_INFO, "Close File Path %s And Delete It From Map", file_config_ptr->local_path_.c_str());
		}
		else
		{
			iter++;
		}
	}
}

void file_server_thread::check_file_configs()
{
	time_t current_timer = time(nullptr);
	int seconds = abs(static_cast<int>(difftime(current_timer, last_check_file_timer_)));
	if (seconds >= 1)
	{
		check_time_out();
		last_check_file_timer_ = time(nullptr);
	}
}

void file_server_thread::express_file(server_message_buffer *message_buffer)
{
	std::shared_ptr<file_config_info> file_config_ptr = find_file_config(message_buffer->linker_handle_);
	if (nullptr == file_config_ptr)
		return;

	//判断文件是否打开;
	if (INVALID_HANDLE_VALUE == file_config_ptr->file_config_handle_)
	{
		//判断文件是否存在
		if (ustd::path::is_file_exist(file_config_ptr->local_path_))
		{
			file_config_ptr->file_config_handle_ = harq_open_file(file_config_ptr->local_path_);
		}
		else
		{
			file_config_ptr->file_config_handle_ = harq_create_file(file_config_ptr->local_path_);
		}

		if (INVALID_HANDLE_VALUE == file_config_ptr->file_config_handle_)
		{
			add_log(LOG_TYPE_ERROR, "Express File Open File Failed file_path=%s", file_config_ptr->local_path_.c_str());
			return;
		}
	}

	request_file* file_buffer = (request_file*)(message_buffer->data_);
	int64_t start_postion = (file_buffer->cur_ - 1) * SINGLE_BLOCK;

	//判断接收的数据是否是当前的值+1
	if (file_config_ptr->current_block_ != file_buffer->cur_ - 1)
	{
		//写入文件失败;
		add_log(LOG_TYPE_ERROR, "Express File Write File Failed current_block=%lld cur - 1=%lld", file_config_ptr->current_block_, file_buffer->cur_ - 1);
		return;
	}
	file_config_ptr->current_block_ = file_buffer->cur_;

	int ret = static_cast<int>(harq_postion_write_file(file_config_ptr->file_config_handle_, start_postion, file_buffer->data_, file_buffer->size_));
	if (ret != file_buffer->size_)
	{
		//写入文件失败;
		add_log(LOG_TYPE_ERROR, "Express File Write File Failed file_path=%s postion=%lld size=%d ret=%d", file_config_ptr->local_path_.c_str(), start_postion, file_buffer->size_, ret);
		return;
	}
	file_config_ptr->last_timer_ = time(nullptr);


	//落地;
	if (file_buffer->cur_ % 10 == 0)
	{
		if (on_progress_ != nullptr)
		{
			on_progress_(10001, (char*)file_config_ptr->local_path_.c_str(), static_cast<int>(file_buffer->max_), static_cast<int>(file_buffer->cur_));
		}

		//向客户端返回进度信息;
		response_progress(file_config_ptr->remote_path_, file_buffer->max_, file_buffer->cur_, message_buffer->linker_handle_);
	}

	//判断是否文件传输已经完毕;
	if (file_buffer->max_ == file_buffer->cur_)
	{
		//写文件完成
		harq_close_file(file_config_ptr->file_config_handle_);
		file_config_ptr->file_config_handle_ = INVALID_HANDLE_VALUE;

		//判断文件是否完成;
		int64_t file_size = ustd::path::get_file_size(file_config_ptr->local_path_.c_str());
		if (file_size == file_config_ptr->file_size_)
		{

			//向上层通知结束;
			if (on_finished_ != nullptr)
			{
				long long file_size = ustd::path::get_file_size(file_config_ptr->local_path_.c_str());
				on_finished_(10001, (char*)file_config_ptr->local_path_.c_str(), file_size);
				return;
			}

			//向客户端返回完成信息;
			response_finished(file_config_ptr->remote_path_, message_buffer->linker_handle_, file_size);
		}
		else
		{
			add_log(LOG_TYPE_ERROR, "Express Recv File Failed Save Path=%s Client Size=%lld Recv Size=%lld", file_config_ptr->local_path_.c_str(), file_config_ptr->file_size_, file_size);
		}
	}
}

void file_server_thread::express_logout(server_message_buffer *message_buffer)
{
	add_log(LOG_TYPE_INFO, "Express Client Logout linker_handle=%d ip=%s port=%d", message_buffer->linker_handle_, message_buffer->remote_ip_, message_buffer->remote_port_);
	delete_file_config(message_buffer->linker_handle_);

	if(nullptr != on_disconnect_)
	{
		on_disconnect_(10001, message_buffer->remote_ip_, message_buffer->remote_port_);
	}
}

void file_server_thread::express_buffer(server_message_buffer *message_buffer)
{
	request_buffer* buffer_ptr = (request_buffer*)(message_buffer->data_);
	if (on_buffer_ != nullptr)
	{
		on_buffer_(10001, message_buffer->data_ + sizeof(request_buffer), buffer_ptr->size_);
	}
}

void file_server_thread::add_log(const int &log_type, const char *log_text_format, ...)
{
	std::lock_guard<std::recursive_mutex> guard_log(log_lock_);

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

	if(nullptr != write_log_ptr_)
		write_log_ptr_->write_log3(log_type, log_text);
}

int file_server_thread::send_buffer_to_client(const unsigned short &message_id, char *data, const int &size, const int &linker_handle)
{
	header header_ptr;
	memset(&header_ptr, 0, sizeof(header_ptr));
	header_ptr.protocol_id_ = message_id;

	int buffer_len = size + sizeof(header_ptr);
	char *buffer_ptr = new char[buffer_len];
	memcpy(buffer_ptr, &header_ptr, sizeof(header_ptr));

	if (size > 0)
		memcpy(buffer_ptr + sizeof(header_ptr), data, size);

	udp_manager_->send_buffer(buffer_ptr, buffer_len, linker_handle);

	delete[] buffer_ptr;
	return size;
}

