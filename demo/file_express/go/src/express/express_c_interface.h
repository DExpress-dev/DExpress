
#include <stdbool.h>

//so透出的函数类型定义
typedef void (*EXPRESS_LOGIN)(int express_handle, char *remote_ip, int remote_port, char *session);
typedef bool (*EXPRESS_PROGRESS)(int express_handle, char *file_path, int max, int cur);
typedef void (*EXPRESS_FINISH)(int express_handle, char *file_path, long long size);
typedef void (*EXPRESS_DISCONNECT)(int express_handle, char *remote_ip, int remote_port);
typedef void (*EXPRESS_ERROR)(int express_handle, int error, char* remote_ip, int remote_port);

typedef int (*OPEN_CLIENT)(char* bind_ip, char* remote_ip, int remote_port, char* log, char *harq_so_path, char* session, bool encrypted, EXPRESS_LOGIN on_login, EXPRESS_PROGRESS on_progress, EXPRESS_FINISH on_finish, EXPRESS_DISCONNECT on_disconnect, EXPRESS_ERROR on_error);
typedef bool (*SEND_FILE)(int express_handle, char* file_path, char* save_relative_path);
typedef bool (*SEND_DIR)(int express_handle, char* dir_path, char* save_relative_path);
typedef void *(*STOP_SEND)(int express_handle, char* file_path);
typedef void (*CLOSE_CLIENT)(int express_handle);
typedef char* (*VERSION_CLIENT)();


bool init_client(char* library_path);
int start_client(char* bind_ip, char* remote_ip, int remote_port, char* log, char *harq_so_path, char* session, bool encrypted);
bool send_file(int express_handle, char* file_path, char* save_relative_path);
bool send_dir(int express_handle, char* dir_path, char* save_relative_path);
void stop_send_file(int express_handle, char* file_path);
void close_client(int express_handle);
char* version();



