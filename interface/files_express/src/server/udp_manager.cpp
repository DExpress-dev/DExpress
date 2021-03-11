
#if defined(_WIN32)
#include "winsock2.h"
#endif

#include <thread>
#include <mutex>
#include <algorithm>

#include "udp_manager.h"
#include "main_thread.h"
#include "protocol.h"
#include "interface.h"
#include "path.h"

bool on_checkip(const char* remote_ip, const int &remote_port)
{
	file_server_thread::get_instance()->add_log(LOG_TYPE_INFO, "on_checkip---->>>>");
	if (file_server_thread::get_instance()->udp_manager_ != nullptr)
	{
		return file_server_thread::get_instance()->udp_manager_->handle_checkip(remote_ip, remote_port);
	}
	return true;
}

bool on_recv(const char* data, const int &size, const int &linker_handle, const char* remote_ip, const int &remote_port, const int &consume_timer)
{
	header *header_ptr = (header *)(data);
	if (nullptr == header_ptr)
	{
		file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "rudp_on_recv Failed linker_handle=%d size=%d", linker_handle, size);
		return false;
	}

	file_server_thread::get_instance()->add_queue(*header_ptr, (char*)data, size, linker_handle, remote_ip, remote_port);

	if (file_server_thread::get_instance()->udp_manager_ != nullptr)
	{
		file_server_thread::get_instance()->udp_manager_->handle_recv(data, size, linker_handle, remote_ip, remote_port, consume_timer);
	}

	return true;
}

void on_disconnect(const int &linker_handle, const char* remote_ip, const int &remote_port)
{
	file_server_thread::get_instance()->add_log(LOG_TYPE_INFO, "on_disconnect---->>>> %d", linker_handle);
	header header_ptr;
	memset(&header_ptr, 0, sizeof(header_ptr));
	header_ptr.protocol_id_ = HARQ_DISCONNECT;
	file_server_thread::get_instance()->add_queue(header_ptr, nullptr, 0, linker_handle, remote_ip, remote_port);

	if (file_server_thread::get_instance()->udp_manager_ != nullptr)
	{
		file_server_thread::get_instance()->udp_manager_->handle_disconnect(linker_handle, remote_ip, remote_port);
	}
}

void on_error(const int &error, const int &linker_handle, const char* remote_ip, const int &remote_port)
{
	file_server_thread::get_instance()->add_log(LOG_TYPE_INFO, "on_error---->>>> %d", error);
	header header_ptr;
	memset(&header_ptr, 0, sizeof(header_ptr));
	header_ptr.protocol_id_ = HARQ_DISCONNECT;
	file_server_thread::get_instance()->add_queue(header_ptr, nullptr, 0, linker_handle, remote_ip, remote_port);

	if (file_server_thread::get_instance()->udp_manager_ != nullptr)
	{
		file_server_thread::get_instance()->udp_manager_->handle_error(error, linker_handle, remote_ip, remote_port);
	}
}

void on_rto(const char* remote_ip, const int &remote_port, const int &local_rto, const int &remote_rto)
{
	if (file_server_thread::get_instance()->udp_manager_ != nullptr)
	{
		file_server_thread::get_instance()->udp_manager_->handle_rto(remote_ip, remote_port, local_rto, remote_rto);
	}
}

void on_rate(const char* remote_ip, const int &remote_port, const unsigned int &send_rate, const unsigned int &recv_rate)
{
	if (file_server_thread::get_instance()->udp_manager_ != nullptr)
	{
		file_server_thread::get_instance()->udp_manager_->handle_rto(remote_ip, remote_port, send_rate, recv_rate);
	}
}

udp_manager::udp_manager(void *parent)
{
	parent_ = parent;
	load_so();
}

udp_manager::~udp_manager(void)
{
}

