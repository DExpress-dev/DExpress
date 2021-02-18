#include "stdafx.h"
#include <stdio.h>
#include <stdlib.h>
#include <iostream>
#include <fcntl.h>

#include "client.h"
#include "protocol.h"
#include "interface.h"
#include "path.h"
#include "main_thread.h"

const char* VERSION = "1.0.5";

EXPRESS_CLIENT_LIB int open_client(char* bind_ip,
	char* remote_ip,
	int remote_port,
	char* log,
	char *harq_so_path,
	char* session,
	bool encrypted,
	ON_EXPRESS_LOGIN on_login,
	ON_EXPRESS_PROGRESS on_progress,
	ON_EXPRESS_FINISH on_finish,
	ON_EXPRESS_DISCONNECT on_disconnect,
	ON_EXPRESS_ERROR on_error)
{
	std::string tmp_bind_ip(bind_ip, strlen(bind_ip));
	std::string tmp_remote_ip(remote_ip);
	std::string tmp_log(log);
	std::string libso_path(harq_so_path);
	std::string tmp_session(session);

	//启动主线程;
	if (!main_thread::get_instance()->start_)
	{
		main_thread::get_instance()->log_ = tmp_log;
		main_thread::get_instance()->libso_path_ = libso_path;
		main_thread::get_instance()->start_ = true;
		main_thread::get_instance()->on_login_ = on_login;
		main_thread::get_instance()->on_progress_ = on_progress;
		main_thread::get_instance()->on_finish_ = on_finish;
		main_thread::get_instance()->on_disconnect_ = on_disconnect;
		main_thread::get_instance()->on_error_ = on_error;
		main_thread::get_instance()->init();
	}
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "---->>>> open_client interface log=%s version=%s remote_ip=%s remote_port=%d", log, VERSION, remote_ip, remote_port);

	//根据express_handle来得到传输对象
	std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(tmp_remote_ip, remote_port);
	if (remote_server_ptr != nullptr)
	{
		//判断各种参数是否一致;
		if (remote_server_ptr->server_ptr_ == nullptr)
		{
			main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "open_client Failed server_ptr is NULL remote_ip=%s remote_port=%d Found", remote_ip, remote_port);
			return -1000;
		}
		else if (remote_server_ptr->server_ptr_->encrypted_ != encrypted)
		{
			main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "open_client Failed server_ptr_->encrypted_ != encrypted remote_ip=%s remote_port=%d Found", remote_ip, remote_port);
			return -1001;
		}
		else
		{
			if (remote_server_ptr->server_ptr_->udp_manager_ == nullptr)
			{
				main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "open_client Failed server_ptr_->udp_manager is NULL remote_ip=%s remote_port=%d Found", remote_ip, remote_port);
				return -1002;
			}

			if (remote_server_ptr->server_ptr_->current_client_state_ == ccs_logined)
			{
				main_thread::get_instance()->add_log(LOG_TYPE_INFO, "open_client Connect Success Use Exist Client!");
				return remote_server_ptr->server_handle_;
			}

			// 连接断开了，需要重新连接;
			const int timer_space = 100;
			int flag_count = 0;
			int connect_count = (5 * 1000) / timer_space;
			while (1)
			{
				//延时;
				ustd::rudp_public::sleep_delay(timer_space, Millisecond);
				flag_count++;

				//状态判断
				if (nullptr != remote_server_ptr->server_ptr_->udp_manager_ && ERROR_HANDLE != remote_server_ptr->server_ptr_->udp_manager_->linker_handle_ && remote_server_ptr->server_ptr_->current_client_state_ == ccs_logined)
				{
					main_thread::get_instance()->add_log(LOG_TYPE_INFO, "open_client Connect Success!");
					return remote_server_ptr->server_handle_;
				}

				//链接失败
				if (flag_count >= connect_count)
				{
					main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "open_client Failed Connect Timeout");

					if (remote_server_ptr != nullptr)
					{
						//删除
						main_thread::get_instance()->delete_server(remote_server_ptr->server_handle_);
					}
					return -1003;
				}
			}
		}
	}
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "User Use open_client Interface remote_ip=%s remote_port=%d", remote_ip, remote_port);

	//创建新的对象;
	std::shared_ptr<remote_server> new_remote_server_ptr(new remote_server);
	new_remote_server_ptr->server_handle_ = main_thread::get_instance()->get_server_handle();
	express_client_thread *express_ptr = new express_client_thread(main_thread::get_instance(), new_remote_server_ptr->server_handle_);
	express_ptr->bind_ip_ = tmp_bind_ip;
	express_ptr->delay_ = false;
	express_ptr->delay_timer_ = 2000;
	express_ptr->encrypted_ = encrypted;
	express_ptr->libso_path_ = libso_path;
	express_ptr->remote_ip_ = tmp_remote_ip;
	express_ptr->remote_port_ = remote_port;
	express_ptr->session_ = tmp_session;
	new_remote_server_ptr->server_ptr_ = express_ptr;
	new_remote_server_ptr->server_ptr_->server_handle_ = new_remote_server_ptr->server_handle_;
	main_thread::get_instance()->add_server(new_remote_server_ptr);

	//使用新的对象进行连接;
	express_ptr->init();
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "Init server_ptr Params remote_ip=%s remote_port=%d log=%s so_path=%s session=%s", remote_ip, remote_port, log, harq_so_path, session);

	//等待连接返回;
	const int timer_space = 100;
	int flag_count = 0;
	int connect_count = (5 * 1000) / timer_space;
	while (1)
	{
		//延时;
		ustd::rudp_public::sleep_delay(timer_space, Millisecond);
		flag_count++;

		//状态判断
		if (nullptr != express_ptr->udp_manager_ && ERROR_HANDLE != express_ptr->udp_manager_->linker_handle_ && express_ptr->current_client_state_ == ccs_logined)
		{
			main_thread::get_instance()->add_log(LOG_TYPE_INFO, "open_client Connect Success!");
			return new_remote_server_ptr->server_handle_;
		}

		//链接失败
		if (flag_count >= connect_count)
		{
			main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "open_client Failed Connect Timeout");

			if (new_remote_server_ptr != nullptr)
			{
				//删除
				main_thread::get_instance()->delete_server(new_remote_server_ptr->server_handle_);
			}
			return -2;
		}
	}
}

