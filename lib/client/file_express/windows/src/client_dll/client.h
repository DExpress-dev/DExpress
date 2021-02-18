#ifndef EXPRESS_CLIENT_H_
#define EXPRESS_CLIENT_H_

#include <string>
#include <stdio.h>
#include <stdarg.h>
#include <memory>
#include "interface.h"

#ifdef EXPRESS_CLIENT_LIB
	#define EXPRESS_CLIENT_LIB extern "C" _declspec(dllimport) 
#else
	#define EXPRESS_CLIENT_LIB extern "C" _declspec(dllexport) 
#endif

// �򿪿ͻ���;
// ������remote_ip�������IP��ַ��remote_port������˶˿ڣ�log����־����·����harq_so_path��harq��dll���·�� session��session��Ϣ��������Ժ��ԣ� encrypted���������Ƿ�̬���ܣ�Ĭ��Ϊ�����ܣ�
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
	ON_EXPRESS_ERROR on_error);

// �����ļ�
// ������
//express_handle��������
//file_path������ı����ļ�������·����
//save_relative_path���Է������·�������·����
EXPRESS_CLIENT_LIB bool send_file(int express_handle, char* file_path, char* save_relative_path);

// ����Ŀ¼
// ������
//express_handle��������
//dir_path�������Ŀ¼������·����
//save_relative_path���Է������·�������·����
EXPRESS_CLIENT_LIB bool send_dir(int express_handle, char* dir_path, char* save_relative_path);

//ֹͣ����
// ������
//express_handle��������
//file_path����Ҫɾ�����ļ�
EXPRESS_CLIENT_LIB void stop_send(int express_handle, char* file_path);

// �ر�����;
// ������
//express_handle��������
EXPRESS_CLIENT_LIB void close_client(int express_handle);

// �汾��Ϣ
EXPRESS_CLIENT_LIB char* version(void);


#endif	//EXPRESS_CLIENT_H_