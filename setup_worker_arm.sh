#!/bin/bash -v


echo "--- Installing Containerd ---"

apt-get update

apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

apt-get install -y containerd.io cgroup-tools


echo "--- Installing xdp/ebpf libraries ---"
apt-get install -y clang llvm libelf-dev libpcap-dev build-essential linux-tools-common linux-tools-generic tcpdump m4 net-tools bridge-utils
apt-get install -y linux-tools-$(uname -r)
apt-get install -y linux-headers-$(uname -r)

#xdp tools
echo "--- Installing xdp-tools ---"
cd ..
git clone https://github.com/xdp-project/xdp-tools
cd xdp-tools
make
make install
cd ..

echo "--- Getting XDP tutorial for basic setup script ---"
#xdp tutorial setup scripts for testing, comment out if required
git clone https://github.com/xdp-project/xdp-tutorial

echo "--- Installing golang ---"
#golang for compiling, just in case
wget https://go.dev/dl/go1.19.4.linux-arm64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.19.4.linux-arm64.tar.gz
export PATH=$PATH:/usr/local/go/bin