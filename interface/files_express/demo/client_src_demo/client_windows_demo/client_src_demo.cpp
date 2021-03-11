// server_windows_demo.cpp : 定义控制台应用程序的入口点。
//

#include "stdafx.h"
#include "interface.h"

#include <string>
#include <stdio.h>
#include <stdarg.h>
#include <memory>
#include <time.h>

int main()
{
	std::vector<std::string> remote_array;
	remote_array.push_back("10.10.50.136");
	std::string log = "log";
	std::string session = "123456";

	std::vector<std::string> send_file_array;
	send_file_array.push_back("C:/radstudio10_1_upd2_esd.iso");

	int checkPostion = 0;
	while (1)
	{
		checkPostion++;
		::Sleep(1000);
	}
	return 0;
}

