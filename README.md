# mypan

一个百度网盘命令行客户端。

# 安全关切

关于参与方：客户端只与百度平台进行交互，再无第三方。授权结束后，访问凭证仅存储在客户端本地。

关于隔离：客户端使用百度网盘开放API
 - 只能上传到网盘应用特定的子目录，如`/apps/mypan`，其余动作如删除、重命令、列出、下载动作则无此限制
 - 参数中指定网盘目录时，若指定的是相对目录，则是相对应用特定的子目录，如`/apps/mypan`

# 授权

执行以下命令后，根据提示，打开网页，输入用户码授权即可

	mypan auth

# 命令

DATE: 2023/10/03
```
NAME:
   mypan - A baidu netdisk client

USAGE:
   mypan [global options] command [command options] [arguments...]

AUTHOR:
   Yousong Zhou <yszhou4tech@gmail.com>

COMMANDS:
   auth            
   quota           
   uinfo           
   ls, list        
   lsa, listall    
   stat            
   rm, remove      
   up, upload      
   syncup          
   down, download  
   syncdown        
   rename          
   mv, move        
   cp, copy        
   help, h         Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --appid value       (default: 40079350) [$MYPAN_APPID]
   --appkey value      (default: "4uwf4wql9Gtg3Dr79r6sKRgrac4M9uc1") [$MYPAN_APPKEY]
   --secretkey value   (default: "1mBQ9NOpW33EjLcYGzWQxTGUSNteZSfX") [$MYPAN_SECRETKEY]
   --appbasedir value  (default: "/apps/mypan") [$MYPAN_APPBASEDIR]
   --rundir value      (default: "/home/yousong/.mypan") [$MYPAN_RUNDIR]
   --timeout value     (default: 0s)
   --noprogress        (default: false)
   --format value      allowed values are json, table (default: "json")
   --alsologtostderr   log to standard error as well as files (default: false)
   --log_backtrace_at  when logging hits line file:N, emit a stack trace
   --log_dir           If non-empty, write log files in this directory
   --log_link          If non-empty, add symbolic links in this directory to the log files
   --logbuflevel       Buffer log messages logged at this level or lower (-1 means don't buffer; 0 means buffer INFO only; ...). Has limited applicability on non-prod platforms. (default: 0)
   --logtostderr       log to standard error instead of files (default: false)
   --stderrthreshold   logs at or above this threshold go to stderr (default: 2)
   -v                  log level for V logs (default: 0)
   --vmodule           comma-separated list of pattern=N settings for file-filtered logging
   --help, -h          show help
```

# 支持

![donation-wechat](https://github.com/yousong/mypan/assets/4948057/1c9a2878-cc65-4e40-99d8-a0b5d91c7253)
![donation-alipay](https://github.com/yousong/mypan/assets/4948057/990f65c9-d543-46ee-8e68-315e75037d8b)
