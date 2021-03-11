################################################################################
# Automatically-generated file. Do not edit!
################################################################################

# Add inputs and outputs from these tool invocations to the build variables 
CPP_SRCS += \
../business/main_thread.cpp \
../business/udp_manager.cpp 

OBJS += \
./business/main_thread.o \
./business/udp_manager.o 

CPP_DEPS += \
./business/main_thread.d \
./business/udp_manager.d 


# Each subdirectory must supply rules for building sources it contributes
business/main_thread.o: ../business/main_thread.cpp
	@echo 'Building file: $<'
	@echo 'Invoking: GCC C++ Compiler'
	g++ -fsigned-char -fPIC -lstdc++ -std=c++11 -std=c++0x -I/mnt/hgfs/linux/project/CPlus/public/header -I/mnt/hgfs/public/files_express/src/header -I/mnt/hgfs/public/rudp_lib/src -I/mnt/hgfs/public/files_express/linux/server/business -I/mnt/hgfs/public/rudp/header/include -I/mnt/hgfs/public/files_express/src/header -O0 -g3 -Wall -c -fmessage-length=0 -MMD -MP -MF"$(@:%.o=%.d)" -MT"business/main_thread.d" -o "$@" "$<"
	@echo 'Finished building: $<'
	@echo ' '

business/udp_manager.o: ../business/udp_manager.cpp
	@echo 'Building file: $<'
	@echo 'Invoking: GCC C++ Compiler'
	g++ -fsigned-char -fPIC -lstdc++ -std=c++11 -std=c++0x -I/mnt/hgfs/linux/project/CPlus/public/header -I/mnt/hgfs/public/files_express/src/header -I/mnt/hgfs/public/rudp_lib/src -I/mnt/hgfs/public/files_express/linux/server/business -I/mnt/hgfs/public/rudp/header/include -I/mnt/hgfs/public/files_express/src/header -O0 -g3 -Wall -c -fmessage-length=0 -MMD -MP -MF"$(@:%.o=%.d)" -MT"business/udp_manager.d" -o "$@" "$<"
	@echo 'Finished building: $<'
	@echo ' '


