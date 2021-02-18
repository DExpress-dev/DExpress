
#include <thread>
#include <mutex>

#include "udp_manager.h"
#include "main_thread.h"
#include "protocol.h"
#include "interface.h"


void on_connect(const char* remote_ip, const int &remote_port, const int &linker_handle, const long long &time_stamp)
{
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Linker OnConnect Remote_Ip=%s Remote_Port=%d Linker_Handle=%d TimeStamp=%lld", remote_ip, remote_port, linker_handle, time_stamp);

	std::string remoteIp(remote_ip);
	std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(remoteIp, remote_port);
	if (remote_server_ptr != nullptr)
	{
		header header_ptr;
		memset(&header_ptr, 0, sizeof(header_ptr));
		header_ptr.protocol_id_ = HARQ_CONNECT;
		remote_server_ptr->server_ptr_->add_queue(header_ptr, nullptr, 0, linker_handle, remote_ip, remote_port);
	}
}

bool on_recv(const char* data, const int &size, const int &linker_handle, const char* remote_ip, const int &remote_port, const int &consume_timer)
{
	header *header_ptr = (header *)(data);
	if (nullptr == header_ptr)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Linker OnRecv Failed Remote_Ip=%s Remote_Port=%d Linker_Handle=%d Size=%d", remote_ip, remote_port, linker_handle, size);
		return false;
	}

	std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(remote_ip, remote_port);
	if (remote_server_ptr != nullptr)
	{
		remote_server_ptr->server_ptr_->add_queue(*header_ptr, (char*)data, size, linker_handle, remote_ip, remote_port);
	}
	return true;
}

void on_disconnect(const int &linker_handle, const char* remote_ip, const int &remote_port)
{
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Linker Disconnect Remote_Ip=%s Remote_Port=%d Linker_Handle=%d", remote_ip, remote_port, linker_handle);

	std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(remote_ip, remote_port);
	if (remote_server_ptr != nullptr)
	{
		header header_ptr;
		memset(&header_ptr, 0, sizeof(header_ptr));
		header_ptr.protocol_id_ = HARQ_DISCONNECT;
		remote_server_ptr->server_ptr_->add_queue(header_ptr, nullptr, 0, linker_handle, remote_ip, remote_port);
	}
}

void on_error(const int &error, const int &linker_handle, const char* remote_ip, const int &remote_port)
{
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Linker Error Remote_Ip=%s Remote_Port=%d Linker_Handle=%d error=%d", remote_ip, remote_port, linker_handle, error);

	std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(remote_ip, remote_port);
	if (remote_server_ptr != nullptr)
	{
		header header_ptr;
		memset(&header_ptr, 0, sizeof(header_ptr));
		header_ptr.protocol_id_ = HARQ_DISCONNECT;
		remote_server_ptr->server_ptr_->add_queue(header_ptr, nullptr, 0, linker_handle, remote_ip, remote_port);
	}
}

void on_rto(const char* remote_ip, const int &remote_port, const int &local_rto, const int &remote_rto)
{
}

void on_rate(const char* remote_ip, const int &remote_port, const unsigned int &send_rate, const unsigned int &recv_rate)
{
}

udp_manager::udp_manager(void *parent, const std::string &bind_ip, const std::string &remote_ip, const int &remote_port, const std::string &session, const std::string &log, const bool &delay, const int &delay_timer)
{
	parent_ = parent;

	bind_ip_ = bind_ip;
	remote_ip_ = remote_ip;
	remote_port_ = remote_port;
	session_ = session;
	log_ = log;

	delay_ = delay;
	delay_interval_ = delay_timer;
	init_consume_timer();
}

udp_manager::~udp_manager(void)
{
}

int udp_manager::connect_server()
{
	if(harq_start_client_ptr_ != nullptr)
	{
		bool encrypted = ((file_client_thread*)parent_)->encrypted_;
		main_thread::get_instance()->add_log(LOG_TYPE_INFO, "harq_start_client_ptr_ ip=%s port=%d", remote_ip_.c_str(), remote_port_);
		int linker_handle;
		int ret = harq_start_client_ptr_((char*)log_.c_str(),
			(char*)bind_ip_.c_str(),
			(char*)remote_ip_.c_str(),
			remote_port_,
			2000,
			delay_,
			delay_interval_,
			encrypted,
			512 * KB,
			10 * MB,
			&on_connect,
			&on_recv,
			&on_disconnect,
			&on_error,
			&on_rto,
			&on_rate,
			&linker_handle);

		linker_handle_ = linker_handle;
		main_thread::get_instance()->add_log(LOG_TYPE_INFO, "connect_server harq_start_client_ptr ret=%d", ret);
		return ret;
	}
	return -1;
}

