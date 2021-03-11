
#include <stdbool.h>


/*Harq回调函数定义*/

//函数说明：远端地址连接回调
//作用：可以用来限制是否允许远端进行连接
//参数：addr：远端地址
//返回值：bool: false：不允许连接，true：允许连接
typedef bool (*checkip_function)(char* remote_ip, int remote_port);

//函数说明：连接远端结果回调
//作用：可以用来限制是否允许远端进行连接
//参数：addr：远端地址，linker_handle：连接远端的句柄
//返回值：bool: false：不允许连接，true：允许连接
typedef void (*connect_function)(char* remote_ip, int remote_port, int linker_handle, long long time_stamp);

//函数说明：接收数据回调
//作用：通知上层已经接收到数据
//参数：data：数据，size：数据长度，linker_handle：接收数据使用的句柄，addr：数据的远端地址，consume_timer：此数据经过harq传输的耗时时长(毫秒)
//返回值：无
typedef bool (*receive_function)(char* data, int size, int linker_handle, char* remote_ip, int remote_port, int consume_timer);

//函数说明：远端连接断开回调
//作用：通知上层远端连接已经断开
//参数：linker_handle：接收数据使用的句柄，addr：数据的远端地址
//返回值：无
typedef void (*disconnect_function)(int linker_handle, char* remote_ip, int remote_port);

//函数说明：错误回调
//作用：通知上层harq底层发生了错误
//参数：error：错误id，linker_handle：接收数据使用的句柄，addr：数据的远端地址
//返回值：无
typedef void (*error_function)(int error, int linker_handle, char* remote_ip, int remote_port);

//函数说明：rto回调
//作用：通知上层当前harq的rto信息
//参数：addr：数据的远端地址，local_rto：数据的近端RTO，remote_rto：数据的远端RTO
//返回值：无
typedef void(*rto_function)(char* remote_ip, int remote_port, int local_rto, int remote_rto);

//函数说明：速度回调
//作用：通知上层当前harq的传输速度
//参数：addr：数据的远端地址，send_rate：数据的发送速度，recv_rate：数据的接收速度
//返回值：无
typedef void(*rate_function)(char* remote_ip, int remote_port, unsigned int send_rate, unsigned int recv_rate);

//so透出的函数类型定义
typedef int (*HARQ_START_CLIENT)(char *log, char *bind_ip, char *server_ip, int server_port, int timeout, bool delay, int delay_interval, bool encrypted, connect_function on_connect, receive_function on_recv, disconnect_function on_disconnect,  error_function on_error, rto_function on_rto, rate_function on_rate);
typedef int (*HARQ_SEND_BUFFER_HANDLE)(char *buffer, int size, int linker_handle);
typedef void (*HARQ_CLOSE_HANDLE)(int linker_handle);
typedef char *(*HARQ_VERSION)();
typedef void (*HARQ_END_SERVER)();

bool init_client(char* library_path);
int start_client(char* log, char* remote_ip, int remote_port, bool encrypted);
int send_buffer(char* data, int size, int handle);
void close_client(int handle);
char* version();







