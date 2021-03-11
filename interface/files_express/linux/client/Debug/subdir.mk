################################################################################
# Automatically-generated file. Do not edit!
################################################################################

# Add inputs and outputs from these tool invocations to the build variables 
CPP_SRCS += \
/mnt/hgfs/linux/project/CPlus/public/header/linux/file_linux.cpp \
/mnt/hgfs/linux/project/CPlus/public/header/linux/path_linux.cpp \
/mnt/hgfs/linux/project/CPlus/public/header/path.cpp \
/mnt/hgfs/linux/project/CPlus/public/header/public.cpp \
/mnt/hgfs/linux/project/CPlus/public/header/write_log.cpp \
/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/linux/delay_linux.cpp \
/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/rudp_timer.cpp \
/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/rudp_public.cpp \

OBJS += \
/mnt/hgfs/linux/project/CPlus/public/header/linux/file_linux.o \
/mnt/hgfs/linux/project/CPlus/public/header/linux/path_linux.o \
/mnt/hgfs/linux/project/CPlus/public/header/path.o \
/mnt/hgfs/linux/project/CPlus/public/header/public.o \
/mnt/hgfs/linux/project/CPlus/public/header/write_log.o \
/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/linux/delay_linux.o \
/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/rudp_timer.o \
/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/rudp_public.o \

CPP_DEPS += \
/mnt/hgfs/linux/project/CPlus/public/header/linux/file_linux.d \
/mnt/hgfs/linux/project/CPlus/public/header/linux/path_linux.d \
/mnt/hgfs/linux/project/CPlus/public/header/path.d \
/mnt/hgfs/linux/project/CPlus/public/header/public.d \
/mnt/hgfs/linux/project/CPlus/public/header/write_log.d \
/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/linux/delay_linux.d \
/mnt/hgfs/linux/project/CPlus/public/rudp/include/include/rudp_timer.d \
/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/rudp_public.d \

# Each subdirectory must supply rules for building sources it contributes

# RUDP Header
/mnt/hgfs/linux/project/CPlus/public/header/%.o: /mnt/hgfs/linux/project/CPlus/public/header/%.cpp
	@echo 'Building file: $<'
	@echo 'Invoking: GCC C++ Compiler'
	g++ -fsigned-char -fPIC -lstdc++ -std=c++11 \
									-I/mnt/hgfs/linux/project/CPlus/public/header \
									-I/mnt/hgfs/linux/project/CPlus/public/header/linux \
							-O0 -g3 -Wall -c -fmessage-length=0 -MMD -MP -MF"$(@:%.o=%.d)" -MT"$(@:%.o=%.d)" -o "$@" "$<"
	@echo 'Finished building: $<'
	@echo ' '
	
# RUDP Include
/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/%.o: /mnt/hgfs/linux/project/CPlus/public/rudp/header/include/%.cpp
	@echo 'Building file: $<'
	@echo 'Invoking: GCC C++ Compiler'
	g++ -fsigned-char -fPIC -lstdc++ -std=c++11 \
									-I/mnt/hgfs/linux/project/CPlus/public/rudp/header/include \
									-I/mnt/hgfs/linux/project/CPlus/public/rudp/header/include/linux \
							-O0 -g3 -Wall -c -fmessage-length=0 -MMD -MP -MF"$(@:%.o=%.d)" -MT"$(@:%.o=%.d)" -o "$@" "$<"
	@echo 'Finished building: $<'
	@echo ' '
	