void udp_manager::set_option(const std::string &attribute, const std::string &value)
{
	std::string tmp_attribute = attribute;
	transform(tmp_attribute.begin(), tmp_attribute.end(), tmp_attribute.begin(), ::tolower);

	if("bind_ip" == tmp_attribute)
	{
		bind_ip_ = value;
	}
	else if("log" == tmp_attribute)
	{
		log_ = value;
	}
	else if("harq_so_path" == tmp_attribute)
	{
		harq_so_path_ = value;
	}
}

void udp_manager::set_option(const std::string &attribute, const int &value)
{
	std::string tmp_attribute = attribute;
	transform(tmp_attribute.begin(), tmp_attribute.end(), tmp_attribute.begin(), ::tolower);

	if("listen_port" == tmp_attribute)
	{
		listen_port_ = value;
	}
	else if("delay_interval" == tmp_attribute)
	{
		delay_interval_ = value;
	}
}

void udp_manager::set_option(const std::string &attribute, const bool &value)
{
	std::string tmp_attribute = attribute;
	transform(tmp_attribute.begin(), tmp_attribute.end(), tmp_attribute.begin(), ::tolower);

	if("delay" == tmp_attribute)
	{
		delay_ = value;
	}
}

bool udp_manager::load_so()
{
	if(!ustd::path::is_file_exist(((file_server_thread*)parent_)->libso_path_))
	{
		((file_server_thread*)parent_)->add_log(LOG_TYPE_ERROR, "Load so Falied Not Found libso_path=%s error=%s", ((file_server_thread*)parent_)->libso_path_.c_str(), lib_error());
		return false;
	}

	lib_handle_ = lib_load(((file_server_thread*)parent_)->libso_path_.c_str());
	if(nullptr == lib_handle_)
	{
		((file_server_thread*)parent_)->add_log(LOG_TYPE_ERROR, "Load so Falied libso_path=%s error=%s", ((file_server_thread*)parent_)->libso_path_.c_str(), lib_error());
		return false;
	}

	//加载so透出的函数;
	harq_start_server_ptr_ = (HARQ_START_SERVER)lib_function(lib_handle_, "harq_start_server");
	harq_send_buffer_handle_ptr_ = (HARQ_SEND_BUFFER_HANDLE)lib_function(lib_handle_, "harq_send_buffer_handle");
	harq_close_handle_ptr_ = (HARQ_CLOSE_HANDLE)lib_function(lib_handle_, "harq_close_handle");
	harq_version_ptr_ = (HARQ_VERSION)lib_function(lib_handle_, "harq_version");
	harq_end_server_ptr_ = (HARQ_END_SERVER)lib_function(lib_handle_, "harq_end_server");
	if(nullptr == harq_start_server_ptr_)
	{
		((file_server_thread*)parent_)->add_log(LOG_TYPE_ERROR, "dlsym harq_start_server_ptr_ error\n");
		lib_close(lib_handle_);
		return false;
	}
	if(nullptr == harq_send_buffer_handle_ptr_)
	{
		((file_server_thread*)parent_)->add_log(LOG_TYPE_ERROR, "dlsym harq_send_buffer_handle error\n");
		lib_close(lib_handle_);
		return false;
	}
	if(nullptr == harq_close_handle_ptr_)
	{
		((file_server_thread*)parent_)->add_log(LOG_TYPE_ERROR, "dlsym harq_close_handle_ptr_ error\n");
		lib_close(lib_handle_);
		return false;
	}
	if(nullptr == harq_version_ptr_)
	{
		((file_server_thread*)parent_)->add_log(LOG_TYPE_ERROR, "dlsym harq_version_ptr_ error\n");
		lib_close(lib_handle_);
		return false;
	}
	if(nullptr == harq_end_server_ptr_)
	{
		((file_server_thread*)parent_)->add_log(LOG_TYPE_ERROR, "dlsym harq_end_server_ptr_ error\n");
		lib_close(lib_handle_);
		return false;
	}
	return true;
}

