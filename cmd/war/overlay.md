```bash
mkdir -p /tmp/dir1
mkdir -p /tmp/dir2
echo hello > /tmp/dir1/1.txt
echo world > /tmp/dir2/2.txt
mkdir /tmp/upperdir
mkdir /tmp/workdir
mkdir merged
sudo mount -t overlay overlay -o lowerdir=/tmp/dir1:/tmp/dir2,upperdir=/tmp/upperdir,workdir=/tmp/workdir merged

sudo mount -t overlay overlay -o lowerdir=/root/workspace/overlayfs-test/dir1,upperdir=/root/workspace/overlayfs-test/upperdir,workdir=/root/workspace/overlayfs-test/workdir /root/workspace/overlayfs-test/merged
```

```
sudo umount merged
```bash