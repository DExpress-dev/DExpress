/*
 * magpie_server.cpp
 *
 *  Created on: 2019Äê8ÔÂ27ÈÕ
 *      Author: fxh7622
 */

#include <string.h>

#include "server.h"

#include "protocol.h"
#include "interface.h"
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

