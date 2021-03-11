#!/bin/sh
soPath="../so"
fileName="lib_client.so"

if [ ! -f "$soPath" ];then 
	mkdir "$soPath"
fi

cd Debug
make clean
make all
cd ..

if [ ! -f "$fileName" ];then
	mv Debug/"$fileName" "$soPath"
fi

cd Debug
make clean

echo "****Build Magpie Client Library Success****";
