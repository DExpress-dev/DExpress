# 什么是livego<br/>     
livego是基于golang开发的rtmp服务器<br/>     
<br/>     
# 为什么基于golang
*  ## golang在语言基本支持多核CPU均衡使用，支持海量轻量级线程，提高其并发量<br/>   
   当前开源的缺陷：
   - srs只能运行在一个单核下，如果需要多核运行，只能启动多个srs监听不同的端口来提高并发量；<br/>   
   - ngx-rtmp启动多进程后，报文在多个进程内转发，需要二次开发，否则静态推送到多个子进程，效能消耗大；<br/>   
   golang在语言级别解决了上面多进程并发的问题。
*  ## 二次开发简洁快速<br/>   
   golang的开发效率远远高过C/C++

# livego支持哪些特性<br/>     
*  rtmp 推流，拉流
*  支持hls观看
*  支持http-flv观看
*  支持gop-cache缓存
*  静态relay支持：支持静态推流，拉流
*  统计信息支持：支持http在线查看流状态

## rtmp配置指引
livego的rtmp配置是基于json格式，简单好用。<br/> 
{<br/> 
    "listen": 1935,<br/> 
    "hls": "enable",<br/> 
    "servers":[<br/> 
        {<br/> 
        "servername":"live"<br/> 
        }<br/> 
    ]<br/> 
}<br/> 

## rtmp配置示例

{
"listen": 1935,     //rtmp监听端口
"notifyurl":""，   //暂时不配置
"hls": "enable",  //是否开启 hls
"hlsport" : 8090,// hls的拉流端口
"httpflv" : "enable",//是否开启fiv 
"flvport" : 8011,  // flv的拉流端口
"httpoper": "enable", //是否 可以接受 外界的http请求
"operport": 8070,   // 监听外界请求的端口
"engineEnable":"enable", //是否启动切片机
"engine":                   //切片机配置信息
{
    "ffmpeg": "/opt/segmenter",  //切片机程序所在目录
    "vcodec":"copy",
    "acodec":"copy",
    "extra_conf": "/opt/config.conf", // 切片机的配置文件
    "output": "/data/channellist/channel", //生成回放的路径
    "trans_user": ""      //可以不设置 
},
"servers":[
{
"servername":"live"   //服务名称
}
]
}
如上配置，表明:
rtmp监听1935端口
hls使能，并且监听8090端口
httpflv使能，并且监听8011端口
http 操作控制使能，并且监听8070端口

举例：
使用ffmpeg推流:
ffmpeg -re -i test.flv -c copy -f flv rtmp://127.0.0.1:1935/live/stream 
使用ffplay观看
rtmp观看方式: ffplay rtmp://127.0.0.1:1935/live/stream 
hls观看方式: ffplay http://127.0.0.1:8090/live/stream.m3u8 
http-flv观看方式: ffplay http://127.0.0.1:8011/live/stream.flv

使用切片机时需要配置：
"engineEnable":"enable", //是否启动切片机
不使用切片机时需要配置
"engineEnable":"", //是否启动切片机
张帆只需要 curl -i  "http://127.0.0.1:8070/getrtmplist"  即可得到当前正在推流的rtmp流。
注意：  8070 是 livego的http监听端口 的 operport端口