bool udp_manager::start_server()
{
	if(harq_start_server_ptr_ != nullptr)
	{
		express_handle_ = harq_start_server_ptr_((char*)log_.c_str(), (char*)bind_ip_.c_str(), listen_port_, delay_, delay_interval_, 512 * KB, 10 * MB, &on_checkip, &on_recv, &on_disconnect, &on_error, &on_rto, &on_rate);
		if (express_handle_ > 0)
			return true;
		else
			return false;
	}
	return false;
}

bool udp_manager::init()
{
	bool ret = start_server();
	return ret;
}

int udp_manager::send_buffer(char* data, int size, int linker_handle)
{
	if(-1 == linker_handle)
	{
		return -1;
	}
	return harq_send_buffer_handle_ptr_(express_handle_, data, size, linker_handle);
}

int udp_manager::send_login_response(const int &linker_handle)
{
	if (-1 == linker_handle)
		return -1;

	reponse_login login;
	login.header_.protocol_id_ = EXPRESS_RESPONSE_ONLINE;
	login.result_ = 0;
	return harq_send_buffer_handle_ptr_(express_handle_, (char*)&login, sizeof(login), linker_handle);
}

bool udp_manager::handle_checkip(const char* remote_ip, const int &remote_port)
{
	return true;
}

bool udp_manager::handle_recv(const char* data, const int &size, const int &linker_handle, const char* remote_ip, const int &remote_port, const int &consume_timer)
{
	return true;
}

void udp_manager::handle_disconnect(const int &linker_handle, const char* remote_ip, const int &remote_port)
{

}

void udp_manager::handle_error(const int &error, const int &linker_handle, const char* remote_ip, const int &remote_port)
{

}

void udp_manager::handle_rto(const char* remote_ip, const int &remote_port, const int &local_rto, const int &remote_rto)
{

}

void udp_manager::handle_rate(const char* remote_ip, const int &remote_port, const unsigned int &send_rate, const unsigned int &recv_rate)
{

}

void udp_manager::add_handle_ipport(const int &linker_handle, const char* remote_ip, const int &remote_port)
{
	std::lock_guard<std::recursive_mutex> gurad(handle_ipport_lock_);

	std::shared_ptr<handle_ipport> handle_ipport_ptr(new handle_ipport);
	handle_ipport_ptr->linker_handle_ = linker_handle;

	std::string tmp_remote_ip(remote_ip);
	handle_ipport_ptr->remote_ip_ = tmp_remote_ip;
	handle_ipport_ptr->remote_port_ = remote_port;
	handle_ipport_list_.push_back(handle_ipport_ptr);
}

std::shared_ptr<handle_ipport> udp_manager::find_handle_ipport(const char* remote_ip, const int &remote_port)
{
	std::lock_guard<std::recursive_mutex> gurad(handle_ipport_lock_);

	std::string tmp_remote_ip(remote_ip);
	for (auto iter = handle_ipport_list_.begin(); iter != handle_ipport_list_.end(); iter++)
	{
		std::shared_ptr<handle_ipport> handle_ipport_ptr = *iter;
		if (tmp_remote_ip == handle_ipport_ptr->remote_ip_ && remote_port == handle_ipport_ptr->remote_port_)
		{
			return *iter;
		}
	}
	return nullptr;
}

void udp_manager::delete_handle_ipport(const int &linker_handle)
{
	std::lock_guard<std::recursive_mutex> gurad(handle_ipport_lock_);

	for (auto iter = handle_ipport_list_.begin(); iter != handle_ipport_list_.end();)
	{
		std::shared_ptr<handle_ipport> handle_ipport_ptr = *iter;
		if (linker_handle == handle_ipport_ptr->linker_handle_)
		{
			handle_ipport_list_.erase(iter++);
		}
		else
			iter++;
	}
}

void udp_manager::free_handle_ipport()
{
	std::lock_guard<std::recursive_mutex> gurad(handle_ipport_lock_);
	handle_ipport_list_.clear();
}