/*
 * magpie_server.cpp
 *
 *  Created on: 2019��8��27��
 *      Author: fxh7622
 */

#include <string.h>

#include "server.h"

#include "protocol.h"
#include "interface.h"
#include "path.h"

#include "main_thread.h"

const char* VERSION = "1.1.2";

#if defined(_WIN32)
	EXPRESS_SERVER_LIB bool open_server(char *bind_ip, int listen_port, char *log_path, char *harq_path, char *base_path, bool same_name_deleted,
		ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finished, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error)
#else
	bool open_server(char *bind_ip, int listen_port, char *log_path, char *harq_path, char *base_path, bool same_name_deleted,
			ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finished, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error)
#endif
	{
		std::string tmp_bind_ip(bind_ip);
		std::string log(log_path);
		std::string libso_path(harq_path);
		std::string tmp_base_path(base_path);
		file_server_thread::get_instance()->init(tmp_bind_ip, listen_port, log, libso_path, tmp_base_path, same_name_deleted, on_login, on_progress, on_finished, on_buffer, on_disconnect, on_error);
		return true;
	}
/*
#if defined(_WIN32)
	EXPRESS_SERVER_LIB bool send_file(char* remote_ip, int remote_port, char* local_file_path, char* remote_relative_path, char* file_name)
#else
	bool send_file(char* remote_ip, int remote_port, char* local_file_path, char* remote_relative_path, char* file_name)
#endif
	{
		//�ж��Ƿ�Ϊ��
		if (strcmp(local_file_path, "") == 0)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed local_file_path is NULL", local_file_path);
			return false;
		}

		//�ж��ļ��Ƿ����;
		bool exist = ustd::path::is_file_exist(local_file_path);
		if (!exist)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed File Not Found %s", local_file_path);
			return false;
		}

		//�ж������Ƿ����;
		if (nullptr == file_server_thread::get_instance()->udp_manager_)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed File Not Found %s", local_file_path);
			return false;
		}

		//���ݷ��͵�Զ��ip�Ͷ˿��ж��Ƿ����;
		std::shared_ptr<handle_ipport> handle_ipport_ptr = file_server_thread::get_instance()->udp_manager_->find_handle_ipport(remote_ip, remote_port);
		if (nullptr == handle_ipport_ptr)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed File Not Found %s", local_file_path);
			return false;
		}

		//�ж��Ƿ�ȴ��ļ�����������
		if (remote_server_ptr->server_ptr_->get_size() > MAX_WAIT_SIZE)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed wait file size too flow");
			return false;
		}

		//�жϵ�ǰ״̬�Ƿ���Խ����ļ�����;
		if (remote_server_ptr->server_ptr_->current_client_state_ != ccs_logined)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed Client Current State Is Not ccs_logined");
			return false;
		}

		//���Ͷ����м��뷢������;
		std::string absolute_path(local_file_path);
		std::string remote_path(remote_relative_path);
		std::string remote_file_name(file_name);
		remote_server_ptr->server_ptr_->add_file(absolute_path, remote_path, remote_file_name);
		return true;
	}

#if defined(_WIN32)
	EXPRESS_SERVER_LIB bool send_dir(char* remote_ip, int remote_port, char* local_dir_path, char* save_relative_path)
#else
	bool send_dir(char* remote_ip, int remote_port, char* dir_path, char* save_relative_path)
#endif
	{
		//�ж��ļ��Ƿ����;
		bool exist = ustd::path::is_directory_exist(local_dir_path);
		if (!exist)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed File Not Found %s", dir_path);
			return false;
		}



		//����express_handle���õ��������
		std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(express_handle);
		if (remote_server_ptr == nullptr)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed express_handle Not Found %d", express_handle);
			return false;
		}

		//�ж��Ƿ�����;
		if (remote_server_ptr->server_ptr_ == nullptr || remote_server_ptr->server_ptr_->udp_manager_ == nullptr || remote_server_ptr->server_ptr_->udp_manager_->linker_handle_ == -1)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed linker_handle is Null or udp_manager is Null");
			return false;
		}

		//�ж��Ƿ�ȴ��ļ�����������
		if (remote_server_ptr->server_ptr_->get_size() > MAX_WAIT_SIZE)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_file Failed wait file size too flow");
			return false;
		}

		//�жϵ�ǰ״̬�Ƿ���Խ����ļ�����;
		if (remote_server_ptr->server_ptr_->current_client_state_ != ccs_logined)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed Client Current State Is Not ccs_logined");
			return false;
		}

		//��ȡ����Ŀ¼�µ������ļ���Ϣ
		std::vector<std::string> file_vector;
		ustd::path::get_dir_all(dir_path, file_vector);
		if (file_vector.empty())
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_dir Failed Not Found File From Dir");
			return false;
		}

		//�������ļ��ŵ��ȴ����Ͷ�����
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

#if defined(_WIN32)
	EXPRESS_SERVER_LIB bool send_buffer(char* remote_ip, int remote_port, char* data, int size)
#else
	bool send_buffer(char* remote_ip, int remote_port, char* data, int size)
#endif
	{
		//�жϷ��ͳ����Ƿ���ȷ;
		if (size <= 0)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_buffer Failed size is %d", size);
			return false;
		}

		//�ж�data�Ƿ�Ϊ��
		if (nullptr == data)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_buffer Failed data is NULL");
			return false;
		}

		//����express_handle���õ��������
		std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(express_handle);
		if (remote_server_ptr == nullptr)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_buffer Failed express_handle Not Found %d", express_handle);
			return false;
		}

		//�ж��Ƿ�����;
		if (remote_server_ptr->server_ptr_ == nullptr || remote_server_ptr->server_ptr_->udp_manager_ == nullptr || remote_server_ptr->server_ptr_->udp_manager_->linker_handle_ == -1)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "send_buffer Failed linker_handle is Null or udp_manager is Null");
			return false;
		}

		//������ʽ����
		int ret = remote_server_ptr->server_ptr_->udp_manager_->send_buffer(data, size);
		if (ret == size)
			return true;
		else
			return false;
	}


#if defined(_WIN32)
	EXPRESS_SERVER_LIB void stop_send(char* remote_ip, int remote_port, char* local_file_path)
#else
	void stop_send(char* remote_ip, int remote_port, char* local_file_path)
#endif
	{
		//����express_handle���õ��������
		std::shared_ptr<remote_server> remote_server_ptr = main_thread::get_instance()->get_server(express_handle);
		if (remote_server_ptr == nullptr)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "stop_send Failed express_handle Not Found %d", express_handle);
			return;
		}

		//�ж��Ƿ�����;
		if (remote_server_ptr->server_ptr_ == nullptr)
		{
			file_server_thread::get_instance()->add_log(LOG_TYPE_ERROR, "stop_send Failed server_ptr_ is Null");
			return;
		}

		//ɾ���ļ�
		std::string absolute_path(file_path);
		remote_server_ptr->server_ptr_->delete_file(absolute_path);
	}*/

#if defined(_WIN32)
	EXPRESS_SERVER_LIB void close_server()
#else
	void close_server(void)
#endif
	{
	}

#if defined(_WIN32)
	EXPRESS_SERVER_LIB char* version(void)
#else
	char* version(void)
#endif
	{
		return (char*)VERSION;
	}

