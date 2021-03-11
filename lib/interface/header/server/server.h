/*
 * magpie_server.h
 *
 *  Created on: 2019年8月27日
 *      Author: fxh7622
 */

#ifndef EXPRESS_SERVER_H_
#define EXPRESS_SERVER_H_

#include <functional>
#include "interface.h"

	#if defined(_WIN32)

		#ifdef EXPRESS_SERVER_LIB
			#define EXPRESS_SERVER_LIB extern "C" _declspec(dllimport)
		#else
			#define EXPRESS_SERVER_LIB extern "C" _declspec(dllexport)
		#endif

		EXPRESS_SERVER_LIB bool open_server(char *bind_ip, int listen_port, char *log_path, char *harq_path, char *base_path, bool same_name_deleted,
			ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finished, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error);
		/*EXPRESS_SERVER_LIB bool send_file(char* remote_ip, int remote_port, char* local_file_path, char* remote_relative_path, char* file_name);
		EXPRESS_SERVER_LIB bool send_dir(char* remote_ip, int remote_port, char* local_dir_path, char* remote_relative_path);
		EXPRESS_SERVER_LIB bool send_buffer(char* remote_ip, int remote_port, char* data, int size);
		EXPRESS_SERVER_LIB int cur_waiting_size();
		EXPRESS_SERVER_LIB void stop_send(char* local_file_path);*/
		EXPRESS_SERVER_LIB void close_server();
		EXPRESS_SERVER_LIB char* version();

	#else
		extern "C"{
			bool open_server(char *bind_ip, int listen_port, char *log_path, char *harq_path, char *base_path, bool same_name_deleted,
				ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finished, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error);
			/*bool send_file(char* remote_ip, int remote_port, char* local_file_path, char* remote_relative_path, char* file_name);
			bool send_dir(char* remote_ip, int remote_port, char* local_dir_path, char* remote_relative_path);
			bool send_buffer(char* remote_ip, int remote_port, char* data, int size);
			int cur_waiting_size();
			void stop_send(char* local_file_path);*/
			void close_server();
			char* version();
		}

	#endif

#endif /* EXPRESS_SERVER_H_ */
