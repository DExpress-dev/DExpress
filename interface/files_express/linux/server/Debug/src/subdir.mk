################################################################################
# Automatically-generated file. Do not edit!
################################################################################

# Add inputs and outputs from these tool invocations to the build variables 
CPP_SRCS += \
../src/express_server.cpp 

OBJS += \
./src/express_server.o 

CPP_DEPS += \
./src/express_server.d 


# Each subdirectory must supply rules for building sources it contributes
src/express_server.o: ../src/express_server.cpp
	@echo 'Building file: $<'
	@echo 'Invoking: GCC C++ Compiler'
	g++ -fsigned-char -fPIC -lstdc++ -std=c++11 -std=c++0x -I/mnt/hgfs/linux/project/CPlus/public/header -I/mnt/hgfs/public/files_express/src/header -I/mnt/hgfs/public/rudp_lib/src -I/mnt/hgfs/public/files_express/linux/server/business -I/mnt/hgfs/public/rudp/header/include -I/mnt/hgfs/public/files_express/src/header -O0 -g3 -Wall -c -fmessage-length=0 -MMD -MP -MF"$(@:%.o=%.d)" -MT"src/express_server.d" -o "$@" "$<"
	@echo 'Finished building: $<'
	@echo ' '


