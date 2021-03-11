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
		EXPRESS_SERVER_LIB void close_server();
		EXPRESS_SERVER_LIB char* version();

	#else
		extern "C"{
			bool open_server(char *bind_ip, int listen_port, char *log_path, char *harq_path, char *base_path, bool same_name_deleted,
				ON_EXPRESS_LOGIN on_login, ON_EXPRESS_PROGRESS on_progress, ON_EXPRESS_FINISH on_finished, ON_EXPRESS_BUFFER on_buffer, ON_EXPRESS_DISCONNECT on_disconnect, ON_EXPRESS_ERROR on_error);
			void close_server();
			char* version();
		}

	#endif

#endif /* EXPRESS_SERVER_H_ */