EXPRESS_CLIENT_LIB bool send_file(int express_handle, char* file_path, char* save_relative_path)
{
	main_thread::get_instance()->add_log(LOG_TYPE_INFO, "send_file express_handle=%d file_path=%s save_relative_path=%s", express_handle, file_path, save_relative_path);

	if (strcmp(file_path, "") == 0)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed file_path is NULL", file_path);
		return false;
	}

	//判断文件是否存在;
	bool exist = ustd::path::is_file_exist(file_path);
	if (!exist)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed File Not Found %s", file_path);
		return false;
	}

	//根据express_handle来得到传输对象
	std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(express_handle);
	if (remote_server_ptr == nullptr)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed express_handle Not Found %d", express_handle);
		return false;
	}

	//判断是否正常;
	if (remote_server_ptr->server_ptr_ == nullptr || remote_server_ptr->server_ptr_->udp_manager_ == nullptr || remote_server_ptr->server_ptr_->udp_manager_->linker_handle_ == ERROR_HANDLE)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed linker_handle is Null or udp_manager is Null");
		return false;
	}

	//判断当前状态是否可以进行文件传输;
	if (remote_server_ptr->server_ptr_->current_client_state_ != ccs_logined)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed Client Current State Is Not ccs_logined");
		return false;
	}

	//向发送队列中加入发送请求;
	std::string remote_path(save_relative_path);
	std::string absolute_path(file_path);
	std::string base_path = ustd::path::get_full_path(absolute_path);
	std::string relative_path = remote_server_ptr->server_ptr_->get_relative_path(absolute_path, base_path);
	remote_server_ptr->server_ptr_->add_file(absolute_path, relative_path, remote_path);
	return true;
}

EXPRESS_CLIENT_LIB bool send_dir(int express_handle, char* dir_path, char* save_relative_path)
{
	//判断文件是否存在;
	bool exist = ustd::path::is_directory_exist(dir_path);
	if (!exist)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed File Not Found %s", dir_path);
		return false;
	}

	//根据express_handle来得到传输对象
	std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(express_handle);
	if (remote_server_ptr == nullptr)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed express_handle Not Found %d", express_handle);
		return false;
	}

	//判断是否正常;
	if (remote_server_ptr->server_ptr_ == nullptr || remote_server_ptr->server_ptr_->udp_manager_ == nullptr || remote_server_ptr->server_ptr_->udp_manager_->linker_handle_ == ERROR_HANDLE)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed linker_handle is Null or udp_manager is Null");
		return false;
	}

	//判断当前状态是否可以进行文件传输;
	if (remote_server_ptr->server_ptr_->current_client_state_ != ccs_logined)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed Client Current State Is Not ccs_logined");
		return false;
	}

	//获取本地目录下的所有文件信息
	std::vector<std::string> file_vector;
	ustd::path::get_dir_all(dir_path, file_vector);
	if (file_vector.empty())
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed Not Found File From Dir");
		return false;
	}

	//将所有文件放到等待发送队列中
	for (auto iter = file_vector.begin(); iter != file_vector.end(); iter++)
	{
		std::string absolute_path = *iter;
		std::string base_path(dir_path);
		std::string remote_path(save_relative_path);
		std::string relative_path = remote_server_ptr->server_ptr_->get_relative_path(absolute_path, base_path);
		remote_server_ptr->server_ptr_->add_file(absolute_path, relative_path, remote_path);
	}
	return true;
}

EXPRESS_CLIENT_LIB void stop_send(int express_handle, char* file_path)
{
	//根据express_handle来得到传输对象
	std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(express_handle);
	if (remote_server_ptr == nullptr)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "stop_send Failed express_handle Not Found %d", express_handle);
		return;
	}

	//判断是否正常;
	if (remote_server_ptr->server_ptr_ == nullptr)
	{
		main_thread::get_instance()->add_log(LOG_TYPE_ERROR, "stop_send Failed server_ptr_ is Null");
		return ;
	}

	//删除文件
	std::string absolute_path(file_path);
	remote_server_ptr->server_ptr_->delete_file(absolute_path);
}

EXPRESS_CLIENT_LIB void close_client(int express_handle)
{
}

EXPRESS_CLIENT_LIB char* version()
{
	return (char*)VERSION;
}
