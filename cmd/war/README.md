# 关注如下事件

- 当持续写文件时, 收到 write event 的频率
- inotify 的监听目录个数
- inotify 的overflow?
- symlink ?

# 结论

- 移进入本目录, 相当于 create
- 移出本目录, 只会收到那个 entry 的 rename, 不知道目的地, 也不知道它的子文件如何
- 当 entry 被删除/移动时, fsnotify 会自动 remove 它的 watch, 因此我们不用再手动 remove (去看 Add 方法的注释), 而 windows
  在 rename 时不会自动
  watcher.remove 看注释.
- mv 2.go 1.go 则 1.go 的 inode 会是 2.go 的, 2.go 产生一个 RENAME, 1.go 产生一个 CREATE
- truncate -s 体现为 WRITE
- 当有目录移进来时候, 只会收到那个目录的事件, 子的文件无; 因此此时需要 dfs 一下建立关系
- 当目录被移动出监控范围时, 会收到 remove

# fsnotify 准则

1. 只监听目录, 不监听文件, 可以简化很多处理
2. fsnotify 只能监听目录自身, 无法递归监听, 需要自己实现
3. 要特别注意 rename/move, 分进来和出去
    1. 当有其他目录 rename 进入监控目录, 需要对进入的目录进行递归处理, 这个处理比较简单
    2. 当有目录从监控目录中移出去, 需要手动删除它所有子孙目录的监听
4. 正常的 rm -rf dir/ 是从叶子节点开始删除的, 因为你不会错过事件

# 与 mount 相关

- inotify 不会阻碍 umount
- 但当一个目录被 umount 后, fsnotify 没有正确透传 (in_unmount 事件), 你的程序会收到 name=目录, Op==0 (在不魔改 fsnotify
  代码的情况下, stat
  一下这个路径, 如果文件已经不存在就删除它.
  你可以使用它op==0去做判断). 不过问题不大, 因为 unmount 目前只有 overlayfs 会发生, 而它是用于 MergedDir, 如果真的发生
  unmount 了,
  那么就是容器删除了. 久而久之自然就不会再请求它, 从而触发删除.
- inotify 没有 mount, 如果一个目录被 mount 了, 你是不知道的 (我们对此需求也不大)

# TODO
1. 支持注入环境变量
2. 支持 yaml, 支持将 run 直接写在 yaml 里
3. 支持对不同的路径的文件变化采用不同的响应
 