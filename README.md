# chat_test
简单聊天服务器
## 概览
* client tcp 客户端
  * 编译：在client目录 go build
  * 通过标准输入发送指令
    * 顺序：login -> join -> chat
    * login [roleid] [name]
      * 例子：登录角色1，名字huhu：login 1 huhu
    * join [聊天室编号]
      * 例子：加入100号聊天室：join 100
    * chat [聊天内容]
      * 例子：chat 我是聊天内容
      * gm：chat /gm命令 参数
        * 例子：chat /stats 1 查看角色1的信息
        * chat /popular 查看10分钟内次数最多的单词，同一句里面出现多次也算多次
  * 返回信息内容：
    * 直接打印的消息结果
    * 例子：比如收到聊天记录 2022/05/15 12:31:30 received chat history back msg &{[****o how do you do ****o 竟然都要错]} 是直接打印的数组，没有单独区分了
  * 一些细节说明：
    * 词频度统计时，“词”是指除掉*之外，用空格分开的字母组合
* chat tcp 服务器
  * 编译：在chat目录 go build
  * 启动时加-p 指定监听的端口，默认8888
  * 服务器的核心代码：gen_routine 是在公司内自己独立实现的，有完整的单元测试，覆盖率应该是在90%以上
  * 敏感词过滤代码是直接网上找的
  * 敏感词排行是完全独立写的
  * 很多因为时间仓促或者说不是真正的商用项目，导致代码有比较将就的地方
  * 看代码质量还是主要看 gen_routine 比较客观
  * 另外是在家里电脑写的，只跑通了windows版，mac或者linux没有自测