bool udp_manager::init()
{
	lib_handle_ = lib_load(((file_client_thread*)parent_)->libso_path_.c_str());
	if (nullptr == lib_handle_)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "Load so Falied libso_path=%s error=%s", ((file_client_thread*)parent_)->libso_path_.c_str(), lib_error());
		return false;
	}

	//加载so透出的函数;
	harq_start_client_ptr_ = (HARQ_START_CLIENT)lib_function(lib_handle_, "harq_start_client");
	harq_send_buffer_handle_ptr_ = (HARQ_SEND_BUFFER_HANDLE)lib_function(lib_handle_, "harq_send_buffer_handle");
	harq_close_handle_ptr_ = (HARQ_CLOSE_HANDLE)lib_function(lib_handle_, "harq_close_handle");
	harq_version_ptr_ = (HARQ_VERSION)lib_function(lib_handle_, "harq_version");
	harq_end_server_ptr_ = (HARQ_END_SERVER)lib_function(lib_handle_, "harq_end_server");

	if (nullptr == harq_start_client_ptr_)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "dlsym harq_start_client_ptr_ error\n");
		lib_close(lib_handle_);
		return false;
	}
	if (nullptr == harq_send_buffer_handle_ptr_)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "dlsym harq_send_buffer_handle error\n");
		lib_close(lib_handle_);
		return false;
	}
	if (nullptr == harq_close_handle_ptr_)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "dlsym harq_close_handle error\n");
		lib_close(lib_handle_);
		return false;
	}
	if (nullptr == harq_version_ptr_)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "dlsym harq_version_ptr_ error\n");
		lib_close(lib_handle_);
		return false;
	}
	if (nullptr == harq_end_server_ptr_)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "dlsym harq_end_server error\n");
		lib_close(lib_handle_);
		return false;
	}
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Load so Success libharq_path=%s", ((file_client_thread*)parent_)->libso_path_.c_str());
	return true;
}

int udp_manager::connect()
{
	remote_handle_ = connect_server();
	if (-1 == remote_handle_)
	{
		if(parent_ != nullptr)
		{
			main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "begin_client Failed Udp_Ip=%s Udp_Port=%d", remote_ip_.c_str(), remote_port_);
			return remote_handle_;
		}
	}
	else
	{
		if(parent_ != nullptr)
		{
			main_thread::get_instance()->add_log(LOG_TYPE_INFO, "begin_client Success Udp_Ip=%s Udp_Port=%d", remote_ip_.c_str(), remote_port_);
			return remote_handle_;
		}
	}
	return -1;
}

int udp_manager::send_login(const std::string &session)
{
	if (-1 == linker_handle_)
	{
		return -1;
	}

	request_login *login = new request_login;
	memset(login, 0, sizeof(request_login));
	login->header_.protocol_id_ = EXPRESS_REQUEST_ONLINE;
	memset(login->session_, 0, sizeof(login->session_));
	memcpy(login->session_, session.c_str(), session.length());

	int ret = harq_send_buffer_handle_ptr_(remote_handle_, (char*)login, sizeof(request_login), linker_handle_);
	delete login;
	login = nullptr;
	return ret;
}

int udp_manager::send_config(const int64 &size, const std::string &local_path, const std::string &remote_path, std::string &file_name)
{
	if (-1 == linker_handle_)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_config Failed Linker_handle is ERROR_HANDLE");
		return -1;
	}

	request_config config;
	config.header_.protocol_id_ = EXPRESS_REQUEST_CONFIG;

	memset(config.local_path_, 0, sizeof(config.local_path_));
	memcpy(config.local_path_, local_path.c_str(), local_path.length());

	memset(config.remote_path_, 0, sizeof(config.remote_path_));
	memcpy(config.remote_path_, remote_path.c_str(), remote_path.length());

	memset(config.remote_name_, 0, sizeof(config.remote_name_));
	memcpy(config.remote_name_, file_name.c_str(), file_name.length());

	config.file_size_ = size;
	return harq_send_buffer_handle_ptr_(remote_handle_, (char*)&config, sizeof(config), linker_handle_);
}

int udp_manager::send_file_block(int64 max, int64 cur, char* data, int size)
{
	if(-1 == linker_handle_)
	{
		return -1;
	}

	request_file file_block;
	file_block.header_.protocol_id_ = EXPRESS_REQUEST_FILE;
	file_block.max_ = max;
	file_block.cur_ = cur;
	file_block.size_ = size;
	memset(file_block.data_, 0, sizeof(file_block.data_));
	memcpy(file_block.data_, data, size);

	int ret = harq_send_buffer_handle_ptr_(remote_handle_, (char*)&file_block, sizeof(file_block), linker_handle_);
	return ret;
}

int udp_manager::send_buffer(char* data, int size)
{
	if(-1 == linker_handle_)
		return -1;

	char *buffer = new char[size + sizeof(request_buffer)];
	memset(buffer, 0, size + sizeof(request_buffer));

	request_buffer request_buffer_obj;
	request_buffer_obj.header_.protocol_id_ = EXPRESS_REQUEST_BUFFER;
	request_buffer_obj.size_ = size;

	int offset = 0;
	memcpy(buffer + offset, &request_buffer_obj, sizeof(request_buffer_obj));

	offset += sizeof(request_buffer_obj);
	memcpy(buffer + offset, data, size);

	int ret = harq_send_buffer_handle_ptr_(remote_handle_, buffer, size + sizeof(request_buffer), linker_handle_);

	delete[] buffer;
	buffer = nullptr;

	return ret;
}

void udp_manager::init_consume_timer()
{
	std::lock_guard<std::recursive_mutex> gurad(consume_timer_lock_);
	min_consume_timer_ = 0;
	max_consume_timer_ = 0;
}
	
void udp_manager::set_consume_timer(const int &consume_timer)
{
	std::lock_guard<std::recursive_mutex> gurad(consume_timer_lock_);

	if(min_consume_timer_ >= consume_timer)
	{
		min_consume_timer_ = consume_timer;
	}

	if(max_consume_timer_ <= consume_timer)
	{
		max_consume_timer_ = consume_timer;
	}
}
