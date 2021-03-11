#ifndef EXPRESS_CLIENT_H_
#define EXPRESS_CLIENT_H_

#include <string>
#include <stdio.h>
#include <stdarg.h>
#include <memory>
#include "interface.h"

#if defined(_WIN32)

	#ifdef EXPRESS_CLIENT_LIB
		#define EXPRESS_CLIENT_LIB extern "C" _declspec(dllimport)
	#else
		#define EXPRESS_CLIENT_LIB extern "C" _declspec(dllexport)
	#endif

	EXPRESS_CLIENT_LIB int open_client(char* bind_ip, char* remote_ip, int remote_port, char* log, char *harq_so_path, char* session, bool encrypted,
		ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finish, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error);
	EXPRESS_CLIENT_LIB bool send_file(int express_handle, char* local_file_path, char* remote_relative_path, char* file_name);
	EXPRESS_CLIENT_LIB bool send_dir(int express_handle, char* dir_path, char* save_relative_path);
	EXPRESS_CLIENT_LIB bool send_buffer(int express_handle, char* data, int size);
	EXPRESS_CLIENT_LIB int cur_waiting_size(int express_handle);
	EXPRESS_CLIENT_LIB void stop_send(int express_handle, char* file_path);
	EXPRESS_CLIENT_LIB void close_client(int express_handle);
	EXPRESS_CLIENT_LIB char* version(void);

#else
	extern "C"{
		int open_client(char* bind_ip, char* remote_ip, int remote_port, char* log, char *harq_so_path, char* session, bool encrypted,
				ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finish, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error);
		bool send_file(int express_handle, char* local_file_path, char* remote_relative_path, char* file_name);
		bool send_dir(int express_handle, char* dir_path, char* save_relative_path);
		bool send_buffer(int express_handle, char* data, int size);
		int cur_waiting_size(int express_handle);
		void stop_send(int express_handle, char* file_path);
		void close_client(int express_handle);
		char* version(void);
	}
#endif


#endif	//EXPRESS_CLIENT_H_
